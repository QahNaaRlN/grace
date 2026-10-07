package auth

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeEmailStore struct {
	flow       EmailFlow
	attempts   int
	consumed   bool
	logged     bool
	identity   Identity
	invitation string
	limited    bool
	linked     bool
}

func (s *fakeEmailStore) ReserveEmail(context.Context, string, string, int) error {
	if s.limited {
		return ErrRateLimited
	}
	return nil
}
func (s *fakeEmailStore) SaveEmailFlow(_ context.Context, f EmailFlow) error {
	s.flow = f
	s.attempts = 0
	s.consumed = false
	return nil
}
func (s *fakeEmailStore) TakeEmailFlow(_ context.Context, browser, code, purpose, user string) (EmailFlow, error) {
	if s.consumed || browser != s.flow.BrowserHash || purpose != s.flow.Purpose || user != s.flow.UserID || !time.Now().Before(s.flow.Expires) || s.attempts >= 5 {
		return EmailFlow{}, ErrDenied
	}
	if code != s.flow.CodeHash {
		s.attempts++
		return EmailFlow{}, ErrDenied
	}
	s.consumed = true
	return s.flow, nil
}
func (s *fakeEmailStore) Login(_ context.Context, i Identity, inv, token, csrf string) (User, error) {
	s.logged = true
	s.identity = i
	s.invitation = inv
	return User{ID: "owner", Name: "Owner", Role: "admin"}, nil
}
func (s *fakeEmailStore) LinkEmail(context.Context, string, string, string) error {
	s.linked = true
	return nil
}
func (s *fakeEmailStore) RecentSession(context.Context, string, string) (bool, error) {
	return true, nil
}

type fakeEmailSender struct {
	email, code string
	err         error
}

func (s *fakeEmailSender) Ready() bool { return true }
func (s *fakeEmailSender) SendCode(_ context.Context, email, code string) error {
	s.email = email
	s.code = code
	return s.err
}
func emailRequest(body string, cookie *http.Cookie) *http.Request {
	r := httptest.NewRequest("POST", "https://grace.test/api/auth/email", strings.NewReader(body))
	r.Header.Set("Origin", "https://grace.test")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}
func TestEmailCodeBindingAndOneUse(t *testing.T) {
	store, sender := &fakeEmailStore{}, &fakeEmailSender{}
	h := NewEmailHandler(store, sender, "email", "https://grace.test", 1500)
	w := httptest.NewRecorder()
	h.Request(w, emailRequest(`{"email":"USER@example.test","invitation":"`+strings.Repeat("x", 43)+`"}`, nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.Secure || !cookie.HttpOnly || cookie.Name != emailCookie {
		t.Fatal("unsafe cookie")
	}
	if strings.Contains(w.Body.String(), sender.code) || store.flow.Email != "user@example.test" {
		t.Fatal("code leaked or address not normalized")
	}
	w = httptest.NewRecorder()
	h.Verify(w, emailRequest(`{"code":"`+sender.code+`"}`, nil))
	if store.logged || w.Code != 422 {
		t.Fatal("browser binding missing")
	}
	w = httptest.NewRecorder()
	h.Verify(w, emailRequest(`{"code":"`+sender.code+`"}`, cookie))
	if w.Code != 200 || !store.logged || store.identity.Issuer != EmailIssuer || store.invitation != hash(strings.Repeat("x", 43)) {
		t.Fatal("valid login rejected", w.Code)
	}
	store.logged = false
	w = httptest.NewRecorder()
	h.Verify(w, emailRequest(`{"code":"`+sender.code+`"}`, cookie))
	if store.logged || w.Code != 422 {
		t.Fatal("code reusable")
	}
}
func TestEmailDeniedAfterFiveAttemptsOrExpiry(t *testing.T) {
	for _, expired := range []bool{false, true} {
		s := &fakeEmailStore{flow: EmailFlow{BrowserHash: hash(strings.Repeat("b", 43)), CodeHash: hash("grace-email:" + strings.Repeat("b", 43) + ":000001"), Purpose: "login", Expires: time.Now().Add(time.Minute)}}
		if expired {
			s.flow.Expires = time.Now().Add(-time.Second)
		}
		h := NewEmailHandler(s, &fakeEmailSender{}, "email", "https://grace.test", 1500)
		cookie := &http.Cookie{Name: emailCookie, Value: strings.Repeat("b", 43)}
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			h.Verify(w, emailRequest(`{"code":"000000"}`, cookie))
		}
		w := httptest.NewRecorder()
		h.Verify(w, emailRequest(`{"code":"000001"}`, cookie))
		if w.Code != 422 || s.logged {
			t.Fatal("limit/expiry bypass")
		}
	}
}
func TestEmailModeAndOriginFailClosed(t *testing.T) {
	for _, mode := range []string{"email", "telegram", "locked"} {
		h := NewEmailHandler(&fakeEmailStore{}, &fakeEmailSender{}, mode, "https://grace.test", 1500)
		r := emailRequest(`{"email":"user@example.test"}`, nil)
		r.Header.Set("Origin", "https://evil.test")
		w := httptest.NewRecorder()
		h.Request(w, r)
		if w.Code == 200 {
			t.Fatal("unsafe origin or mode accepted")
		}
	}
}
func TestAvatarReencodedAndOversizedRejected(t *testing.T) {
	var data bytes.Buffer
	_ = png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 20, 10)))
	cleaned, err := cleanAvatar(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(cleaned))
	if err != nil || format != "png" || cfg.Width != 256 || cfg.Height != 256 {
		t.Fatal("avatar not normalized")
	}
	if _, err = cleanAvatar([]byte(`<svg onload="alert(1)"></svg>`)); err == nil {
		t.Fatal("SVG accepted")
	}
	data.Reset()
	_ = png.Encode(&data, image.NewGray(image.Rect(0, 0, 2048, 2048)))
	if _, err = cleanAvatar(data.Bytes()); err == nil {
		t.Fatal("pixel limit missing")
	}
}
func TestProfileNamesNotRole(t *testing.T) {
	n := ProfileNames{DisplayName: "  Person  ", FirstName: "  Name ", LastName: strings.Repeat("я", 81)}
	if n.valid() {
		t.Fatal("long surname accepted")
	}
	n.LastName = " Family "
	if !n.valid() || n.FirstName != "Name" || n.LastName != "Family" {
		t.Fatal("profile validation")
	}
}

