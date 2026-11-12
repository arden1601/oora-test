package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// NewGrant is a request to give one person one page, or "may see money".
type NewGrant struct {
	StaffID   string // who receives it
	Kind      string // "page" or "money"
	Page      string // for kind "page"
	Mode      string // for kind "page": "read" or "enter"
	Reason    string // why; never empty
	GrantedBy string // the admin giving it
}

// GrantTargetsExist checks that the person and (for a page grant) the page exist,
// so a bad request is answered clearly instead of failing inside the insert.
func (s *Store) GrantTargetsExist(ctx context.Context, staffID, page string) (staffOK, pageOK bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM staff WHERE id = $1),
		       EXISTS (SELECT 1 FROM pages WHERE path = $2)`, staffID, page).Scan(&staffOK, &pageOK)
	if err != nil {
		return false, false, fmt.Errorf("check grant targets: %w", err)
	}
	return staffOK, pageOK, nil
}

// CreateGrant records a grant with who gave it, when and why, and returns its ID.
func (s *Store) CreateGrant(ctx context.Context, g NewGrant) (string, error) {
	var page, mode any // NULL for a money grant
	if g.Kind == "page" {
		page, mode = g.Page, g.Mode
	}
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO grants (staff_id, kind, page_path, mode, reason, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`, g.StaffID, g.Kind, page, mode, g.Reason, g.GrantedBy).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert grant: %w", err)
	}
	return id, nil
}

// RevokeGrant takes a grant away. The row is kept: only revoked_by and revoked_at
// are filled in. ErrNotFound if there is no such active grant.
func (s *Store) RevokeGrant(ctx context.Context, grantID, revokedBy string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE grants SET revoked_by = $2, revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL`, grantID, revokedBy)
	if err != nil {
		return fmt.Errorf("revoke grant: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke grant: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeactivateStaff marks a person as having left. Their next request is refused,
// because the staff row is read on every request. ErrNotFound if there is no such person.
func (s *Store) DeactivateStaff(ctx context.Context, staffID string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE staff SET is_active = false WHERE id = $1`, staffID)
	if err != nil {
		return fmt.Errorf("deactivate staff: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("deactivate staff: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// StaffRow is one line of the staff list on the admin screen.
type StaffRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	Branch     string `json:"branch"`
	Active     bool   `json:"active"`
}

// ListStaff lists everyone, for the admin screen.
func (s *Store) ListStaff(ctx context.Context) ([]StaffRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.display_name, d.name, b.name, s.is_active
		FROM staff s
		JOIN departments d ON d.id = s.department_id
		JOIN branches b ON b.id = s.branch_id
		ORDER BY d.name, s.display_name`)
	if err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	defer rows.Close()

	out := []StaffRow{}
	for rows.Next() {
		var r StaffRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Department, &r.Branch, &r.Active); err != nil {
			return nil, fmt.Errorf("read staff: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	return out, nil
}

// PageAccess is one row of the permission table.
type PageAccess struct {
	Page string `json:"page"`
	Mode string `json:"mode"`
}

// GrantRow is one grant in a person's history, active or taken away.
type GrantRow struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Page      string     `json:"page,omitempty"`
	Mode      string     `json:"mode,omitempty"`
	Reason    string     `json:"reason"`
	GrantedBy string     `json:"granted_by"` // staff ID, not a name
	GrantedAt time.Time  `json:"granted_at"`
	RevokedBy string     `json:"revoked_by,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// StaffAccess is everything the admin screen shows about one person's access.
type StaffAccess struct {
	StaffRow
	Money           string       `json:"money"`
	DepartmentPages []PageAccess `json:"department_pages"`
	Grants          []GrantRow   `json:"grants"`
}

// AccessOf collects one person's access: what the department gives, plus every
// grant with its history. ErrNotFound if there is no such person.
func (s *Store) AccessOf(ctx context.Context, staffID string) (StaffAccess, error) {
	var a StaffAccess
	var departmentID string
	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.display_name, d.name, b.name, s.is_active, d.money, d.id
		FROM staff s
		JOIN departments d ON d.id = s.department_id
		JOIN branches b ON b.id = s.branch_id
		WHERE s.id = $1`, staffID).
		Scan(&a.ID, &a.Name, &a.Department, &a.Branch, &a.Active, &a.Money, &departmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return StaffAccess{}, ErrNotFound
	}
	if err != nil {
		return StaffAccess{}, fmt.Errorf("load staff: %w", err)
	}

	a.DepartmentPages = []PageAccess{}
	pageRows, err := s.db.QueryContext(ctx, `
		SELECT page_path, mode FROM department_pages
		WHERE department_id = $1 ORDER BY page_path, mode`, departmentID)
	if err != nil {
		return StaffAccess{}, fmt.Errorf("list department pages: %w", err)
	}
	defer pageRows.Close()
	for pageRows.Next() {
		var pa PageAccess
		if err := pageRows.Scan(&pa.Page, &pa.Mode); err != nil {
			return StaffAccess{}, fmt.Errorf("read department page: %w", err)
		}
		a.DepartmentPages = append(a.DepartmentPages, pa)
	}
	if err := pageRows.Err(); err != nil {
		return StaffAccess{}, fmt.Errorf("list department pages: %w", err)
	}

	a.Grants = []GrantRow{}
	grantRows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, coalesce(page_path, ''), coalesce(mode, ''), reason,
		       granted_by, granted_at, coalesce(revoked_by::text, ''), revoked_at
		FROM grants WHERE staff_id = $1 ORDER BY granted_at DESC`, staffID)
	if err != nil {
		return StaffAccess{}, fmt.Errorf("list grants: %w", err)
	}
	defer grantRows.Close()
	for grantRows.Next() {
		var g GrantRow
		if err := grantRows.Scan(&g.ID, &g.Kind, &g.Page, &g.Mode, &g.Reason,
			&g.GrantedBy, &g.GrantedAt, &g.RevokedBy, &g.RevokedAt); err != nil {
			return StaffAccess{}, fmt.Errorf("read grant: %w", err)
		}
		a.Grants = append(a.Grants, g)
	}
	if err := grantRows.Err(); err != nil {
		return StaffAccess{}, fmt.Errorf("list grants: %w", err)
	}
	return a, nil
}
