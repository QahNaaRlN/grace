package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stubStore struct {
	session   Session
	token     string
	err       error
	flow      Flow
	flowTaken bool
	loggedIn  bool
	revoked   bool
}

func (s *stubStore) SaveFlow(_ context.Context, f Flow) error { s.flow = f; return s.err }
func (s *stubStore) TakeFlow(_ context.Context, state, browser string) (Flow, error) {
	if s.flowTaken || state != s.flow.State || browser != s.flow.Browser {
		return Flow{}, ErrDenied
	}
	s.flowTaken = true
	return s.flow, s.err
}
func (s *stubStore) Login(_ context.Context, _ Identity, _, token, csrf string) (User, error) {
	if s.err != nil {
		return User{}, s.err
	}
	s.loggedIn = true
	s.token = token
	s.session.CSRFHash = hash(csrf)
	return s.session.User, nil
}
func (s *stubStore) FindSession(_ context.Context, token string) (Session, error) {
	if token != s.token || s.revoked || time.Now().After(s.session.Expires) {
		return Session{}, ErrDenied
	}
	return s.session, s.err
}
func (s *stubStore) Revoke(context.Context, string, string, bool) error {
	s.revoked = true
	return s.err
}
func (s *stubStore) CreateInvitation(context.Context, string, string) (Invitation, error) {
	return Invitation{}, s.err
}
func (s *stubStore) Disable(context.Context, string, string) error { return s.err }

type stubProvider struct{ err error }

func (p stubProvider) AuthorizationURL(f Flow) string {
	return "https://provider.test/auth?state=" + f.State
}
func (p stubProvider) Exchange(context.Context, string, Flow) (Identity, error) {
	return Identity{Issuer: TelegramIssuer, Subject: "test"}, p.err
}

func TestCallbackBindingReplayAndCookie(t *testing.T) {
	for _, scenario := range []string{"valid", "browser", "state", "provider", "denied"} {
		t.Run(scenario, func(t *testing.T) {
			s := &stubStore{session: Session{User: User{ID: "owner", Role: "admin"}, Expires: time.Now().Add(time.Hour)}}
			p := stubProvider{}
			if scenario == "provider" {
				p.err = ErrDenied
			}
			h := NewHandler(s, p, "telegram", "https://grace.test")
			start := httptest.NewRequest("POST", "https://grace.test/api/auth/login", strings.NewReader(`{}`))
			start.Header.Set("Origin", "https://grace.test")
			w := httptest.NewRecorder()
			h.Login(w, start)
			if w.Code != 200 {
				t.Fatalf("start: %d %s", w.Code, w.Body.String())
			}
			f := s.flow
			browser := f.Browser
			state := f.State
			if scenario == "browser" {
				browser = randomToken()
			}
			if scenario == "state" {
				state = randomToken()
			}
			if scenario == "denied" {
				s.err = ErrDenied
			}
			callback := func() *httptest.ResponseRecorder {
				r := httptest.NewRequest("GET", "https://grace.test/api/auth/callback?code=ok&state="+state, nil)
				r.AddCookie(&http.Cookie{Name: flowCookie, Value: browser})
				w := httptest.NewRecorder()
				h.Callback(w, r)
				return w
			}
			w = callback()
			if scenario != "valid" {
				if s.loggedIn || w.Header().Get("Location") != "/login?error=authentication_failed" {
					t.Fatal("unsafe callback accepted")
				}
				return
			}
			if !s.loggedIn || w.Header().Get("Location") != "/" {
				t.Fatal("valid callback failed")
			}
			var found bool
			for _, c := range w.Result().Cookies() {
				if c.Name == sessionCookie {
					found = true
					if !c.Secure || !c.HttpOnly || c.Path != "/" || c.Domain != "" || c.SameSite != http.SameSiteLaxMode {
						t.Fatal("unsafe session cookie")
					}
				}
			}
			if !found {
				t.Fatal("session cookie missing")
			}
			s.loggedIn = false
			_ = callback()
			if s.loggedIn {
				t.Fatal("replayed callback accepted")
			}
		})
	}
}

func TestSessionAndMutationAuthorization(t *testing.T) {
	token := randomToken()
	for _, scenario := range []string{"valid", "missing", "expired", "revoked", "origin", "csrf", "member"} {
		t.Run(scenario, func(t *testing.T) {
			s := &stubStore{token: token, session: Session{User: User{ID: "user", Role: "admin"}, CSRFHash: hash(csrfToken(token)), Expires: time.Now().Add(time.Hour)}}
			if scenario == "expired" {
				s.session.Expires = time.Now().Add(-time.Hour)
			}
			if scenario == "revoked" {
				s.revoked = true
			}
			if scenario == "member" {
				s.session.User.Role = "member"
			}
			h := NewHandler(s, stubProvider{}, "telegram", "https://grace.test")
			r := httptest.NewRequest("POST", "https://grace.test/api/auth/invitations", strings.NewReader(`{"name":"Alice"}`))
			r.Header.Set("Origin", "https://grace.test")
			r.Header.Set("X-CSRF-Token", csrfToken(token))
			if scenario != "missing" {
				r.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
			}
			if scenario == "origin" {
				r.Header.Set("Origin", "https://evil.test")
			}
			if scenario == "csrf" {
				r.Header.Set("X-CSRF-Token", "wrong")
			}
			w := httptest.NewRecorder()
			h.Require(http.HandlerFunc(h.Invite)).ServeHTTP(w, r)
			want := 201
			switch scenario {
			case "missing", "expired", "revoked":
				want = 401
			case "origin", "csrf", "member":
				want = 403
			}
			if w.Code != want {
				t.Fatalf("got %d want %d", w.Code, want)
			}
		})
	}
}

func TestLockedModeAndLoginOriginFailClosed(t *testing.T) {
	h := NewHandler(nil, nil, "locked", "https://grace.test")
	w := httptest.NewRecorder()
	h.Require(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("protected route reached") })).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/accounts", nil))
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	h = NewHandler(&stubStore{}, stubProvider{}, "telegram", "https://grace.test")
	w = httptest.NewRecorder()
	h.Login(w, httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`)))
	if w.Code != 403 {
		t.Fatal("login Origin missing accepted")
	}
}
