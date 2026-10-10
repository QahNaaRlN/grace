package auth

import (
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type PasswordSender interface {
	Ready() bool
	SendLink(context.Context, string, string, string) error
}
type PasswordHandler struct {
	repo                *Repository
	sender              PasswordSender
	mode, origin, dummy string
	budget              int
	work, mail          chan struct{}
}

func NewPasswordHandler(repo *Repository, sender PasswordSender, mode, origin string, budget int) *PasswordHandler {
	return &PasswordHandler{repo: repo, sender: sender, mode: mode, origin: strings.TrimRight(origin, "/"), budget: budget, dummy: hashPassword(randomToken()), work: make(chan struct{}, 2), mail: make(chan struct{}, 2)}
}
func (h *PasswordHandler) Methods(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	web.WriteJSON(w, 200, map[string]any{"mode": h.mode, "password": h.mode == "password" && h.sender.Ready(), "email": false, "telegram": false, "emailLinkAvailable": h.sender.Ready()})
}
func (h *PasswordHandler) begin(w http.ResponseWriter, r *http.Request) bool {
	noStore(w)
	if h.mode != "password" {
		web.WriteError(w, 503, "auth_unavailable", "Вход по паролю пока недоступен.")
		return false
	}
	if h.origin == "" || r.Header.Get("Origin") != h.origin {
		web.WriteError(w, 403, "access_denied", "Недопустимый источник запроса.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	return true
}
func (h *PasswordHandler) reserve(w http.ResponseWriter, r *http.Request, email string) bool {
	source, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		source = r.RemoteAddr
	}
	err = h.repo.ReservePasswordAttempt(r.Context(), email, source)
	if err != nil {
		code := 503
		if errors.Is(err, ErrRateLimited) {
			code = 429
			w.Header().Set("Retry-After", "60")
		}
		web.WriteError(w, code, "auth_unavailable", "Повторите попытку позже.")
		return false
	}
	return true
}
func (h *PasswordHandler) acquire(w http.ResponseWriter) bool {
	select {
	case h.work <- struct{}{}:
		return true
	default:
		web.WriteError(w, 429, "rate_limited", "Повторите попытку позже.")
		return false
	}
}
func (h *PasswordHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.begin(w, r) {
		return
	}
	var input struct{ Email, Password string }
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	email, err := normalizeEmail(input.Email)
	if err != nil || len(input.Password) > 512 {
		web.WriteError(w, 401, "invalid_credentials", "Неверная почта или пароль.")
		return
	}
	if !h.reserve(w, r, email) || !h.acquire(w) {
		return
	}
	defer func() { <-h.work }()
	_, encoded, _, lookupErr := h.repo.PasswordIdentity(r.Context(), email)
	candidate := encoded
	if lookupErr != nil || candidate == "" {
		candidate = h.dummy
	}
	matched := verifyPassword(input.Password, candidate)
	if lookupErr != nil || encoded == "" || !matched {
		web.WriteError(w, 401, "invalid_credentials", "Неверная почта или пароль.")
		return
	}
	session := randomToken()
	user, err := h.repo.PasswordSession(r.Context(), email, encoded, session)
	if err != nil {
		web.WriteError(w, 401, "invalid_credentials", "Неверная почта или пароль.")
		return
	}
	setCookie(w, sessionCookie, session, 7*24*60*60)
	web.WriteJSON(w, 200, map[string]any{"user": user, "csrfToken": csrfToken(session), "mode": "password"})
}
func (h *PasswordHandler) send(token, email, purpose string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer func() { <-h.mail }()
	path := "/reset-password#token="
	if purpose == "register" {
		path = "/register#token="
	}
	if err := h.sender.SendLink(ctx, email, h.origin+path+token, purpose); err != nil {
		h.repo.CancelPasswordToken(ctx, hash(token))
	}
}
func (h *PasswordHandler) requestToken(w http.ResponseWriter, r *http.Request, registration bool) {
	if !registration {
		started := time.Now()
		defer func() {
			remaining := 250*time.Millisecond - time.Since(started)
			if remaining > 0 {
				timer := time.NewTimer(remaining)
				defer timer.Stop()
				select {
				case <-timer.C:
				case <-r.Context().Done():
				}
			}
		}()
	}
	if !h.begin(w, r) {
		return
	}
	if !h.sender.Ready() {
		web.WriteError(w, 503, "auth_unavailable", "Почтовая отправка пока недоступна.")
		return
	}
	var input struct{ Email, Password, Invitation string }
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	if !registration && (input.Password != "" || input.Invitation != "") {
		web.WriteError(w, 400, "invalid_request", "Для восстановления нужна только почта.")
		return
	}
	email, err := normalizeEmail(input.Email)
	if err != nil {
		web.WriteError(w, 422, "invalid_request", "Проверьте адрес почты.")
		return
	}
	if !h.reserve(w, r, email) {
		return
	}
	if registration && (!validPassword(input.Password) || len(input.Invitation) != 43) {
		web.WriteError(w, 422, "invalid_request", "Нужны пароль от 15 до 128 символов и действующее приглашение.")
		return
	}
	if err = h.repo.ReserveEmail(r.Context(), email, "password-mail", h.budget); err != nil {
		web.WriteError(w, 429, "rate_limited", "Повторите отправку через минуту.")
		return
	}
	select {
	case h.mail <- struct{}{}:
	default:
		web.WriteError(w, 429, "rate_limited", "Повторите отправку позже.")
		return
	}
	pending := false
	defer func() {
		if !pending {
			<-h.mail
		}
	}()
	f := PasswordToken{Email: email, Purpose: "reset"}
	raw := randomToken()
	f.TokenHash = hash(raw)
	if registration {
		if !h.acquire(w) {
			return
		}
		f.PasswordHash = hashPassword(input.Password)
		<-h.work
		f.Purpose = "register"
		f.InvitationHash = hash(input.Invitation)
		f.UserID, err = h.repo.RegistrationTarget(r.Context(), email, input.Invitation)
		if err != nil {
			web.WriteError(w, 403, "registration_unavailable", "Проверьте приглашение и адрес почты. Для существующего аккаунта используйте восстановление пароля.")
			return
		}
	} else {
		var user User
		user, _, f.Revision, err = h.repo.PasswordIdentity(r.Context(), email)
		f.UserID = user.ID
		if err != nil && !errors.Is(err, ErrDenied) {
			web.WriteError(w, 503, "auth_unavailable", "Повторите отправку позже.")
			return
		}
	}
	if f.UserID != "" {
		if err = h.repo.SavePasswordToken(r.Context(), f); err != nil {
			web.WriteError(w, 503, "auth_unavailable", "Повторите отправку позже.")
			return
		}
		pending = true
		go h.send(raw, email, f.Purpose)
	}
	web.WriteJSON(w, 202, map[string]any{"message": "Если действие доступно для этой почты, письмо придёт в ближайшее время.", "resendAfterSeconds": 60})
}
func (h *PasswordHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.requestToken(w, r, true)
}
func (h *PasswordHandler) Forgot(w http.ResponseWriter, r *http.Request) { h.requestToken(w, r, false) }
func (h *PasswordHandler) finish(w http.ResponseWriter, r *http.Request, registration bool) {
	if !h.begin(w, r) {
		return
	}
	var input struct{ Token, Password string }
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	if registration && input.Password != "" {
		web.WriteError(w, 400, "invalid_request", "Для подтверждения нужна только ссылка.")
		return
	}
	if len(input.Token) != 43 || (!registration && !validPassword(input.Password)) {
		web.WriteError(w, 422, "invalid_request", "Проверьте ссылку и пароль от 15 до 128 символов.")
		return
	}
	if !h.reserve(w, r, "token:"+hash(input.Token)) {
		return
	}
	encoded, purpose := "", "register"
	if !registration {
		if !h.acquire(w) {
			return
		}
		encoded = hashPassword(input.Password)
		<-h.work
		purpose = "reset"
	}
	if err := h.repo.FinishPasswordToken(r.Context(), input.Token, purpose, encoded); err != nil {
		web.WriteError(w, 422, "invalid_token", "Ссылка неверна, истекла или уже использована. Запросите новое письмо.")
		return
	}
	setCookie(w, sessionCookie, "", -1)
	w.WriteHeader(204)
}
func (h *PasswordHandler) Confirm(w http.ResponseWriter, r *http.Request) { h.finish(w, r, true) }
func (h *PasswordHandler) Reset(w http.ResponseWriter, r *http.Request)   { h.finish(w, r, false) }
func (h *PasswordHandler) Invite(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	p, _ := access.Current(r.Context())
	if p.Role != "admin" || h.mode != "password" {
		web.WriteError(w, 403, "access_denied", "Недостаточно прав.")
		return
	}
	var input struct{ Name, Email string }
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	email, err := normalizeEmail(input.Email)
	name := strings.TrimSpace(input.Name)
	if err != nil || name == "" || utf8.RuneCountInString(name) > 160 {
		web.WriteError(w, 422, "invalid_request", "Укажите имя и почту приглашённого.")
		return
	}
	invite, err := h.repo.CreateBoundInvitation(r.Context(), p.ID, name, email)
	if err != nil {
		web.WriteError(w, 409, "invitation_unavailable", "Приглашение недоступно. Возможно, эта почта уже зарегистрирована.")
		return
	}
	web.WriteJSON(w, 201, map[string]any{"token": invite.Token, "email": email, "expiresAt": invite.Expires, "url": h.origin + "/register#invitation=" + invite.Token})
}
