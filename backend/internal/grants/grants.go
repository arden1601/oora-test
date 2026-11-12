package grants

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"staffportal/internal/httpx"
	"staffportal/internal/store"
)

// The admin API. These handlers run only after the guard allowed the request on
// the "/admin" page, so who counts as an admin is decided by the permission
// table, not here.

// Create handles POST /admin/grants: give one person one page, or "may see money".
func Create(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, admin store.Person) {
		var in struct {
			StaffID string `json:"staff_id"`
			Kind    string `json:"kind"`
			Page    string `json:"page"`
			Mode    string `json:"mode"`
			Reason  string `json:"reason"`
		}
		if err := httpx.ReadJSON(w, r, &in); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "bad_request")
			return
		}
		in.Reason = strings.TrimSpace(in.Reason)

		// Every grant must say why (M5).
		if in.Reason == "" {
			httpx.WriteError(w, http.StatusBadRequest, "reason_required")
			return
		}
		if !store.IsID(in.StaffID) {
			httpx.WriteError(w, http.StatusBadRequest, "unknown_staff")
			return
		}
		switch in.Kind {
		case "page":
			if in.Mode != "read" && in.Mode != "enter" {
				httpx.WriteError(w, http.StatusBadRequest, "mode_must_be_read_or_enter")
				return
			}
		case "money":
			in.Page, in.Mode = "", ""
		default:
			httpx.WriteError(w, http.StatusBadRequest, "kind_must_be_page_or_money")
			return
		}

		staffOK, pageOK, err := st.GrantTargetsExist(r.Context(), in.StaffID, in.Page)
		if err != nil {
			fail(w, err)
			return
		}
		if !staffOK {
			httpx.WriteError(w, http.StatusBadRequest, "unknown_staff")
			return
		}
		if in.Kind == "page" && !pageOK {
			httpx.WriteError(w, http.StatusBadRequest, "unknown_page")
			return
		}

		id, err := st.CreateGrant(r.Context(), store.NewGrant{
			StaffID: in.StaffID, Kind: in.Kind, Page: in.Page, Mode: in.Mode,
			Reason: in.Reason, GrantedBy: admin.ID,
		})
		if err != nil {
			fail(w, err)
			return
		}
		slog.Info("grant given", "grant_id", id, "staff_id", in.StaffID, "kind", in.Kind, "by", admin.ID)
		httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// Revoke handles DELETE /admin/grants/{id}: take a grant away. The row is kept.
func Revoke(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, admin store.Person) {
		id := r.PathValue("id")
		if !store.IsID(id) {
			httpx.WriteError(w, http.StatusNotFound, "not_found")
			return
		}
		err := st.RevokeGrant(r.Context(), id, admin.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found")
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		slog.Info("grant taken away", "grant_id", id, "by", admin.ID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListStaff handles GET /admin/staff: everyone, for the admin screen.
func ListStaff(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, admin store.Person) {
		rows, err := st.ListStaff(r.Context())
		if err != nil {
			fail(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"rows": rows})
	}
}

// AccessOf handles GET /admin/staff/{id}/access: one person's access and grant history.
func AccessOf(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, admin store.Person) {
		id := r.PathValue("id")
		if !store.IsID(id) {
			httpx.WriteError(w, http.StatusNotFound, "not_found")
			return
		}
		a, err := st.AccessOf(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found")
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, a)
	}
}

// Deactivate handles POST /admin/staff/{id}/deactivate: the person has left.
func Deactivate(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, admin store.Person) {
		id := r.PathValue("id")
		if !store.IsID(id) {
			httpx.WriteError(w, http.StatusNotFound, "not_found")
			return
		}
		err := st.DeactivateStaff(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found")
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		slog.Info("staff deactivated", "staff_id", id, "by", admin.ID)
		w.WriteHeader(http.StatusNoContent)
	}
}

func fail(w http.ResponseWriter, err error) {
	slog.Error("admin request failed", "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error")
}
