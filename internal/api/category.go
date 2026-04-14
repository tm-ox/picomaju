package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/category"
)

type categoryHandler struct {
	store *category.Store
}

// GET /api/categories
func (h *categoryHandler) list(w http.ResponseWriter, r *http.Request) {
	cats, err := h.store.List()
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, cats)
}

// POST /api/categories
func (h *categoryHandler) create(w http.ResponseWriter, r *http.Request) {
	var c category.Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	c.System = false
	if err := h.store.Create(&c); err != nil {
		jsonErr(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, c)
}

// DELETE /api/categories/:id
func (h *categoryHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(id); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
