package store

import (
	"context"
	"fmt"
	"time"
)

// Every query here takes a Scope and filters on it:
//
//	WHERE ($1 OR branch_id = $2)
//
// $1 is true for head office (all branches); otherwise only branch $2 matches.

// Booking is one row on the bookings page.
type Booking struct {
	ID        string    `json:"id"`
	Branch    string    `json:"branch"`
	Customer  string    `json:"customer"`
	Treatment string    `json:"treatment"`
	StartsAt  time.Time `json:"starts_at"`
	Status    string    `json:"status"`
}

// Bookings lists the bookings the scope allows.
func (s *Store) Bookings(ctx context.Context, scope Scope) ([]Booking, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT bk.id, br.name, p.display_name, bk.treatment, bk.starts_at, bk.status
		FROM bookings bk
		JOIN branches br ON br.id = bk.branch_id
		JOIN patients p ON p.id = bk.patient_id
		WHERE ($1 OR bk.branch_id = $2)
		ORDER BY bk.starts_at`, scope.AllBranches, scope.BranchID)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	out := []Booking{}
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.Branch, &b.Customer, &b.Treatment, &b.StartsAt, &b.Status); err != nil {
			return nil, fmt.Errorf("read booking: %w", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	return out, nil
}

// Patient is one row on the patients page: the card, without the medical record.
type Patient struct {
	ID     string `json:"id"`
	Branch string `json:"branch"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
}

// Patients lists the patient cards the scope allows.
func (s *Store) Patients(ctx context.Context, scope Scope) ([]Patient, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, br.name, p.display_name, p.phone
		FROM patients p
		JOIN branches br ON br.id = p.branch_id
		WHERE ($1 OR p.branch_id = $2)
		ORDER BY p.display_name`, scope.AllBranches, scope.BranchID)
	if err != nil {
		return nil, fmt.Errorf("list patients: %w", err)
	}
	defer rows.Close()

	out := []Patient{}
	for rows.Next() {
		var p Patient
		if err := rows.Scan(&p.ID, &p.Branch, &p.Name, &p.Phone); err != nil {
			return nil, fmt.Errorf("read patient: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list patients: %w", err)
	}
	return out, nil
}

// MedicalRecord is one row on the medical record page.
type MedicalRecord struct {
	ID      string `json:"id"`
	Branch  string `json:"branch"`
	Patient string `json:"patient"`
	Note    string `json:"note"`
}

// MedicalRecords lists the medical records the scope allows.
func (s *Store) MedicalRecords(ctx context.Context, scope Scope) ([]MedicalRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, br.name, p.display_name, m.note
		FROM medical_records m
		JOIN branches br ON br.id = m.branch_id
		JOIN patients p ON p.id = m.patient_id
		WHERE ($1 OR m.branch_id = $2)
		ORDER BY p.display_name`, scope.AllBranches, scope.BranchID)
	if err != nil {
		return nil, fmt.Errorf("list medical records: %w", err)
	}
	defer rows.Close()

	out := []MedicalRecord{}
	for rows.Next() {
		var m MedicalRecord
		if err := rows.Scan(&m.ID, &m.Branch, &m.Patient, &m.Note); err != nil {
			return nil, fmt.Errorf("read medical record: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list medical records: %w", err)
	}
	return out, nil
}

// BranchRevenue is the sales total of one branch, in whole rupiah.
type BranchRevenue struct {
	Branch      string `json:"branch"`
	TotalRupiah int64  `json:"total_rupiah"`
}

// Revenue gives the sales total per branch, for the branches the scope allows.
func (s *Store) Revenue(ctx context.Context, scope Scope) ([]BranchRevenue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT br.name, sum(sa.amount_rupiah)
		FROM sales sa
		JOIN branches br ON br.id = sa.branch_id
		WHERE ($1 OR sa.branch_id = $2)
		GROUP BY br.name
		ORDER BY br.name`, scope.AllBranches, scope.BranchID)
	if err != nil {
		return nil, fmt.Errorf("sum revenue: %w", err)
	}
	defer rows.Close()

	out := []BranchRevenue{}
	for rows.Next() {
		var r BranchRevenue
		if err := rows.Scan(&r.Branch, &r.TotalRupiah); err != nil {
			return nil, fmt.Errorf("read revenue: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sum revenue: %w", err)
	}
	return out, nil
}

// InsertSale stores one sale under the entering person's own branch and returns
// its ID. It returns nothing else: entering a sale never shows other sales or totals (M4).
func (s *Store) InsertSale(ctx context.Context, by Person, item string, amountRupiah int64) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO sales (branch_id, entered_by, item, amount_rupiah)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, by.BranchID, by.ID, item, amountRupiah).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert sale: %w", err)
	}
	return id, nil
}
