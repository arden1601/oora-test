package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"staffportal/internal/access"
)

// Session limits. One rule for everyone.
const (
	sessionIdleLimit = "15 minutes" // signed out after this long without a request
	sessionMaxAge    = "12 hours"   // signed out this long after signing in, whatever happens
)

// Person is the signed-in member of staff, read fresh from the database on every request.
type Person struct {
	ID             string
	Name           string
	DepartmentID   string
	DepartmentName string
	Money          access.Money // the department's money setting
	BranchID       string
	BranchName     string
	HeadOffice     bool
	Active         bool
}

// Scope says which branches' rows a person may see. Every query on branch data
// takes a Scope, so no query can forget the branch rule (T3).
type Scope struct {
	AllBranches bool   // head office
	BranchID    string // otherwise only this branch
}

// Scope gives the rows this person may see: all branches for head office, else their own.
func (p Person) Scope() Scope {
	return Scope{AllBranches: p.HeadOffice, BranchID: p.BranchID}
}

// LoginRecord is what sign-in needs to check a password.
type LoginRecord struct {
	StaffID      string
	PasswordHash string
	Active       bool
}

// LoginRecordByEmail finds the person signing in. ErrNotFound if there is no such email.
func (s *Store) LoginRecordByEmail(ctx context.Context, email string) (LoginRecord, error) {
	var rec LoginRecord
	err := s.db.QueryRowContext(ctx,
		`SELECT id, password_hash, is_active FROM staff WHERE email = $1`, email).
		Scan(&rec.StaffID, &rec.PasswordHash, &rec.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return LoginRecord{}, ErrNotFound
	}
	if err != nil {
		return LoginRecord{}, fmt.Errorf("load login record: %w", err)
	}
	return rec, nil
}

// CreateSession stores a new session under the hash of its token.
func (s *Store) CreateSession(ctx context.Context, tokenHash, staffID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, staff_id, expires_at)
		VALUES ($1, $2, now() + $3::interval)`, tokenHash, staffID, sessionMaxAge)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// DeleteSession ends a session. Deleting one that is already gone is not an error.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// PersonBySession finds who a session belongs to and marks the session as just used.
// ErrNotFound if the session does not exist, has expired, or sat idle too long.
func (s *Store) PersonBySession(ctx context.Context, tokenHash string) (Person, error) {
	var staffID string
	err := s.db.QueryRowContext(ctx, `
		UPDATE sessions SET last_seen_at = now()
		WHERE token_hash = $1
		  AND expires_at > now()
		  AND last_seen_at > now() - $2::interval
		RETURNING staff_id`, tokenHash, sessionIdleLimit).Scan(&staffID)
	if errors.Is(err, sql.ErrNoRows) {
		return Person{}, ErrNotFound
	}
	if err != nil {
		return Person{}, fmt.Errorf("load session: %w", err)
	}

	var p Person
	err = s.db.QueryRowContext(ctx, `
		SELECT s.id, s.display_name, s.is_active, d.id, d.name, d.money, b.id, b.name, b.is_head_office
		FROM staff s
		JOIN departments d ON d.id = s.department_id
		JOIN branches b ON b.id = s.branch_id
		WHERE s.id = $1`, staffID).
		Scan(&p.ID, &p.Name, &p.Active, &p.DepartmentID, &p.DepartmentName, &p.Money,
			&p.BranchID, &p.BranchName, &p.HeadOffice)
	if err != nil {
		return Person{}, fmt.Errorf("load staff: %w", err)
	}
	return p, nil
}

// Facts loads what the access decision needs for one person, page and action.
// It runs on every request, so a revoked grant counts at once (T4).
func (s *Store) Facts(ctx context.Context, p Person, page string, action access.Action) (access.Facts, error) {
	f := access.Facts{Active: p.Active, Money: p.Money}
	var showsMoney sql.NullBool // empty when the page is not in the pages table
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT shows_money FROM pages WHERE path = $1),
			EXISTS (SELECT 1 FROM department_pages
			        WHERE department_id = $2 AND page_path = $1 AND mode = $3),
			EXISTS (SELECT 1 FROM grants
			        WHERE staff_id = $4 AND kind = 'page' AND page_path = $1 AND mode = $3
			          AND revoked_at IS NULL),
			EXISTS (SELECT 1 FROM grants
			        WHERE staff_id = $4 AND kind = 'money' AND revoked_at IS NULL)`,
		page, p.DepartmentID, string(action), p.ID).
		Scan(&showsMoney, &f.DepartmentHasPage, &f.PageGrant, &f.MoneyGrant)
	if err != nil {
		return access.Facts{}, fmt.Errorf("load access facts: %w", err)
	}
	// An unknown page is treated as a money page: the safer of the two answers.
	f.ShowsMoney = !showsMoney.Valid || showsMoney.Bool
	return f, nil
}

// Page is one row of the pages table.
type Page struct {
	Path       string
	ShowsMoney bool
}

// ListPages lists every page the portal knows, in path order.
func (s *Store) ListPages(ctx context.Context) ([]Page, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, shows_money FROM pages ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	defer rows.Close()

	out := []Page{}
	for rows.Next() {
		var pg Page
		if err := rows.Scan(&pg.Path, &pg.ShowsMoney); err != nil {
			return nil, fmt.Errorf("read page: %w", err)
		}
		out = append(out, pg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	return out, nil
}

// InsertAudit records one refused access. staffID is empty when nobody was signed in.
func (s *Store) InsertAudit(ctx context.Context, staffID, page, action, reason string) error {
	var who any // NULL when empty
	if staffID != "" {
		who = staffID
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_log (staff_id, page_path, action, reason_code)
		VALUES ($1, $2, $3, $4)`, who, page, action, reason)
	if err != nil {
		return fmt.Errorf("insert audit row: %w", err)
	}
	return nil
}
