package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) SaveFlow(ctx context.Context, f Flow) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO auth_login_flows
		(state_hash,browser_hash,verifier,nonce,invitation_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6)`,
		hash(f.State), hash(f.Browser), f.Verifier, f.Nonce, f.InvitationHash, f.Expires)
	return err
}
func (r *Repository) TakeFlow(ctx context.Context, state, browser string) (Flow, error) {
	var f Flow
	err := r.pool.QueryRow(ctx, `DELETE FROM auth_login_flows WHERE state_hash=$1 AND browser_hash=$2 AND expires_at > now()
		RETURNING verifier,nonce,invitation_hash,expires_at`, hash(state), hash(browser)).Scan(&f.Verifier, &f.Nonce, &f.InvitationHash, &f.Expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrDenied
	}
	return f, err
}
func (r *Repository) Login(ctx context.Context, identity Identity, invitationHash, session, csrf string) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialize binding an identity across simultaneous logins/invitations.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", identity.Issuer+":"+identity.Subject); err != nil {
		return User{}, err
	}
	var user User
	var disabled bool
	err = tx.QueryRow(ctx, `SELECT u.id::text,u.display_name,u.role,u.disabled FROM auth_identities i
		JOIN users u ON u.id=i.user_id WHERE i.issuer=$1 AND i.subject=$2 FOR UPDATE OF u`, identity.Issuer, identity.Subject).Scan(&user.ID, &user.Name, &user.Role, &disabled)
	if err == nil && invitationHash != "" {
		return User{}, ErrDenied
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if invitationHash == "" {
			return User{}, ErrDenied
		}
		err = tx.QueryRow(ctx, `SELECT u.id::text,u.display_name,u.role,u.disabled FROM auth_invitations i
			JOIN users u ON u.id=i.user_id WHERE i.token_hash=$1 AND i.used_at IS NULL AND i.expires_at>now() AND (i.bound_email IS NULL OR ($2=$3 AND i.bound_email=$4))
			FOR UPDATE OF i,u`, invitationHash, identity.Issuer, EmailIssuer, identity.Subject).Scan(&user.ID, &user.Name, &user.Role, &disabled)
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrDenied
		}
		if err != nil {
			return User{}, err
		}
		if disabled {
			return User{}, ErrDenied
		}
		_, err = tx.Exec(ctx, `INSERT INTO auth_identities(issuer,subject,user_id) VALUES($1,$2,$3)`, identity.Issuer, identity.Subject, user.ID)
		if err != nil {
			return User{}, ErrDenied
		}
		if _, err = tx.Exec(ctx, `UPDATE auth_invitations SET used_at=now() WHERE token_hash=$1`, invitationHash); err != nil {
			return User{}, err
		}
	}
	if err != nil {
		return User{}, err
	}
	if disabled {
		return User{}, ErrDenied
	}
	_, err = tx.Exec(ctx, `INSERT INTO auth_sessions(token_hash,csrf_hash,user_id,expires_at) VALUES($1,$2,$3,now()+interval '7 days')`, hash(session), hash(csrf), user.ID)
	if err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}
func (r *Repository) FindSession(ctx context.Context, token string) (Session, error) {
	var s Session
	err := r.pool.QueryRow(ctx, `SELECT u.id::text,u.display_name,u.role,s.csrf_hash,s.expires_at FROM auth_sessions s
		JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND NOT u.disabled`, hash(token)).Scan(&s.User.ID, &s.User.Name, &s.User.Role, &s.CSRFHash, &s.Expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrDenied
	}
	return s, err
}
func (r *Repository) Revoke(ctx context.Context, userID, token string, all bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE user_id=$1 AND ($2 OR token_hash=$3)`, userID, all, hash(token))
	return err
}
func (r *Repository) CreateInvitation(ctx context.Context, adminID, name string) (Invitation, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 160 {
		return Invitation{}, ErrDenied
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var allowed bool
	if err = tx.QueryRow(ctx, `SELECT role='admin' AND NOT disabled FROM users WHERE id=$1 FOR UPDATE`, adminID).Scan(&allowed); err != nil || !allowed {
		return Invitation{}, ErrDenied
	}
	var userID string
	if err = tx.QueryRow(ctx, `INSERT INTO users(display_name,role) VALUES($1,'member') RETURNING id::text`, name).Scan(&userID); err != nil {
		return Invitation{}, err
	}
	invite, err := insertInvitation(ctx, tx, userID)
	if err != nil {
		return invite, err
	}
	return invite, tx.Commit(ctx)
}
func insertInvitation(ctx context.Context, tx pgx.Tx, userID string) (Invitation, error) {
	i := Invitation{Token: randomToken(), Expires: time.Now().UTC().Add(24 * time.Hour)}
	_, err := tx.Exec(ctx, `INSERT INTO auth_invitations(token_hash,user_id,expires_at) VALUES($1,$2,$3)`, hash(i.Token), userID, i.Expires)
	return i, err
}

// OwnerInvitation is an operator-only operation, never exposed over HTTP.
func (r *Repository) OwnerInvitation(ctx context.Context) (Invitation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Invitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id=$1 FOR UPDATE`, access.BootstrapOwnerID).Scan(&id); err != nil {
		return Invitation{}, err
	}
	var linked bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_identities WHERE user_id=$1)`, id).Scan(&linked); err != nil {
		return Invitation{}, err
	}
	if linked {
		return Invitation{}, fmt.Errorf("owner identity already linked")
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_invitations SET used_at=now() WHERE user_id=$1 AND used_at IS NULL`, id); err != nil {
		return Invitation{}, err
	}
	i, err := insertInvitation(ctx, tx, id)
	if err != nil {
		return i, err
	}
	return i, tx.Commit(ctx)
}
func (r *Repository) Disable(ctx context.Context, adminID, userID string) error {
	if userID == adminID || userID == access.BootstrapOwnerID {
		return ErrDenied
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var allowed bool
	if err = tx.QueryRow(ctx, `SELECT role='admin' AND NOT disabled FROM users WHERE id=$1 FOR UPDATE`, adminID).Scan(&allowed); err != nil || !allowed {
		return ErrDenied
	}
	result, err := tx.Exec(ctx, `UPDATE users SET disabled=true WHERE id::text=$1`, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrDenied
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE user_id::text=$1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
