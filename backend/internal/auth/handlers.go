package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"staffportal/internal/store"
)

// CookieName is the name of the session cookie.
const CookieName = "session"

// A hash that matches no password. Checking against it when the email is unknown
// makes a wrong email take as long as a wrong password, so timing gives nothing away.
const dummyHash = "pbkdf2-sha256$600000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// SessionHash reads the session cookie and returns the hash it is stored under.
func SessionHash(r *http.Request) (string, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return HashSessionToken(c.Value), true
}

// Login handles POST /login: email and password in, session cookie out.
// secureCookie must be true wherever the site is served over HTTPS.
func Login(st *store.Store, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request")
			return
		}

		rec, err := st.LoginRecordByEmail(r.Context(), in.Email)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			slog.Error("sign-in failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		found := err == nil
		hash := dummyHash
		if found {
			hash = rec.PasswordHash
		}
		// One answer for a wrong email, a wrong password and a person who has left.
		if !CheckPassword(hash, in.Password) || !found || !rec.Active {
			// Neither the email typed nor the password is logged or stored.
			slog.Info("sign-in refused", "reason", "invalid_credentials")
			if err := st.InsertAudit(r.Context(), "", "/login", "sign_in", "invalid_credentials"); err != nil {
				slog.Error("audit failed", "error", err)
			}
			writeError(w, http.StatusUnauthorized, "invalid_credentials")
			return
		}

		token, tokenHash, err := NewSessionToken()
		if err != nil {
			slog.Error("sign-in failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if err := st.CreateSession(r.Context(), tokenHash, rec.StaffID); err != nil {
			slog.Error("sign-in failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     CookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true, // page scripts cannot read it
			Secure:   secureCookie,
			SameSite: http.SameSiteLaxMode, // not sent on requests started by other sites
		})
		slog.Info("signed in", "staff_id", rec.StaffID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// Logout handles POST /logout: the session is deleted on the server, so the old
// cookie is worthless at once.
func Logout(st *store.Store, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hash, ok := SessionHash(r); ok {
			if err := st.DeleteSession(r.Context(), hash); err != nil {
				slog.Error("sign-out failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
		}
		http.SetCookie(w, &http.Cookie{
			Name: CookieName, Value: "", Path: "/", MaxAge: -1,
			HttpOnly: true, Secure: secureCookie, SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code})
}
