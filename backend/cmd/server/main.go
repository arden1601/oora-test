// Command server starts the staff portal HTTP service.
//
// main only wires things together. Rules and logic live in internal/.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"staffportal/api"
	"staffportal/internal/access"
	"staffportal/internal/auth"
	"staffportal/internal/grants"
	"staffportal/internal/httpx"
	"staffportal/internal/pages"
	"staffportal/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := store.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("database not reachable", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := store.Migrate(ctx, db); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newMux(db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("server starting", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// newMux registers every route. Sign-in is open; every other route names its
// page and action and goes through the guard, which asks the access decision.
func newMux(db *sql.DB) *http.ServeMux {
	st := store.New(db)
	guard := &httpx.Guard{Store: st}
	// Set COOKIE_SECURE=true wherever the site is served over HTTPS.
	secure := os.Getenv("COOKIE_SECURE") == "true"

	const read, enter = access.ActionRead, access.ActionEnter

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)

	// API documentation at /docs. Set API_DOCS=false to switch it off in production.
	if os.Getenv("API_DOCS") != "false" {
		api.Register(mux)
	}

	mux.HandleFunc("POST /login", auth.Login(st, secure))
	mux.HandleFunc("POST /logout", auth.Logout(st, secure))

	// Who am I, and which pages may I open? A page like the others, behind the guard.
	mux.HandleFunc("GET /me", guard.Page("/me", read, guard.Me()))

	// The five pages.
	mux.HandleFunc("GET /bookings", guard.Page("/bookings", read, pages.Bookings(st)))
	mux.HandleFunc("GET /patients", guard.Page("/patients", read, pages.Patients(st)))
	mux.HandleFunc("GET /medical-record", guard.Page("/medical-record", read, pages.MedicalRecord(st)))
	mux.HandleFunc("GET /revenue", guard.Page("/revenue", read, pages.Revenue(st)))
	mux.HandleFunc("GET /sales-entry", guard.Page("/sales-entry", read, pages.SalesEntryForm()))
	mux.HandleFunc("POST /sales-entry", guard.Page("/sales-entry", enter, pages.EnterSale(st)))

	// The admin API: look with "read", change with "enter", both on the "/admin" page.
	mux.HandleFunc("GET /admin/staff", guard.Page("/admin", read, grants.ListStaff(st)))
	mux.HandleFunc("GET /admin/staff/{id}/access", guard.Page("/admin", read, grants.AccessOf(st)))
	mux.HandleFunc("POST /admin/staff/{id}/deactivate", guard.Page("/admin", enter, grants.Deactivate(st)))
	mux.HandleFunc("POST /admin/grants", guard.Page("/admin", enter, grants.Create(st)))
	mux.HandleFunc("DELETE /admin/grants/{id}", guard.Page("/admin", enter, grants.Revoke(st)))

	return mux
}

// handleHealthz tells a load balancer or reviewer that the process is up.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("ok\n"))
}
