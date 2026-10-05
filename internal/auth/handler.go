package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/web"
)

const sessionCookie = "__Host-grace_session"
const flowCookie = "__Host-grace_login"

type Handler struct {
	store       Store
	provider    Provider
	mode        string
	origin      string
	mu          sync.Mutex
	loginWindow time.Time
	loginCount  int
}

func NewHandler(store Store, provider Provider, mode, origin string) *Handler {
	return &Handler{store: store, provider: provider, mode: mode, origin: strings.TrimRight(origin, "/")}
}
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}
func tokenCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil || len(c.Value) != 43 {
		return ""
	}
	return c.Value
}
func setCookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (h *Handler) sameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == h.origin && h.origin != ""
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if h.mode != "telegram" || h.provider == nil {
		web.WriteError(w, 503, "auth_unavailable", "Вход пока не настроен.")
		return
	}
	if !h.sameOrigin(r) {
		web.WriteError(w, 403, "access_denied", "Недопустимый источник запроса.")
		return
	}
	h.mu.Lock()
	if time.Since(h.loginWindow) >= time.Minute {
		h.loginWindow = time.Now()
		h.loginCount = 0
	}
	h.loginCount++
	limited := h.loginCount > 60
	h.mu.Unlock()
	if limited {
		w.Header().Set("Retry-After", "60")
		web.WriteError(w, 429, "rate_limited", "Попробуйте позже.")
		return
	}
	var input struct {
		Invitation string `json:"invitation"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	if input.Invitation != "" && len(input.Invitation) != 43 {
		web.WriteError(w, 400, "invalid_request", "Недопустимое приглашение.")
		return
	}
	f := Flow{State: randomToken(), Browser: randomToken(), Verifier: randomToken(), Nonce: randomToken(), Expires: time.Now().UTC().Add(10 * time.Minute)}
	if input.Invitation != "" {
		f.InvitationHash = hash(input.Invitation)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.store.SaveFlow(ctx, f); err != nil {
		web.WriteError(w, 503, "auth_unavailable", "Вход временно недоступен.")
		return
	}
	setCookie(w, flowCookie, f.Browser, 600)
	web.WriteJSON(w, 200, map[string]string{"authorizationUrl": h.provider.AuthorizationURL(f)})
}
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	browser := tokenCookie(r, flowCookie)
	setCookie(w, flowCookie, "", -1)
	fail := func(stage string) {
		slog.Warn("authentication callback rejected", "stage", stage)
		http.Redirect(w, r, "/login?error=authentication_failed", http.StatusSeeOther)
	}
	state, code := r.URL.Query().Get("state"), r.URL.Query().Get("code")
	if h.mode != "telegram" || h.provider == nil || browser == "" || len(state) != 43 || code == "" || len(code) > 4096 || r.URL.Query().Get("error") != "" {
		fail("browser_or_callback")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	f, err := h.store.TakeFlow(ctx, state, browser)
	if err != nil {
		fail("login_flow")
		return
	}
	identity, err := h.provider.Exchange(ctx, code, f)
	if err != nil {
		fail(failureStage(err))
		return
	}
	token := randomToken()
	if _, err = h.store.Login(ctx, identity, f.InvitationHash, token, csrfToken(token)); err != nil {
		fail("identity_or_invitation")
		return
	}
	setCookie(w, sessionCookie, token, 7*24*3600)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Require protects every financial route before a repository sees the request.
func (h *Handler) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		if h.mode == "development" {
			p := access.Principal{ID: access.BootstrapOwnerID, Name: "Владелец (development)", Role: "admin"}
			next.ServeHTTP(w, r.WithContext(access.WithPrincipal(r.Context(), p)))
			return
		}
		if h.mode != "telegram" {
			web.WriteError(w, 401, "unauthenticated", "Войдите в Grace.")
			return
		}
		token := tokenCookie(r, sessionCookie)
		if token == "" {
			web.WriteError(w, 401, "unauthenticated", "Войдите в Grace.")
			return
		}
		s, err := h.store.FindSession(r.Context(), token)
		if errors.Is(err, ErrDenied) {
			setCookie(w, sessionCookie, "", -1)
			web.WriteError(w, 401, "unauthenticated", "Сессия завершена.")
			return
		}
		if err != nil {
			web.WriteError(w, 503, "auth_unavailable", "Проверка доступа временно недоступна.")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			actual := hash(r.Header.Get("X-CSRF-Token"))
			if !h.sameOrigin(r) || subtle.ConstantTimeCompare([]byte(actual), []byte(s.CSRFHash)) != 1 {
				web.WriteError(w, 403, "csrf_failed", "Обновите страницу и повторите действие.")
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(access.WithPrincipal(r.Context(), s.User.Principal())))
	})
}
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	csrf := "development"
	if h.mode == "telegram" {
		csrf = csrfToken(tokenCookie(r, sessionCookie))
	}
	web.WriteJSON(w, 200, map[string]any{"user": User{ID: p.ID, Name: p.Name, Role: p.Role}, "csrfToken": csrf, "mode": h.mode})
}
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request)         { h.revoke(w, r, false) }
func (h *Handler) RevokeSessions(w http.ResponseWriter, r *http.Request) { h.revoke(w, r, true) }
func (h *Handler) revoke(w http.ResponseWriter, r *http.Request, all bool) {
	p, _ := access.Current(r.Context())
	if err := h.store.Revoke(r.Context(), p.ID, tokenCookie(r, sessionCookie), all); err != nil {
		web.WriteError(w, 503, "auth_unavailable", "Не удалось завершить сессию.")
		return
	}
	setCookie(w, sessionCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	if p.Role != "admin" {
		web.WriteError(w, 403, "access_denied", "Недостаточно прав.")
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	if len(strings.TrimSpace(input.Name)) == 0 || len(input.Name) > 160 {
		web.WriteError(w, 400, "invalid_request", "Укажите имя до 160 символов.")
		return
	}
	i, err := h.store.CreateInvitation(r.Context(), p.ID, input.Name)
	if err != nil {
		web.WriteError(w, 503, "auth_unavailable", "Не удалось создать приглашение.")
		return
	}
	web.WriteJSON(w, http.StatusCreated, i)
}
func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	if p.Role != "admin" {
		web.WriteError(w, 403, "access_denied", "Недостаточно прав.")
		return
	}
	// chi sets the path value via the adapter registered in httpserver.
	if err := h.store.Disable(r.Context(), p.ID, r.PathValue("userId")); err != nil {
		web.WriteError(w, 403, "access_denied", "Действие недоступно.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
