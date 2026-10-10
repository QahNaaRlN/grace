package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"unicode/utf8"
)

type PasswordToken struct {
	TokenHash, Purpose, UserID, Email, InvitationHash, PasswordHash string
	Revision                                                        int64
}

func (r *Repository) ReservePasswordAttempt(ctx context.Context, email, source string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	minute := time.Now().UTC().Truncate(time.Minute)
	for _, limit := range []struct {
		key string
		max int
	}{{"password-source:" + source, 10}, {"password-email:" + email, 5}, {"password-global", 100}} {
		var count int
		err = tx.QueryRow(ctx, `INSERT INTO auth_email_rate_windows(key_hash,expires_at,requests) VALUES($1,$2,1) ON CONFLICT(key_hash) DO UPDATE SET requests=auth_email_rate_windows.requests+1 RETURNING requests`, hash(limit.key+minute.Format(time.RFC3339)), minute.Add(time.Minute)).Scan(&count)
		if err != nil {
			return err
		}
		if count > limit.max {
			return ErrRateLimited
		}
	}
	_, err = tx.Exec(ctx, `DELETE FROM auth_email_rate_windows WHERE expires_at<now()-interval '2 days'`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) PasswordIdentity(ctx context.Context, email string) (User, string, int64, error) {
	var user User
	var encoded string
	var revision int64
	err := r.pool.QueryRow(ctx, `SELECT u.id::text,u.display_name,u.role,COALESCE(c.password_hash,''),COALESCE(c.revision,0)
 FROM users u JOIN auth_identities i ON i.user_id=u.id AND i.issuer=$1 AND i.subject=$2
 LEFT JOIN auth_password_credentials c ON c.user_id=u.id WHERE NOT u.disabled`, EmailIssuer, email).Scan(&user.ID, &user.Name, &user.Role, &encoded, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrDenied
	}
	return user, encoded, revision, err
}
func (r *Repository) CreateBoundInvitation(ctx context.Context, admin, name, email string) (Invitation, error) {
	name = strings.TrimSpace(name)
	normalized, validationErr := normalizeEmail(email)
	if validationErr != nil || normalized != email || name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 160 {
		return Invitation{}, ErrDenied
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var allowed, existing bool
	if err = tx.QueryRow(ctx, `SELECT role='admin' AND NOT disabled FROM users WHERE id=$1 FOR UPDATE`, admin).Scan(&allowed); err != nil || !allowed {
		return Invitation{}, ErrDenied
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_identities WHERE issuer=$1 AND subject=$2)`, EmailIssuer, email).Scan(&existing); err != nil {
		return Invitation{}, err
	}
	if existing {
		return Invitation{}, ErrDenied
	}
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO users(display_name,role) VALUES($1,'member') RETURNING id::text`, name).Scan(&id); err != nil {
		return Invitation{}, err
	}
	invite, err := insertInvitation(ctx, tx, id)
	if err != nil {
		return invite, err
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_invitations SET bound_email=$2 WHERE token_hash=$1`, hash(invite.Token), email); err != nil {
		return Invitation{}, err
	}
	return invite, tx.Commit(ctx)
}
func (r *Repository) RegistrationTarget(ctx context.Context, email, invitation string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT u.id::text FROM auth_invitations i JOIN users u ON u.id=i.user_id
 WHERE i.token_hash=$1 AND i.bound_email=$2 AND i.used_at IS NULL AND i.expires_at>now() AND NOT u.disabled
 AND NOT EXISTS(SELECT 1 FROM auth_identities WHERE issuer=$3 AND subject=$2)
 AND NOT EXISTS(SELECT 1 FROM auth_identities WHERE user_id=u.id AND issuer=$3)
 AND NOT EXISTS(SELECT 1 FROM auth_password_credentials WHERE user_id=u.id)`, hash(invitation), email, EmailIssuer).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrDenied
	}
	return id, err
}
func (r *Repository) SavePasswordToken(ctx context.Context, token PasswordToken) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO auth_password_tokens(token_hash,purpose,user_id,email,invitation_hash,password_hash,revision,expires_at)
 VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,now()+interval '30 minutes')`, token.TokenHash, token.Purpose, token.UserID, token.Email, token.InvitationHash, token.PasswordHash, token.Revision)
	if err == nil {
		_, err = r.pool.Exec(ctx, `DELETE FROM auth_password_tokens WHERE expires_at<now()-interval '1 hour'`)
	}
	return err
}
func (r *Repository) CancelPasswordToken(ctx context.Context, tokenHash string) {
	_, _ = r.pool.Exec(ctx, `DELETE FROM auth_password_tokens WHERE token_hash=$1 AND used_at IS NULL`, tokenHash)
}
func (r *Repository) FinishPasswordToken(ctx context.Context, raw, purpose, newHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var f PasswordToken
	// First lookup does not consume; acquire user lock before token lock for reset concurrency.
	err = tx.QueryRow(ctx, `SELECT user_id::text FROM auth_password_tokens WHERE token_hash=$1 AND purpose=$2 AND used_at IS NULL AND expires_at>now()`, hash(raw), purpose).Scan(&f.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDenied
	}
	if err != nil {
		return err
	}
	var disabled bool
	if err = tx.QueryRow(ctx, `SELECT disabled FROM users WHERE id=$1 FOR UPDATE`, f.UserID).Scan(&disabled); err != nil || disabled {
		return ErrDenied
	}
	err = tx.QueryRow(ctx, `SELECT email,COALESCE(invitation_hash,''),COALESCE(password_hash,''),revision FROM auth_password_tokens
 WHERE token_hash=$1 AND purpose=$2 AND used_at IS NULL AND expires_at>now() FOR UPDATE`, hash(raw), purpose).Scan(&f.Email, &f.InvitationHash, &f.PasswordHash, &f.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDenied
	}
	if err != nil {
		return err
	}
	if purpose == "register" {
		var id string
		err = tx.QueryRow(ctx, `UPDATE auth_invitations SET used_at=now() WHERE token_hash=$1 AND user_id=$2 AND bound_email=$3 AND used_at IS NULL AND expires_at>now() RETURNING user_id::text`, f.InvitationHash, f.UserID, f.Email).Scan(&id)
		if err != nil {
			return ErrDenied
		}
		if _, err = tx.Exec(ctx, `INSERT INTO auth_identities(issuer,subject,user_id) VALUES($1,$2,$3)`, EmailIssuer, f.Email, f.UserID); err != nil {
			return ErrDenied
		}
		newHash = f.PasswordHash
	} else {
		var email string
		var revision int64
		err = tx.QueryRow(ctx, `SELECT i.subject,COALESCE(c.revision,0) FROM auth_identities i LEFT JOIN auth_password_credentials c ON c.user_id=i.user_id
   WHERE i.user_id=$1 AND i.issuer=$2`, f.UserID, EmailIssuer).Scan(&email, &revision)
		if err != nil || email != f.Email || revision != f.Revision {
			return ErrDenied
		}
	}
	if purpose == "register" {
		if _, err = tx.Exec(ctx, `INSERT INTO auth_password_credentials(user_id,password_hash) VALUES($1,$2)`, f.UserID, newHash); err != nil {
			return ErrDenied
		}
	} else {
		if _, err = tx.Exec(ctx, `INSERT INTO auth_password_credentials(user_id,password_hash) VALUES($1,$2)
   ON CONFLICT(user_id) DO UPDATE SET password_hash=EXCLUDED.password_hash,revision=auth_password_credentials.revision+1,updated_at=now()`, f.UserID, newHash); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_password_tokens SET used_at=now() WHERE user_id=$1 AND used_at IS NULL`, f.UserID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, f.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) PasswordSession(ctx context.Context, email, expectedHash, session string) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var user User
	var encoded string
	err = tx.QueryRow(ctx, `SELECT u.id::text,u.display_name,u.role,c.password_hash FROM users u JOIN auth_identities i ON i.user_id=u.id
 JOIN auth_password_credentials c ON c.user_id=u.id WHERE i.issuer=$1 AND i.subject=$2 AND NOT u.disabled FOR UPDATE OF u`, EmailIssuer, email).Scan(&user.ID, &user.Name, &user.Role, &encoded)
	if err != nil || encoded != expectedHash {
		return User{}, ErrDenied
	}
	if _, err = tx.Exec(ctx, `INSERT INTO auth_sessions(token_hash,csrf_hash,user_id,expires_at) VALUES($1,$2,$3,now()+interval '7 days')`, hash(session), hash(csrfToken(session)), user.ID); err != nil {
		return User{}, err
	}
	return user, tx.Commit(ctx)
}
