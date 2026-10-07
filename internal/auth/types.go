package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
)

var ErrDenied = errors.New("authentication denied")

type User struct {
	AvatarURL *string `json:"avatarUrl,omitempty"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
}

func (u User) Principal() access.Principal {
	return access.Principal{ID: u.ID, Name: u.Name, Role: u.Role}
}

type Flow struct {
	State          string
	Browser        string
	Verifier       string
	Nonce          string
	InvitationHash string
	Expires        time.Time
}
type Identity struct{ Issuer, Subject, Name string }
type Session struct {
	User     User
	CSRFHash string
	Expires  time.Time
}
type Invitation struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expiresAt"`
}

type Store interface {
	SaveFlow(context.Context, Flow) error
	TakeFlow(context.Context, string, string) (Flow, error)
	Login(context.Context, Identity, string, string, string) (User, error)
	FindSession(context.Context, string) (Session, error)
	Revoke(context.Context, string, string, bool) error
	CreateInvitation(context.Context, string, string) (Invitation, error)
	Disable(context.Context, string, string) error
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure random unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func csrfToken(session string) string { return hash("grace-csrf:" + session) }
