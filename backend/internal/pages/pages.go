package pages

import (
	"log/slog"
	"net/http"
	"strings"

	"staffportal/internal/httpx"
	"staffportal/internal/store"
)

// The handlers here run only after the guard allowed the request. They do not
// decide access. Their one duty is to pass the person's Scope to every query.

// Bookings handles GET /bookings.
func Bookings(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, p store.Person) {
		rows, err := st.Bookings(r.Context(), p.Scope())
		if err != nil {
			fail(w, "bookings", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"page": "/bookings", "rows": rows})
	}
}

// Patients handles GET /patients.
func Patients(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, p store.Person) {
		rows, err := st.Patients(r.Context(), p.Scope())
		if err != nil {
			fail(w, "patients", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"page": "/patients", "rows": rows})
	}
}

// MedicalRecord handles GET /medical-record.
func MedicalRecord(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, p store.Person) {
		rows, err := st.MedicalRecords(r.Context(), p.Scope())
		if err != nil {
			fail(w, "medical-record", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"page": "/medical-record", "rows": rows})
	}
}

// Revenue handles GET /revenue: the only page that shows money.
func Revenue(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, p store.Person) {
		rows, err := st.Revenue(r.Context(), p.Scope())
		if err != nil {
			fail(w, "revenue", err)
			return
		}
		var total int64
		for _, row := range rows {
			total += row.TotalRupiah
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"page": "/revenue", "rows": rows, "total_rupiah": total,
		})
	}
}

// SalesEntryForm handles GET /sales-entry. No department has "read" on this page,
// so in practice the guard refuses it. If someone is ever given it, they get the
// empty form and nothing else: this page never shows sales or totals (M4).
func SalesEntryForm() httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, p store.Person) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"page": "/sales-entry", "fields": []string{"item", "amount_rupiah"},
		})
	}
}

// EnterSale handles POST /sales-entry: store one sale and answer with its ID only.
func EnterSale(st *store.Store) httpx.PageHandler {
	return func(w http.ResponseWriter, r *http.Request, p store.Person) {
		var in struct {
			Item         string `json:"item"`
			AmountRupiah int64  `json:"amount_rupiah"`
		}
		if err := httpx.ReadJSON(w, r, &in); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "bad_request")
			return
		}
		in.Item = strings.TrimSpace(in.Item)
		if in.Item == "" || in.AmountRupiah <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "item_and_positive_amount_required")
			return
		}
		id, err := st.InsertSale(r.Context(), p, in.Item, in.AmountRupiah)
		if err != nil {
			fail(w, "sales-entry", err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// fail logs the real error for us and shows the client only a code.
func fail(w http.ResponseWriter, page string, err error) {
	slog.Error("page failed", "page", page, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error")
}
