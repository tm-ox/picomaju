package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/role"
	"picomaju/internal/sop"
)

type roleHandler struct {
	roles *role.Store
	sops  *sop.Store
}

// GET /api/roles
func (h *roleHandler) list(w http.ResponseWriter, r *http.Request) {
	roles, err := h.roles.List()
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if roles == nil {
		roles = []role.Role{}
	}
	jsonOK(w, roles)
}

// GET /api/roles/:id
func (h *roleHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ro, err := h.roles.Get(id)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, ro)
}

// POST /api/roles
func (h *roleHandler) create(w http.ResponseWriter, r *http.Request) {
	var ro role.Role
	if err := json.NewDecoder(r.Body).Decode(&ro); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.roles.Create(&ro); err != nil {
		jsonErr(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, ro)
}

// PUT /api/roles/:id
func (h *roleHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ro role.Role
	if err := json.NewDecoder(r.Body).Decode(&ro); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	ro.ID = id
	if err := h.roles.Update(&ro); err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, ro)
}

// DELETE /api/roles/:id
func (h *roleHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.roles.Delete(id); err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/roles/:id/compile
func (h *roleHandler) compile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ro, err := h.roles.Get(id)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	allSOPs, err := h.sops.List()
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	result, err := sop.Compile(ro.ID, ro.Categories, ro.SOPs, allSOPs)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	jsonOK(w, result)
}
