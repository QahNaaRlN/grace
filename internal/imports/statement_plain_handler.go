package imports

import (
	"net/http"
)

func (h *Handler) PlainReview(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in PlainReviewInput
	if !decode(w, r, &in) {
		return
	}
	out, e := h.repo.ReviewPlain(r.Context(), id, key(r), in)
	w.Header().Set("Cache-Control", "no-store")
	respond(w, out, e)
}
