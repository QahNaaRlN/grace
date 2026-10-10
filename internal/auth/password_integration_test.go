//go:build integration

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type passwordTestMail struct{ links chan string }

func (m *passwordTestMail) Ready() bool { return true }
func (m *passwordTestMail) SendLink(ctx context.Context, email, link, purpose string) error {
	select {
	case m.links <- link:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func passwordMailToken(t *testing.T, m *passwordTestMail) string {
	t.Helper()
	select {
	case link := <-m.links:
		u, err := url.Parse(link)
		if err != nil {
			t.Fatal("invalid link")
		}
		values, _ := url.ParseQuery(u.Fragment)
		return values.Get("token")
	case <-time.After(3 * time.Second):
		t.Fatal("no test mail")
		return ""
	}
}
func TestPasswordLifecycleAgainstPostgreSQL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := NewRepository(pool)
	mail := &passwordTestMail{links: make(chan string, 10)}
	h := NewPasswordHandler(repo, mail, "password", "https://grace.test", 1000)
	fingerprint := func() string {
		var result string
		err := pool.QueryRow(ctx, `SELECT md5((SELECT jsonb_agg(to_jsonb(a) ORDER BY a.id)::text FROM accounts a)||COALESCE((SELECT jsonb_agg(to_jsonb(e) ORDER BY e.id)::text FROM financial_events e),'')||COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY p.id)::text FROM postings p),''))`).Scan(&result)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	before := fingerprint()
	email := "owner-password@example.test"
	if _, err = pool.Exec(ctx, `INSERT INTO auth_identities(issuer,subject,user_id) VALUES($1,$2,$3)`, EmailIssuer, email, access.BootstrapOwnerID); err != nil {
		t.Fatal(err)
	}
	counter := 0
	call := func(handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
		t.Helper()
		payload, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/", strings.NewReader(string(payload)))
		r.Header.Set("Origin", "https://grace.test")
		counter++
		r.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", counter)
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	known := call(h.Forgot, map[string]string{"email": email})
	if known.Code != 202 {
		t.Fatal("forgot known", known.Code)
	}
	firstToken := passwordMailToken(t, mail)
	unknown := call(h.Forgot, map[string]string{"email": "unknown-password@example.test"})
	if unknown.Code != 202 || known.Body.String() != unknown.Body.String() {
		t.Fatal("account enumeration")
	}
	password := "synthetic password phrase"
	first := call(h.Reset, map[string]string{"token": firstToken, "password": password})
	if first.Code != 204 {
		t.Fatal("first password", first.Code)
	}
	if call(h.Reset, map[string]string{"token": firstToken, "password": password}).Code != 422 {
		t.Fatal("reset replay")
	}
	login := call(h.Login, map[string]string{"email": email, "password": password})
	if login.Code != 200 {
		t.Fatal("login", login.Code)
	}
	var session struct{ User User }
	if err = json.Unmarshal(login.Body.Bytes(), &session); err != nil || session.User.ID != access.BootstrapOwnerID || session.User.Role != "admin" {
		t.Fatal("owner identity changed")
	}
	var cookie *http.Cookie
	for _, c := range login.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly || !cookie.Secure {
		t.Fatal("session cookie")
	}
	if _, err = repo.FindSession(ctx, cookie.Value); err != nil {
		t.Fatal("session missing")
	}
	// A stolen/reset link issued before a credential change must not work later.
	user, _, revision, _ := repo.PasswordIdentity(ctx, email)
	newer, stale := randomToken(), randomToken()
	for _, raw := range []string{newer, stale} {
		if err = repo.SavePasswordToken(ctx, PasswordToken{TokenHash: hash(raw), Purpose: "reset", UserID: user.ID, Email: email, Revision: revision}); err != nil {
			t.Fatal(err)
		}
	}
	if call(h.Reset, map[string]string{"token": newer, "password": "another synthetic password"}).Code != 204 {
		t.Fatal("password replacement")
	}
	if _, err = repo.FindSession(ctx, cookie.Value); err != ErrDenied {
		t.Fatal("old session survived reset")
	}
	if call(h.Reset, map[string]string{"token": stale, "password": password}).Code != 422 {
		t.Fatal("stale reset survived")
	}
	// Admin-created invitation is bound to email and does not activate on GET.
	invite, err := repo.CreateBoundInvitation(ctx, access.BootstrapOwnerID, "Synthetic member", "member-password@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if call(h.Register, map[string]string{"email": "wrong@example.test", "password": password, "invitation": invite.Token}).Code != 403 {
		t.Fatal("wrong-email invite")
	}
	registered := call(h.Register, map[string]string{"email": "member-password@example.test", "password": password, "invitation": invite.Token})
	if registered.Code != 202 {
		t.Fatal("register", registered.Code)
	}
	token := passwordMailToken(t, mail)
	if call(h.Login, map[string]string{"email": "member-password@example.test", "password": password}).Code != 401 {
		t.Fatal("unverified login")
	}
	var used bool
	if err = pool.QueryRow(ctx, `SELECT used_at IS NOT NULL FROM auth_invitations WHERE token_hash=$1`, hash(invite.Token)).Scan(&used); err != nil || used {
		t.Fatal("invite consumed early")
	}
	// Two simultaneous finalizations grant exactly one credential.
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest("POST", "/", strings.NewReader(`{"token":"`+token+`"}`))
			r.Header.Set("Origin", "https://grace.test")
			r.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", i+1)
			w := httptest.NewRecorder()
			h.Confirm(w, r)
			results <- w.Code
		}(i)
	}
	wg.Wait()
	close(results)
	success, denied := 0, 0
	for status := range results {
		if status == 204 {
			success++
		} else if status == 422 {
			denied++
		}
	}
	if success != 1 || denied != 1 {
		t.Fatal("confirmation race")
	}
	if call(h.Login, map[string]string{"email": "member-password@example.test", "password": password}).Code != 200 {
		t.Fatal("member cannot log in without invitation")
	}
	ownerLogin := call(h.Login, map[string]string{"email": email, "password": "another synthetic password"})
	if ownerLogin.Code != 200 {
		t.Fatal("new owner password failed")
	}
	var ownerCookie *http.Cookie
	for _, c := range ownerLogin.Result().Cookies() {
		if c.Name == sessionCookie {
			ownerCookie = c
		}
	}
	protected := NewHandler(repo, nil, "password", "https://grace.test").Require(http.HandlerFunc(h.Invite))
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Protected member","email":"protected@example.test"}`))
	r.Header.Set("Origin", "https://grace.test")
	r.AddCookie(ownerCookie)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("CSRF bypass")
	}
	r = httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Protected member","email":"protected@example.test"}`))
	r.Header.Set("Origin", "https://grace.test")
	r.Header.Set("X-CSRF-Token", csrfToken(ownerCookie.Value))
	r.AddCookie(ownerCookie)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal("protected invitation", w.Code)
	}
	r = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://grace.test")
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("anonymous invitation")
	}
	if call(h.Login, map[string]string{"email": email, "password": "another synthetic password", "invitation": invite.Token}).Code != 400 {
		t.Fatal("login accepted invitation")
	}
	member, _, _, _ := repo.PasswordIdentity(ctx, "member-password@example.test")
	raw := randomToken()
	if err = repo.SavePasswordToken(ctx, PasswordToken{TokenHash: hash(raw), Purpose: "reset", UserID: member.ID, Email: "member-password@example.test", Revision: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE auth_password_tokens SET expires_at=now()-interval '1 second' WHERE token_hash=$1`, hash(raw)); err != nil {
		t.Fatal(err)
	}
	if call(h.Reset, map[string]string{"token": raw, "password": password}).Code != 422 {
		t.Fatal("expired token")
	}
	if _, err = pool.Exec(ctx, `UPDATE users SET disabled=true WHERE id=$1`, member.ID); err != nil {
		t.Fatal(err)
	}
	if call(h.Login, map[string]string{"email": "member-password@example.test", "password": password}).Code != 401 {
		t.Fatal("disabled user login")
	}
	if fingerprint() != before {
		t.Fatal("financial fingerprint changed")
	}
	// Persistent limiter: all requests share the same window across repository instances.
	for i := 0; i < 5; i++ {
		if err = repo.ReservePasswordAttempt(ctx, "rate@example.test", "limited-source"); err != nil {
			t.Fatal("limit too early")
		}
	}
	if NewRepository(pool).ReservePasswordAttempt(ctx, "rate@example.test", "limited-source") != ErrRateLimited {
		t.Fatal("limiter reset with handler")
	}
}
