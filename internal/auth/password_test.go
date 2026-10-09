package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPasswordHashAndPolicy(t *testing.T) {
	password := "a memorable long phrase"
	encoded := hashPassword(password)
	if encoded == hashPassword(password) || !verifyPassword(password, encoded) || verifyPassword(password+"x", encoded) {
		t.Fatal("salt/verification failure")
	}
	if !validPassword("пятнадцать слов для пароля") || validPassword("short") || validPassword(strings.Repeat("x", 129)) {
		t.Fatal("policy failure")
	}
	if verifyPassword(password, "$argon2id$v=19$m=999999999,t=2,p=1$bad$bad") {
		t.Fatal("unbounded parameters accepted")
	}
}
func TestPasswordPublicOriginAndMode(t *testing.T) {
	h := &PasswordHandler{mode: "password", origin: "https://grace.test"}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://evil.test")
	if h.begin(w, r) || w.Code != 403 {
		t.Fatal("cross origin accepted")
	}
	h.mode = "email"
	w = httptest.NewRecorder()
	if h.begin(w, r) || w.Code != 503 {
		t.Fatal("password enabled in legacy mode")
	}
}