func TestSMTPSenderNeverAuthenticatesWithoutTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	commands := make(chan string, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			commands <- ""
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		_, _ = conn.Write([]byte("220 test SMTP\r\n"))
		reader := bufio.NewReader(conn)
		var received strings.Builder
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				break
			}
			received.WriteString(line)
			if strings.HasPrefix(line, "EHLO") {
				_, _ = conn.Write([]byte("250-test\r\n250 AUTH PLAIN\r\n"))
			} else {
				_, _ = conn.Write([]byte("250 OK\r\n"))
			}
		}
		commands <- received.String()
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	sender := NewSMTPSender(SMTPSettings{Host: host, Port: port, Username: "private-user", Password: "private-password", From: "no-reply@example.test", Security: "starttls"})
	err = sender.SendCode(context.Background(), "user@example.test", "123456")
	if err == nil || err.Error() != "mail TLS required" {
		t.Fatal("plaintext server accepted", err)
	}
	if received := <-commands; strings.Contains(received, "AUTH") || strings.Contains(received, "MAIL") || strings.Contains(received, "123456") {
		t.Fatal("credentials/message transmitted without TLS")
	}
}

func TestMailDeliveryFailureDoesNotExposeSecrets(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	store := &fakeEmailStore{}
	sender := &fakeEmailSender{err: errors.New("private-smtp-password")}
	h := NewEmailHandler(store, sender, "email", "https://grace.test", 1500)
	w := httptest.NewRecorder()
	h.Request(w, emailRequest(`{"email":"private-user@example.test"}`, nil))
	if w.Code == 200 || len(w.Result().Cookies()) != 0 {
		t.Fatal("failed delivery created browser login")
	}
	for _, secret := range []string{"private-smtp-password", sender.code, "private-user@example.test"} {
		if secret != "" && (strings.Contains(logs.String(), secret) || strings.Contains(w.Body.String(), secret)) {
			t.Fatal("delivery failure leaked secret")
		}
	}
}

func TestEmailModeDisablesTelegramEntryPoints(t *testing.T) {
	store := &stubStore{}
	h := NewHandler(store, stubProvider{}, "email", "https://grace.test")
	for _, endpoint := range []struct {
		handle http.HandlerFunc
		status int
	}{{h.Login, 503}, {h.Callback, 303}} {
		w := httptest.NewRecorder()
		endpoint.handle(w, httptest.NewRequest("GET", "https://grace.test/api/auth/login", nil))
		if w.Code != endpoint.status || store.loggedIn || store.flowTaken {
			t.Fatal("Telegram entry point accepted in email mode", w.Code)
		}
	}
}
