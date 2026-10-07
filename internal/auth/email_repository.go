package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"unicode/utf8"
)

const EmailIssuer = "grace:verified-email"

var ErrRateLimited = errors.New("email rate limited")

type EmailFlow struct {
	BrowserHash, CodeHash, Email, Purpose, UserID, InvitationHash string
	Expires                                                       time.Time
}
type EmailStore interface {
	ReserveEmail(context.Context, string, string, int) error
	SaveEmailFlow(context.Context, EmailFlow) error
	TakeEmailFlow(context.Context, string, string, string, string) (EmailFlow, error)
	Login(context.Context, Identity, string, string, string) (User, error)
	LinkEmail(context.Context, string, string, string) error
}

func (r *Repository) ReserveEmail(ctx context.Context, email, source string, budget int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := time.Now().UTC()
	hour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	minute := hour.Add(time.Duration(now.Minute()) * time.Minute)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	limits := []struct {
		key string
		end time.Time
		max int
	}{
		{"month:" + month.Format("200601"), month.AddDate(0, 1, 0), budget},
		{"minute:" + minute.Format(time.RFC3339), minute.Add(time.Minute), 30},
		{"source:" + source + ":" + minute.Format(time.RFC3339), minute.Add(time.Minute), 20},
		{"email-hour:" + email + ":" + hour.Format(time.RFC3339), hour.Add(time.Hour), 5},
		{"email-minute:" + email + ":" + minute.Format(time.RFC3339), minute.Add(time.Minute), 1},
	}
	for _, limit := range limits {
		var count int
		err = tx.QueryRow(ctx, `INSERT INTO auth_email_rate_windows(key_hash,expires_at,requests) VALUES($1,$2,1) ON CONFLICT(key_hash) DO UPDATE SET requests=auth_email_rate_windows.requests+1 RETURNING requests`, hash(limit.key), limit.end).Scan(&count)
		if err != nil {
			return err
		}
		if count > limit.max {
			return ErrRateLimited
		}
	}
	var reserved string
	err = tx.QueryRow(ctx, `INSERT INTO auth_email_rate_windows(key_hash,expires_at,requests) VALUES($1,now()+interval '60 seconds',1) ON CONFLICT(key_hash) DO UPDATE SET expires_at=EXCLUDED.expires_at,requests=1 WHERE auth_email_rate_windows.expires_at<=now() RETURNING key_hash`, hash("cooldown:"+email)).Scan(&reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRateLimited
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_email_rate_windows WHERE expires_at < now()-interval '2 days'; DELETE FROM auth_email_challenges WHERE expires_at < now()-interval '1 hour'`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) SaveEmailFlow(ctx context.Context, f EmailFlow) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO auth_email_challenges(browser_hash,code_hash,email,purpose,user_id,invitation_hash,expires_at) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7)`, f.BrowserHash, f.CodeHash, f.Email, f.Purpose, f.UserID, f.InvitationHash, f.Expires)
	return err
}
func (r *Repository) TakeEmailFlow(ctx context.Context, browserHash, codeHash, purpose, userID string) (EmailFlow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return EmailFlow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var f EmailFlow
	var attempts int
	err = tx.QueryRow(ctx, `SELECT browser_hash,code_hash,email,purpose,COALESCE(user_id::text,''),invitation_hash,expires_at,attempts FROM auth_email_challenges WHERE browser_hash=$1 FOR UPDATE`, browserHash).Scan(&f.BrowserHash, &f.CodeHash, &f.Email, &f.Purpose, &f.UserID, &f.InvitationHash, &f.Expires, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrDenied
	}
	if err != nil {
		return f, err
	}
	if f.Purpose != purpose || f.UserID != userID || attempts >= 5 || !time.Now().Before(f.Expires) {
		return f, ErrDenied
	}
	if subtle.ConstantTimeCompare([]byte(f.CodeHash), []byte(codeHash)) != 1 {
		if _, err = tx.Exec(ctx, `UPDATE auth_email_challenges SET attempts=attempts+1 WHERE browser_hash=$1`, browserHash); err != nil {
			return f, err
		}
		if err = tx.Commit(ctx); err != nil {
			return f, err
		}
		return f, ErrDenied
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_email_challenges WHERE browser_hash=$1`, browserHash); err != nil {
		return f, err
	}
	return f, tx.Commit(ctx)
}
func (r *Repository) LinkEmail(ctx context.Context, userID, email, currentToken string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, EmailIssuer+":"+email); err != nil {
		return err
	}
	var fresh bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_sessions WHERE user_id=$1 AND token_hash=$2 AND created_at>now()-interval '15 minutes' AND revoked_at IS NULL AND expires_at>now())`, userID, hash(currentToken)).Scan(&fresh); err != nil || !fresh {
		return ErrDenied
	}
	var disabled bool
	if err = tx.QueryRow(ctx, `SELECT disabled FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&disabled); err != nil || disabled {
		return ErrDenied
	}
	var claimed string
	err = tx.QueryRow(ctx, `SELECT user_id::text FROM auth_identities WHERE issuer=$1 AND subject=$2`, EmailIssuer, email).Scan(&claimed)
	if err == nil && claimed != userID {
		return ErrDenied
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO auth_identities(issuer,subject,user_id) VALUES($1,$2,$3) ON CONFLICT(user_id,issuer) DO UPDATE SET subject=EXCLUDED.subject`, EmailIssuer, email, userID); err != nil {
		return ErrDenied
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE user_id=$1 AND token_hash<>$2`, userID, hash(currentToken)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EmailMigrationInvitation is operator-only and never blindly verifies an email.
func (r *Repository) EmailMigrationInvitation(ctx context.Context, userID, email string) (Invitation, error) {
	var err error
	email, err = normalizeEmail(email)
	if err != nil {
		return Invitation{}, ErrDenied
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var disabled bool
	if err = tx.QueryRow(ctx, `SELECT disabled FROM users WHERE id::text=$1 FOR UPDATE`, userID).Scan(&disabled); err != nil || disabled {
		return Invitation{}, ErrDenied
	}
	var linked bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_identities WHERE user_id::text=$1 AND issuer=$2)`, userID, EmailIssuer).Scan(&linked); err != nil {
		return Invitation{}, err
	}
	if linked {
		return Invitation{}, fmt.Errorf("email already linked")
	}
	var claimed bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_identities WHERE issuer=$1 AND subject=$2)`, EmailIssuer, email).Scan(&claimed); err != nil || claimed {
		return Invitation{}, ErrDenied
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_invitations SET used_at=now() WHERE user_id::text=$1 AND used_at IS NULL`, userID); err != nil {
		return Invitation{}, err
	}
	i := Invitation{Token: randomToken(), Expires: time.Now().UTC().Add(24 * time.Hour)}
	if _, err = tx.Exec(ctx, `INSERT INTO auth_invitations(token_hash,user_id,expires_at,bound_email) VALUES($1,$2,$3,$4)`, hash(i.Token), userID, i.Expires, email); err != nil {
		return Invitation{}, err
	}
	return i, tx.Commit(ctx)
}

