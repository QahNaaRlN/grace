package auth

import (
	"bytes"
	"context"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
)

type ProfileStore interface {
	Profile(context.Context, string) (Profile, error)
	UpdateProfile(context.Context, string, ProfileNames) error
	Avatar(context.Context, string) ([]byte, error)
	SetAvatar(context.Context, string, []byte) error
}
type ProfileHandler struct{ store ProfileStore }

func NewProfileHandler(store ProfileStore) *ProfileHandler { return &ProfileHandler{store: store} }
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	data, err := h.store.Profile(r.Context(), p.ID)
	if err != nil {
		web.WriteError(w, 503, "profile_unavailable", "Профиль временно недоступен.")
		return
	}
	web.WriteJSON(w, 200, data)
}
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	var input ProfileNames
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	if !input.valid() {
		web.WriteError(w, 422, "invalid_profile", "Проверьте имя и фамилию.")
		return
	}
	p, _ := access.Current(r.Context())
	if err := h.store.UpdateProfile(r.Context(), p.ID, input); err != nil {
		web.WriteError(w, 503, "profile_unavailable", "Не удалось сохранить профиль.")
		return
	}
	h.Get(w, r)
}
func (h *ProfileHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	data, err := h.store.Avatar(r.Context(), p.ID)
	if err != nil || len(data) == 0 {
		web.WriteError(w, http.StatusNotFound, "avatar_not_found", "Аватар не задан.")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	_, _ = w.Write(data)
}
func cleanAvatar(data []byte) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > 4000000 {
		return nil, ErrDenied
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrDenied
	}
	bounds := decoded.Bounds()
	side := bounds.Dx()
	if bounds.Dy() < side {
		side = bounds.Dy()
	}
	left := bounds.Min.X + (bounds.Dx()-side)/2
	top := bounds.Min.Y + (bounds.Dy()-side)/2
	avatar := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			avatar.Set(x, y, decoded.At(left+x*side/256, top+y*side/256))
		}
	}
	var output bytes.Buffer
	if err = png.Encode(&output, avatar); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, (2<<20)+8192)
	reader, err := r.MultipartReader()
	if err != nil {
		web.WriteError(w, 400, "invalid_avatar", "Выберите PNG или JPEG.")
		return
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "avatar" {
		web.WriteError(w, 400, "invalid_avatar", "Выберите изображение.")
		return
	}
	defer part.Close()
	data, err := io.ReadAll(io.LimitReader(part, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		web.WriteError(w, 413, "avatar_too_large", "Размер аватара должен быть не больше 2 МБ.")
		return
	}
	if extra, err := reader.NextPart(); err != io.EOF || extra != nil {
		web.WriteError(w, 400, "invalid_avatar", "Можно загрузить одно изображение.")
		return
	}
	cleaned, err := cleanAvatar(data)
	if err != nil {
		web.WriteError(w, 422, "invalid_avatar", "Нужен PNG/JPEG не больше 4 миллионов пикселей.")
		return
	}
	p, _ := access.Current(r.Context())
	if err = h.store.SetAvatar(r.Context(), p.ID, cleaned); err != nil {
		web.WriteError(w, 503, "profile_unavailable", "Не удалось сохранить аватар.")
		return
	}
	h.Get(w, r)
}
func (h *ProfileHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	p, _ := access.Current(r.Context())
	if err := h.store.SetAvatar(r.Context(), p.ID, nil); err != nil {
		web.WriteError(w, 503, "profile_unavailable", "Не удалось удалить аватар.")
		return
	}
	w.WriteHeader(204)
}
