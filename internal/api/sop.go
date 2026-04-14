package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/sop"
)

type sopHandler struct {
	store *sop.Store
}

// GET /sops — list all SOPs.
func (h *sopHandler) list(w http.ResponseWriter, r *http.Request) {
	sops, err := h.store.List()
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sops == nil {
		sops = []*sop.SOP{}
	}
	jsonOK(w, sops)
}

// GET /sops/:id — get one SOP.
func (h *sopHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.store.Get(id)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, s)
}

// POST /sops — create a SOP.
func (h *sopHandler) create(w http.ResponseWriter, r *http.Request) {
	var s sop.SOP
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.Create(&s); err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, s)
}

// PUT /sops/:id — replace a SOP.
func (h *sopHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var s sop.SOP
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.ID = id
	if err := h.store.Update(&s); err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, s)
}

// DELETE /sops/:id — delete a SOP.
func (h *sopHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(id); err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /sops/:id/validate — validate a SOP without compiling.
func (h *sopHandler) validate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.store.Get(id)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	result := sop.Validate(s)
	jsonOK(w, result)
}
