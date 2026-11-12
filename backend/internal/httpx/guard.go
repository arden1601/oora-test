package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"staffportal/internal/access"
	"staffportal/internal/audit"
	"staffportal/internal/auth"
	"staffportal/internal/store"
)

// PageHandler is a handler that runs only after access was allowed. It receives
// the signed-in person, whose Scope every data query must use.
type PageHandler func(w http.ResponseWriter, r *http.Request, p store.Person)

// Guard puts the access decision in front of a page. Every route except sign-in
// goes through it, on every request (T1).
type Guard struct {
	Store *store.Store
}

// Page wraps a handler: it finds the signed-in person, asks the access decision
// for this page and action, and either runs the handler or answers 401 / 403.
func (g *Guard) Page(page string, action access.Action, next PageHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Who is asking? The session and the staff row are read fresh each time.
		tokenHash, ok := auth.SessionHash(r)
		if !ok {
			audit.Refused(ctx, g.Store, "", page, string(action), "not_signed_in")
			WriteError(w, http.StatusUnauthorized, "not_signed_in")
			return
		}
		person, err := g.Store.PersonBySession(ctx, tokenHash)
		if errors.Is(err, store.ErrNotFound) {
			audit.Refused(ctx, g.Store, "", page, string(action), "not_signed_in")
			WriteError(w, http.StatusUnauthorized, "not_signed_in")
			return
		}
		if err != nil {
			slog.Error("load person failed", "error", err)
			WriteError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// May they? The decision itself lives in internal/access and nowhere else.
		facts, err := g.Store.Facts(ctx, person, page, action)
		if err != nil {
			slog.Error("load access facts failed", "error", err)
			WriteError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		decision := access.Decide(facts)
		if !decision.Allowed {
			audit.Refused(ctx, g.Store, person.ID, page, string(action), decision.Reason)
			WriteError(w, http.StatusForbidden, "forbidden")
			return
		}

		next(w, r, person)
	}
}

// WriteJSON sends v as a JSON answer.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response failed", "error", err)
	}
}

// WriteError sends a short error code. Raw errors are never shown to the client.
func WriteError(w http.ResponseWriter, status int, code string) {
	WriteJSON(w, status, map[string]string{"error": code})
}

// ReadJSON reads a small JSON request body into v.
func ReadJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	return json.NewDecoder(r.Body).Decode(v)
}
