package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

const emailCookie = "__Host-grace_email"

var emailCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return "", ErrDenied
	}
	for _, c := range value {
		if c > 127 {
			return "", ErrDenied
		}
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Name != "" || address.Address != value || !strings.Contains(value, "@") {
		return "", ErrDenied
	}
	return strings.ToLower(value), nil
}

type EmailSender interface {
	SendCode(context.Context, string, string) error
	Ready() bool
}
type EmailHandler struct {
	store        EmailStore
	sender       EmailSender
	mode, origin string
	budget       int
}

func NewEmailHandler(store EmailStore, sender EmailSender, mode, origin string, budget int) *EmailHandler {
	return &EmailHandler{store: store, sender: sender, mode: mode, origin: strings.TrimRight(origin, "/"), budget: budget}
}
func (h *EmailHandler) configured() bool { return h.sender != nil && h.sender.Ready() }
func (h *EmailHandler) Methods(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	web.WriteJSON(w, 200, map[string]any{"mode": h.mode, "email": h.mode == "email" && h.configured(), "telegram": h.mode == "telegram", "emailLinkAvailable": h.configured() && (h.mode == "email" || h.mode == "telegram")})
}
func (h *EmailHandler) Request(w http.ResponseWriter, r *http.Request) { h.request(w, r, "login", "") }
func (h *EmailHandler) RequestLink(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	h.request(w, r, "link", p.ID)
}
func (h *EmailHandler) request(w http.ResponseWriter, r *http.Request, purpose, userID string) {
	noStore(w)
	if !h.configured() || (purpose == "login" && h.mode != "email") {
		web.WriteError(w, 503, "auth_unavailable", "Отправка кода пока недоступна.")
		return
	}
	if r.Header.Get("Origin") != h.origin || h.origin == "" {
		web.WriteError(w, 403, "access_denied", "Недопустимый источник запроса.")
		return
	}
	if purpose == "link" {
		recent, ok := h.store.(interface {
			RecentSession(context.Context, string, string) (bool, error)
		})
		if !ok {
			web.WriteError(w, 503, "auth_unavailable", "Действие недоступно.")
			return
		}
		fresh, err := recent.RecentSession(r.Context(), userID, tokenCookie(r, sessionCookie))
		if err != nil || !fresh {
			web.WriteError(w, 403, "reauth_required", "Для изменения почты войдите в Grace заново. Если почта ещё не привязана, обратитесь к администратору.")
			return
		}
	}
	var input struct {
		Email      string `json:"email"`
		Invitation string `json:"invitation"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	email, err := normalizeEmail(input.Email)
	if err != nil || (input.Invitation != "" && len(input.Invitation) != 43) || (purpose == "link" && input.Invitation != "") {
		web.WriteError(w, 400, "invalid_request", "Проверьте адрес почты и приглашение.")
		return
	}
	source, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		source = r.RemoteAddr
	}
	if err = h.store.ReserveEmail(r.Context(), email, source, h.budget); err != nil {
		if err == ErrRateLimited {
			w.Header().Set("Retry-After", "60")
			web.WriteError(w, 429, "rate_limited", "Код недавно отправлен или достигнут лимит. Попробуйте позже.")
		} else {
			web.WriteError(w, 503, "auth_unavailable", "Отправка кода временно недоступна.")
		}
		return
	}
	browser := randomToken()
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		web.WriteError(w, 503, "auth_unavailable", "Не удалось подготовить код.")
		return
	}
	code := fmt.Sprintf("%06d", number.Int64())
	invitationHash := ""
	if input.Invitation != "" {
		invitationHash = hash(input.Invitation)
	}
	if purpose == "login" && invitationHash == "" {
		if resolver, ok := h.store.(interface {
			EmailMigrationHash(context.Context, string) (string, error)
		}); ok {
			invitationHash, err = resolver.EmailMigrationHash(r.Context(), email)
			if err != nil {
				web.WriteError(w, 503, "auth_unavailable", "Отправка кода временно недоступна.")
				return
			}
		}
	}
	flow := EmailFlow{BrowserHash: hash(browser), CodeHash: hash("grace-email:" + browser + ":" + code), Email: email, Purpose: purpose, UserID: userID, InvitationHash: invitationHash, Expires: time.Now().UTC().Add(10 * time.Minute)}
	if err = h.store.SaveEmailFlow(r.Context(), flow); err != nil {
		web.WriteError(w, 503, "auth_unavailable", "Отправка кода временно недоступна.")
		return
	}
	if err = h.sender.SendCode(r.Context(), email, code); err != nil {
		web.WriteError(w, 503, "delivery_unavailable", "Письмо не удалось отправить. Попробуйте позже.")
		return
	}
	setCookie(w, emailCookie, browser, 600)
	// Uniform response, including unknown/uninvited addresses. Invitation is checked only after possession is proven.
	web.WriteJSON(w, 200, map[string]any{"expiresInSeconds": 600, "resendAfterSeconds": 60})
}
func (h *EmailHandler) Verify(w http.ResponseWriter, r *http.Request) { h.verify(w, r, "login", "") }
func (h *EmailHandler) VerifyLink(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	h.verify(w, r, "link", p.ID)
}
func (h *EmailHandler) verify(w http.ResponseWriter, r *http.Request, purpose, userID string) {
	noStore(w)
	if !h.configured() || (purpose == "login" && h.mode != "email") {
		web.WriteError(w, 503, "auth_unavailable", "Вход по почте недоступен.")
		return
	}
	if r.Header.Get("Origin") != h.origin || h.origin == "" {
		web.WriteError(w, 403, "access_denied", "Недопустимый источник запроса.")
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	browser := tokenCookie(r, emailCookie)
	if browser == "" || !emailCodePattern.MatchString(input.Code) {
		web.WriteError(w, 422, "invalid_code", "Проверьте шестизначный код.")
		return
	}
	flow, err := h.store.TakeEmailFlow(r.Context(), hash(browser), hash("grace-email:"+browser+":"+input.Code), purpose, userID)
	if err != nil {
		web.WriteError(w, 422, "invalid_code", "Код неверен, истёк или уже использован.")
		return
	}
	if purpose == "link" {
		if err = h.store.LinkEmail(r.Context(), userID, flow.Email, tokenCookie(r, sessionCookie)); err != nil {
			web.WriteError(w, 409, "email_unavailable", "Не удалось привязать почту. Проверьте адрес или войдите заново.")
			return
		}
		setCookie(w, emailCookie, "", -1)
		w.WriteHeader(204)
		return
	}
	session := randomToken()
	csrf := csrfToken(session)
	user, err := h.store.Login(r.Context(), Identity{Issuer: EmailIssuer, Subject: flow.Email}, flow.InvitationHash, session, csrf)
	if err != nil {
		web.WriteError(w, 403, "invitation_required", "Вход недоступен. Для первого входа нужно действующее приглашение.")
		return
	}
	setCookie(w, emailCookie, "", -1)
	setCookie(w, sessionCookie, session, 7*24*60*60)
	web.WriteJSON(w, 200, map[string]any{"user": user, "csrfToken": csrf, "mode": "email"})
}