type Profile struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	Email       *string `json:"email"`
	AvatarURL   *string `json:"avatarUrl"`
}
type ProfileNames struct {
	DisplayName string `json:"displayName"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
}

func (n *ProfileNames) valid() bool {
	n.DisplayName = strings.TrimSpace(n.DisplayName)
	n.FirstName = strings.TrimSpace(n.FirstName)
	n.LastName = strings.TrimSpace(n.LastName)
	return n.DisplayName != "" && utf8.RuneCountInString(n.DisplayName) <= 160 && utf8.RuneCountInString(n.FirstName) <= 80 && utf8.RuneCountInString(n.LastName) <= 80
}
func (r *Repository) Profile(ctx context.Context, id string) (Profile, error) {
	var p Profile
	var digest *string
	err := r.pool.QueryRow(ctx, `SELECT u.id::text,u.display_name,u.first_name,u.last_name,(SELECT subject FROM auth_identities WHERE user_id=u.id AND issuer=$2),CASE WHEN avatar_png IS NULL THEN NULL ELSE md5(avatar_png) END FROM users u WHERE u.id=$1 AND NOT disabled`, id, EmailIssuer).Scan(&p.ID, &p.DisplayName, &p.FirstName, &p.LastName, &p.Email, &digest)
	if digest != nil {
		url := "/api/auth/profile/avatar?v=" + *digest
		p.AvatarURL = &url
	}
	return p, err
}
func (r *Repository) UpdateProfile(ctx context.Context, id string, n ProfileNames) error {
	result, err := r.pool.Exec(ctx, `UPDATE users SET display_name=$2,first_name=$3,last_name=$4 WHERE id=$1 AND NOT disabled`, id, n.DisplayName, n.FirstName, n.LastName)
	if err == nil && result.RowsAffected() != 1 {
		return ErrDenied
	}
	return err
}
func (r *Repository) Avatar(ctx context.Context, id string) ([]byte, error) {
	var data []byte
	err := r.pool.QueryRow(ctx, `SELECT avatar_png FROM users WHERE id=$1 AND NOT disabled`, id).Scan(&data)
	return data, err
}
func (r *Repository) SetAvatar(ctx context.Context, id string, data []byte) error {
	result, err := r.pool.Exec(ctx, `UPDATE users SET avatar_png=$2 WHERE id=$1 AND NOT disabled`, id, data)
	if err == nil && result.RowsAffected() != 1 {
		return ErrDenied
	}
	return err
}
func (r *Repository) RecentSession(ctx context.Context, userID, token string) (bool, error) {
	var valid bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_sessions s JOIN users u ON u.id=s.user_id WHERE s.user_id::text=$1 AND s.token_hash=$2 AND s.created_at>now()-interval '15 minutes' AND s.revoked_at IS NULL AND s.expires_at>now() AND NOT u.disabled)`, userID, hash(token)).Scan(&valid)
	return valid, err
}
func (r *Repository) EmailMigrationHash(ctx context.Context, email string) (string, error) {
	var token string
	err := r.pool.QueryRow(ctx, `SELECT i.token_hash FROM auth_invitations i JOIN users u ON u.id=i.user_id WHERE i.bound_email=$1 AND i.used_at IS NULL AND i.expires_at>now() AND NOT u.disabled AND NOT EXISTS(SELECT 1 FROM auth_identities WHERE user_id=u.id AND issuer=$2) ORDER BY i.expires_at DESC LIMIT 1`, email, EmailIssuer).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return token, err
}
