package main

// End-to-end tests for the rules in AGENTS.md (M1-M5, T1-T5). Each test starts
// from a freshly migrated and seeded database and talks to the real HTTP routes.
//
// They need PostgreSQL: run them with `make test`, which starts it.

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"staffportal/internal/store"
)

const demoPassword = "abra-demo-2026"

// Seeded people, by what they are.
const (
	management  = "management@example.com"
	growth      = "growth@example.com"
	finance     = "finance@example.com"
	marketing   = "marketing@example.com"
	hr          = "hr@example.com"
	partners    = "partnerships@example.com"
	crm         = "crm@example.com"
	clinicLead  = "clinic-lead@example.com" // Clinic Operations, Kemang
	frontDesk   = "frontdesk@example.com"   // Kemang
	consultant  = "consultant@example.com"
	doctor      = "doctor@example.com"
	frontDeskM  = "frontdesk-menteng@example.com"
	kemangTotal = 2_250_000
	// Kemang 2,250,000 + Menteng 900,000 + 500,000 in each of the 8 other clinics.
	allTotal    = 7_150_000
	allBookings = 12 // 2 in Kemang, 2 in Menteng, 1 in each of the 8 other clinics
)

// ---- Sign-in ----

func TestLoginWithRightPasswordSetsASessionCookie(t *testing.T) {
	p := newPortal(t)
	rec := p.post(nil, "/login", map[string]string{"email": frontDesk, "password": demoPassword})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login: got %d, want 204", rec.Code)
	}
	c := sessionCookie(rec)
	if c == nil {
		t.Fatal("login set no session cookie")
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie must be HttpOnly and SameSite=Lax, got HttpOnly=%v SameSite=%v", c.HttpOnly, c.SameSite)
	}
}

func TestLoginWithWrongPasswordIsRefused(t *testing.T) {
	p := newPortal(t)
	rec := p.post(nil, "/login", map[string]string{"email": frontDesk, "password": "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
	if sessionCookie(rec) != nil {
		t.Fatal("a failed login set a session cookie")
	}
}

func TestUnknownEmailGetsTheSameAnswerAsWrongPassword(t *testing.T) {
	p := newPortal(t)
	wrong := p.post(nil, "/login", map[string]string{"email": frontDesk, "password": "wrong"})
	unknown := p.post(nil, "/login", map[string]string{"email": "nobody@example.com", "password": "wrong"})
	if wrong.Code != unknown.Code || wrong.Body.String() != unknown.Body.String() {
		t.Fatalf("answers differ: %d %q vs %d %q", wrong.Code, wrong.Body, unknown.Code, unknown.Body)
	}
}

func TestPasswordsAreStoredOnlyAsHashes(t *testing.T) { // T5
	p := newPortal(t)
	rows, err := p.db.Query(`SELECT password_hash FROM staff`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if strings.Contains(hash, demoPassword) || !strings.HasPrefix(hash, "pbkdf2-sha256$") {
			t.Fatalf("stored value is not a hash: %q", hash)
		}
	}
}

func TestSessionIsStoredOnlyAsAHash(t *testing.T) {
	p := newPortal(t)
	c := p.login(frontDesk)
	var n int
	p.queryRow(&n, `SELECT count(*) FROM sessions WHERE token_hash = $1`, c.Value)
	if n != 0 {
		t.Fatal("the cookie value itself is stored in the sessions table")
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	p := newPortal(t)
	c := p.login(frontDesk)
	if rec := p.post(c, "/logout", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: got %d, want 204", rec.Code)
	}
	if rec := p.get(c, "/bookings"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after logout: got %d, want 401", rec.Code)
	}
}

func TestNoCookieIsRefused(t *testing.T) {
	p := newPortal(t)
	if rec := p.get(nil, "/bookings"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}

func TestIdleSessionIsRefused(t *testing.T) {
	p := newPortal(t)
	c := p.login(frontDesk)
	p.exec(`UPDATE sessions SET last_seen_at = now() - interval '16 minutes'`)
	if rec := p.get(c, "/bookings"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("idle 16 minutes: got %d, want 401", rec.Code)
	}
}

func TestSessionOlderThanTwelveHoursIsRefused(t *testing.T) {
	p := newPortal(t)
	c := p.login(frontDesk)
	p.exec(`UPDATE sessions SET expires_at = now() - interval '1 minute'`)
	if rec := p.get(c, "/bookings"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired: got %d, want 401", rec.Code)
	}
}

// ---- The permission table ----

func TestPermissionTable(t *testing.T) { // T1, T2: every cell of the table in the design
	p := newPortal(t)
	type cell struct {
		method, path string
		want         int
	}
	const ok, no = http.StatusOK, http.StatusForbidden
	sale := http.StatusCreated
	table := map[string][]cell{
		management: {{"GET", "/bookings", ok}, {"GET", "/patients", ok}, {"GET", "/medical-record", ok}, {"POST", "/sales-entry", sale}, {"GET", "/revenue", ok}},
		growth:     {{"GET", "/bookings", ok}, {"GET", "/patients", ok}, {"GET", "/medical-record", ok}, {"POST", "/sales-entry", sale}, {"GET", "/revenue", ok}},
		finance:    {{"GET", "/bookings", no}, {"GET", "/patients", no}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", ok}},
		crm:        {{"GET", "/bookings", ok}, {"GET", "/patients", no}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
		clinicLead: {{"GET", "/bookings", ok}, {"GET", "/patients", ok}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
		frontDesk:  {{"GET", "/bookings", ok}, {"GET", "/patients", ok}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", sale}, {"GET", "/revenue", no}},
		consultant: {{"GET", "/bookings", no}, {"GET", "/patients", ok}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
		doctor:     {{"GET", "/bookings", no}, {"GET", "/patients", ok}, {"GET", "/medical-record", ok}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
		marketing:  {{"GET", "/bookings", no}, {"GET", "/patients", no}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
		hr:         {{"GET", "/bookings", no}, {"GET", "/patients", no}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
		partners:   {{"GET", "/bookings", no}, {"GET", "/patients", no}, {"GET", "/medical-record", no}, {"POST", "/sales-entry", no}, {"GET", "/revenue", no}},
	}
	for email, cells := range table {
		c := p.login(email)
		for _, cell := range cells {
			var rec *httptest.ResponseRecorder
			if cell.method == "POST" {
				rec = p.post(c, cell.path, map[string]any{"item": "Facial", "amount_rupiah": 100000})
			} else {
				rec = p.get(c, cell.path)
			}
			if rec.Code != cell.want {
				t.Errorf("%s %s %s: got %d, want %d", email, cell.method, cell.path, rec.Code, cell.want)
			}
		}
	}
}

func TestManagementAndGrowthHaveARowForEveryPage(t *testing.T) { // M1
	// "See everything" is plain data: a new page without these rows fails here.
	p := newPortal(t)
	var missing int
	p.queryRow(&missing, `
		SELECT count(*) FROM pages pg, departments d
		WHERE d.id IN ('management', 'growth')
		  AND NOT EXISTS (SELECT 1 FROM department_pages dp
		                  WHERE dp.department_id = d.id AND dp.page_path = pg.path)`)
	if missing != 0 {
		t.Fatalf("%d page/department pairs have no row for Management or Growth", missing)
	}
}

// ---- Money ----

func TestDoctorNeverSeesRevenueEvenWithGrants(t *testing.T) { // M2
	p := newPortal(t)
	admin := p.login(management)
	p.grantPage(admin, doctor, "/revenue", "read")
	p.grantMoney(admin, doctor)
	if rec := p.get(p.login(doctor), "/revenue"); rec.Code != http.StatusForbidden {
		t.Fatalf("doctor with both grants: got %d, want 403", rec.Code)
	}
}

func TestFrontDeskNeverSeesRevenueEvenWithGrants(t *testing.T) { // M4
	p := newPortal(t)
	admin := p.login(management)
	p.grantPage(admin, frontDesk, "/revenue", "read")
	p.grantMoney(admin, frontDesk)
	if rec := p.get(p.login(frontDesk), "/revenue"); rec.Code != http.StatusForbidden {
		t.Fatalf("front desk with both grants: got %d, want 403", rec.Code)
	}
}

func TestRegularStaffNeedBothGrantsToSeeRevenue(t *testing.T) { // M3
	p := newPortal(t)
	admin := p.login(management)
	lead := p.login(clinicLead)

	p.grantPage(admin, clinicLead, "/revenue", "read")
	if rec := p.get(lead, "/revenue"); rec.Code != http.StatusForbidden {
		t.Fatalf("page grant only: got %d, want 403", rec.Code)
	}

	p.grantMoney(admin, clinicLead)
	if rec := p.get(lead, "/revenue"); rec.Code != http.StatusOK {
		t.Fatalf("page and money grant: got %d, want 200", rec.Code)
	}
}

func TestFrontDeskEntersASaleButGetsNoTotalsBack(t *testing.T) { // M4
	p := newPortal(t)
	c := p.login(frontDesk)

	rec := p.post(c, "/sales-entry", map[string]any{"item": "Facial", "amount_rupiah": 500000})
	if rec.Code != http.StatusCreated {
		t.Fatalf("enter a sale: got %d, want 201", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if len(body) != 1 || body["id"] == nil {
		t.Fatalf("the answer must hold only the new sale's id, got %v", body)
	}

	if rec := p.get(c, "/sales-entry"); rec.Code != http.StatusForbidden {
		t.Fatalf("reading the sales entry page: got %d, want 403", rec.Code)
	}
}

func TestASaleIsStoredUnderTheEnteringPersonsBranch(t *testing.T) { // T3
	p := newPortal(t)
	p.post(p.login(frontDeskM), "/sales-entry", map[string]any{"item": "Facial", "amount_rupiah": 100000})
	var branch string
	p.queryRow(&branch, `SELECT b.name FROM sales s JOIN branches b ON b.id = s.branch_id
		WHERE s.amount_rupiah = 100000`)
	if branch != "Abra Menteng" {
		t.Fatalf("sale stored under %q, want Abra Menteng", branch)
	}
}

func TestSaleWithBadAmountIsRejected(t *testing.T) {
	p := newPortal(t)
	rec := p.post(p.login(frontDesk), "/sales-entry", map[string]any{"item": "Facial", "amount_rupiah": 0})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
}

// ---- Branches ----

func TestBranchStaffSeeOnlyTheirOwnBranch(t *testing.T) { // T3
	p := newPortal(t)
	rows := p.rows(p.get(p.login(frontDesk), "/patients"))
	if len(rows) != 2 {
		t.Fatalf("Kemang front desk sees %d patients, want 2", len(rows))
	}
	for _, row := range rows {
		if row["branch"] != "Abra Kemang" {
			t.Fatalf("Kemang front desk sees a patient of %v", row["branch"])
		}
	}
}

func TestHeadOfficeSeesAllBranches(t *testing.T) { // T3
	p := newPortal(t)
	rows := p.rows(p.get(p.login(crm), "/bookings"))
	if len(rows) != allBookings {
		t.Fatalf("head office sees %d bookings, want all %d", len(rows), allBookings)
	}
	branches := map[any]bool{}
	for _, row := range rows {
		branches[row["branch"]] = true
	}
	if len(branches) != 10 {
		t.Fatalf("head office sees bookings of %d clinics, want all 10", len(branches))
	}
}

func TestRevenueTotalsFollowTheBranch(t *testing.T) { // T3, M3
	p := newPortal(t)
	if got := p.total(p.get(p.login(finance), "/revenue")); got != allTotal {
		t.Fatalf("finance sees total %d, want %d", got, allTotal)
	}

	admin := p.login(management)
	p.grantPage(admin, clinicLead, "/revenue", "read")
	p.grantMoney(admin, clinicLead)
	if got := p.total(p.get(p.login(clinicLead), "/revenue")); got != kemangTotal {
		t.Fatalf("Kemang clinic lead sees total %d, want only Kemang's %d", got, kemangTotal)
	}
}

// ---- Ten branches ----

func TestSeedHasTenClinicsAndOneHeadOffice(t *testing.T) { // T3
	p := newPortal(t)
	var clinics, headOffices int
	p.queryRow2(&clinics, &headOffices, `
		SELECT count(*) FILTER (WHERE NOT is_head_office), count(*) FILTER (WHERE is_head_office)
		FROM branches`)
	if clinics != 10 || headOffices != 1 {
		t.Fatalf("got %d clinics and %d head offices, want 10 and 1", clinics, headOffices)
	}
}

func TestEveryClinicHasOnePersonInEachClinicDepartment(t *testing.T) {
	p := newPortal(t)
	// Every clinic must hold exactly one person in each of the four clinic departments.
	var wrong int
	p.queryRow(&wrong, `
		SELECT count(*)
		FROM branches b
		CROSS JOIN (VALUES ('clinic_ops'), ('front_desk'), ('consultants'), ('doctor')) AS d (id)
		WHERE NOT b.is_head_office
		  AND (SELECT count(*) FROM staff s
		       WHERE s.branch_id = b.id AND s.department_id = d.id) <> 1`)
	if wrong != 0 {
		t.Fatalf("%d clinic/department pairs do not have exactly one person", wrong)
	}
	var clinicStaff int
	p.queryRow(&clinicStaff, `
		SELECT count(*) FROM staff s JOIN branches b ON b.id = s.branch_id WHERE NOT b.is_head_office`)
	if clinicStaff != 40 {
		t.Fatalf("got %d clinic staff, want 40 (4 in each of 10 clinics)", clinicStaff)
	}
}

func TestHeadOfficeHasOnePersonInEachHeadOfficeDepartment(t *testing.T) {
	p := newPortal(t)
	var people, departments int
	p.queryRow2(&people, &departments, `
		SELECT count(*), count(DISTINCT s.department_id)
		FROM staff s JOIN branches b ON b.id = s.branch_id WHERE b.is_head_office`)
	if people != 7 || departments != 7 {
		t.Fatalf("head office has %d people in %d departments, want 7 in 7", people, departments)
	}
}

func TestEveryClinicFrontDeskSeesOnlyItsOwnBranch(t *testing.T) { // T3
	p := newPortal(t)
	rows, err := p.db.Query(`
		SELECT s.email, b.name FROM staff s JOIN branches b ON b.id = s.branch_id
		WHERE s.department_id = 'front_desk' ORDER BY b.name`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	desks := map[string]string{} // email -> branch name
	for rows.Next() {
		var email, branch string
		if err := rows.Scan(&email, &branch); err != nil {
			t.Fatalf("scan: %v", err)
		}
		desks[email] = branch
	}
	rows.Close()
	if len(desks) != 10 {
		t.Fatalf("got %d front desks, want one in each of 10 clinics", len(desks))
	}

	for email, branch := range desks {
		c := p.login(email)
		for _, page := range []string{"/patients", "/bookings"} {
			got := p.rows(p.get(c, page))
			if len(got) == 0 {
				t.Errorf("%s sees nothing on %s, want the rows of %s", email, page, branch)
			}
			for _, row := range got {
				if row["branch"] != branch {
					t.Errorf("%s (%s) sees a %s row of %v", email, branch, page, row["branch"])
				}
			}
		}
	}
}

// ---- /me: what the signed-in person may open ----

func TestMeListsWhatThePersonMayOpen(t *testing.T) { // T1, T2
	p := newPortal(t)
	me := p.me(p.login(frontDesk))
	if me.Department != "Front Desk" || me.Branch != "Abra Kemang" || me.HeadOffice {
		t.Fatalf("who: %+v", me)
	}
	want := map[string][2]bool{ // page -> {read, enter}
		"/bookings":       {true, false},
		"/patients":       {true, false},
		"/medical-record": {false, false},
		"/sales-entry":    {false, true},
		"/revenue":        {false, false},
		"/admin":          {false, false},
		"/me":             {true, false},
	}
	if len(me.Pages) != len(want) {
		t.Fatalf("got %d pages, want %d", len(me.Pages), len(want))
	}
	for _, pg := range me.Pages {
		if w, ok := want[pg.Page]; !ok || pg.Read != w[0] || pg.Enter != w[1] {
			t.Errorf("%s: read=%v enter=%v, want %v", pg.Page, pg.Read, pg.Enter, w)
		}
	}
}

func TestMeChangesOnTheNextRequestAfterGrantAndRevoke(t *testing.T) { // M5, T4
	p := newPortal(t)
	admin := p.login(management)
	m := p.login(marketing)
	if p.me(m).mayRead("/bookings") {
		t.Fatal("before the grant: /me offers /bookings to marketing")
	}
	id := p.grantPage(admin, marketing, "/bookings", "read")
	if !p.me(m).mayRead("/bookings") {
		t.Fatal("after the grant: /me does not offer /bookings")
	}
	p.do(admin, "DELETE", "/admin/grants/"+id, nil)
	if p.me(m).mayRead("/bookings") {
		t.Fatal("after the revoke: /me still offers /bookings")
	}
}

func TestMeNeverOffersRevenueToADoctorEvenWithGrants(t *testing.T) { // M2
	p := newPortal(t)
	admin := p.login(management)
	p.grantPage(admin, doctor, "/revenue", "read")
	p.grantMoney(admin, doctor)
	if p.me(p.login(doctor)).mayRead("/revenue") {
		t.Fatal("/me offers /revenue to a doctor")
	}
}

func TestMeIsRefusedForADeactivatedPerson(t *testing.T) { // T4
	p := newPortal(t)
	fd := p.login(frontDesk)
	p.post(p.login(management), "/admin/staff/"+p.staffID(frontDesk)+"/deactivate", nil)
	if rec := p.get(fd, "/me"); rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestMeWithoutASessionIs401(t *testing.T) { // T1
	p := newPortal(t)
	if rec := p.get(nil, "/me"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}

func TestMeWritesNoAuditRows(t *testing.T) { // T1: a list of pages is not a refused access
	p := newPortal(t)
	p.me(p.login(doctor))
	var n int
	p.queryRow(&n, `SELECT count(*) FROM audit_log`)
	if n != 0 {
		t.Fatalf("got %d audit rows, want 0", n)
	}
}

// ---- Grants ----

func TestGrantOpensAPageOnTheNextRequest(t *testing.T) { // M5
	p := newPortal(t)
	m := p.login(marketing)
	if rec := p.get(m, "/bookings"); rec.Code != http.StatusForbidden {
		t.Fatalf("before the grant: got %d, want 403", rec.Code)
	}
	p.grantPage(p.login(management), marketing, "/bookings", "read")
	if rec := p.get(m, "/bookings"); rec.Code != http.StatusOK {
		t.Fatalf("after the grant: got %d, want 200", rec.Code)
	}
}

func TestGrantRecordsWhoGaveItWhenAndWhy(t *testing.T) { // M5
	p := newPortal(t)
	id := p.grantPage(p.login(management), marketing, "/bookings", "read")
	var by, reason string
	var hasTime bool
	p.queryRow3(&by, &reason, &hasTime, `
		SELECT s.email, g.reason, g.granted_at IS NOT NULL
		FROM grants g JOIN staff s ON s.id = g.granted_by WHERE g.id = $1`, id)
	if by != management || reason != "test reason" || !hasTime {
		t.Fatalf("grant record: by=%q reason=%q hasTime=%v", by, reason, hasTime)
	}
}

func TestRevokedGrantIsRefusedOnTheNextRequestAndTheRowIsKept(t *testing.T) { // M5, T4
	p := newPortal(t)
	admin := p.login(management)
	m := p.login(marketing)
	id := p.grantPage(admin, marketing, "/bookings", "read")
	if rec := p.get(m, "/bookings"); rec.Code != http.StatusOK {
		t.Fatalf("with the grant: got %d, want 200", rec.Code)
	}

	if rec := p.do(admin, "DELETE", "/admin/grants/"+id, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: got %d, want 204", rec.Code)
	}
	if rec := p.get(m, "/bookings"); rec.Code != http.StatusForbidden {
		t.Fatalf("after revoke: got %d, want 403", rec.Code)
	}

	var by string
	var revoked bool
	p.queryRow2(&by, &revoked, `
		SELECT s.email, g.revoked_at IS NOT NULL
		FROM grants g JOIN staff s ON s.id = g.revoked_by WHERE g.id = $1`, id)
	if by != management || !revoked {
		t.Fatalf("revoked grant row: by=%q revoked=%v", by, revoked)
	}
}

func TestGrantWithoutAReasonIsRejected(t *testing.T) { // M5
	p := newPortal(t)
	rec := p.post(p.login(management), "/admin/grants", map[string]string{
		"staff_id": p.staffID(marketing), "kind": "page", "page": "/bookings", "mode": "read", "reason": " ",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
}

func TestOnlyAdminsCanGiveGrants(t *testing.T) { // rule 5: never decide access outside the decision
	p := newPortal(t)
	rec := p.post(p.login(finance), "/admin/grants", map[string]string{
		"staff_id": p.staffID(finance), "kind": "page", "page": "/bookings", "mode": "read", "reason": "self-service",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("finance giving itself a grant: got %d, want 403", rec.Code)
	}
}

func TestAdminSeesOnePersonsAccess(t *testing.T) {
	p := newPortal(t)
	admin := p.login(management)
	p.grantPage(admin, marketing, "/bookings", "read")
	rec := p.get(admin, "/admin/staff/"+p.staffID(marketing)+"/access")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	var body struct {
		Department string           `json:"department"`
		Money      string           `json:"money"`
		Grants     []map[string]any `json:"grants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body.Department != "Marketing" || body.Money != "no" || len(body.Grants) != 1 {
		t.Fatalf("access view: %+v", body)
	}
}

// ---- Leaving ----

func TestDeactivatedPersonIsRefusedOnTheNextRequest(t *testing.T) { // T4
	p := newPortal(t)
	fd := p.login(frontDesk)
	if rec := p.get(fd, "/bookings"); rec.Code != http.StatusOK {
		t.Fatalf("before: got %d, want 200", rec.Code)
	}
	rec := p.post(p.login(management), "/admin/staff/"+p.staffID(frontDesk)+"/deactivate", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("deactivate: got %d, want 204", rec.Code)
	}
	if rec := p.get(fd, "/bookings"); rec.Code != http.StatusForbidden {
		t.Fatalf("after deactivation: got %d, want 403", rec.Code)
	}
	if rec := p.post(nil, "/login", map[string]string{"email": frontDesk, "password": demoPassword}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("deactivated person signing in again: got %d, want 401", rec.Code)
	}
}

// ---- Audit ----

func TestRefusalWritesOneAuditRowWithWhoPageAndWhen(t *testing.T) { // T1
	p := newPortal(t)
	p.get(p.login(doctor), "/revenue")

	var n int
	p.queryRow(&n, `SELECT count(*) FROM audit_log`)
	if n != 1 {
		t.Fatalf("got %d audit rows, want 1", n)
	}
	var who, reason string
	var hasTime bool
	p.queryRow3(&who, &reason, &hasTime, `
		SELECT s.email, a.reason_code, a.at IS NOT NULL
		FROM audit_log a JOIN staff s ON s.id = a.staff_id
		WHERE a.page_path = '/revenue' AND a.action = 'read'`)
	if who != doctor || reason != "money_forbidden" || !hasTime {
		t.Fatalf("audit row: who=%q reason=%q hasTime=%v", who, reason, hasTime)
	}
}

func TestAllowedRequestWritesNoAuditRow(t *testing.T) { // T1
	p := newPortal(t)
	p.get(p.login(doctor), "/medical-record")
	var n int
	p.queryRow(&n, `SELECT count(*) FROM audit_log`)
	if n != 0 {
		t.Fatalf("got %d audit rows, want 0", n)
	}
}

func TestNoPasswordOrPersonalDataInLogsOrAudit(t *testing.T) { // rule 2
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(old)

	p := newPortal(t)
	p.post(nil, "/login", map[string]string{"email": frontDesk, "password": "a-wrong-password"})
	c := p.login(frontDesk)
	p.get(c, "/revenue")
	p.get(c, "/patients")

	var audit string
	p.queryRow(&audit, `SELECT coalesce(string_agg(page_path || ' ' || action || ' ' || reason_code, ' '), '')
		FROM audit_log`)
	for _, secret := range []string{demoPassword, "a-wrong-password", frontDesk, "Sample Customer", c.Value} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("the log output contains %q", secret)
		}
		if strings.Contains(audit, secret) {
			t.Errorf("the audit log contains %q", secret)
		}
	}
}

// ---- helpers ----

type portal struct {
	t       *testing.T
	db      *sql.DB
	handler http.Handler
}

// newPortal gives a test its own clean copy of the schema and seed data, in a
// separate PostgreSQL schema so the developer's local data is never touched.
func newPortal(t *testing.T) *portal {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Fatal("TEST_DATABASE_URL is not set: run the tests with `make test`, which starts PostgreSQL")
	}
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", "portal_test")
	u.RawQuery = q.Encode()

	db, err := store.Open(u.String())
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	p := &portal{t: t, db: db}
	p.exec(`DROP SCHEMA IF EXISTS portal_test CASCADE`)
	p.exec(`CREATE SCHEMA portal_test`)
	if err := store.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	p.handler = newMux(db)
	return p
}

func (p *portal) do(c *http.Cookie, method, path string, body any) *httptest.ResponseRecorder {
	p.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			p.t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if c != nil {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	p.handler.ServeHTTP(rec, req)
	return rec
}

func (p *portal) get(c *http.Cookie, path string) *httptest.ResponseRecorder {
	p.t.Helper()
	return p.do(c, "GET", path, nil)
}

func (p *portal) post(c *http.Cookie, path string, body any) *httptest.ResponseRecorder {
	p.t.Helper()
	return p.do(c, "POST", path, body)
}

func (p *portal) login(email string) *http.Cookie {
	p.t.Helper()
	rec := p.post(nil, "/login", map[string]string{"email": email, "password": demoPassword})
	c := sessionCookie(rec)
	if rec.Code != http.StatusNoContent || c == nil {
		p.t.Fatalf("login as %s: got %d", email, rec.Code)
	}
	return c
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			return c
		}
	}
	return nil
}

func (p *portal) staffID(email string) string {
	p.t.Helper()
	var id string
	p.queryRow(&id, `SELECT id FROM staff WHERE email = $1`, email)
	return id
}

func (p *portal) grantPage(admin *http.Cookie, email, page, mode string) string {
	p.t.Helper()
	return p.grant(admin, map[string]string{
		"staff_id": p.staffID(email), "kind": "page", "page": page, "mode": mode, "reason": "test reason",
	})
}

func (p *portal) grantMoney(admin *http.Cookie, email string) string {
	p.t.Helper()
	return p.grant(admin, map[string]string{
		"staff_id": p.staffID(email), "kind": "money", "reason": "test reason",
	})
}

func (p *portal) grant(admin *http.Cookie, body map[string]string) string {
	p.t.Helper()
	rec := p.post(admin, "/admin/grants", body)
	if rec.Code != http.StatusCreated {
		p.t.Fatalf("create grant %v: got %d %s", body, rec.Code, rec.Body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.ID == "" {
		p.t.Fatalf("create grant: bad response %q", rec.Body)
	}
	return out.ID
}

// meAnswer is the body of GET /me.
type meAnswer struct {
	Name       string `json:"name"`
	Department string `json:"department"`
	Branch     string `json:"branch"`
	HeadOffice bool   `json:"head_office"`
	Pages      []struct {
		Page  string `json:"page"`
		Read  bool   `json:"read"`
		Enter bool   `json:"enter"`
	} `json:"pages"`
}

func (m meAnswer) mayRead(page string) bool {
	for _, pg := range m.Pages {
		if pg.Page == page {
			return pg.Read
		}
	}
	return false
}

func (p *portal) me(c *http.Cookie) meAnswer {
	p.t.Helper()
	rec := p.get(c, "/me")
	if rec.Code != http.StatusOK {
		p.t.Fatalf("GET /me: got %d, want 200", rec.Code)
	}
	var out meAnswer
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		p.t.Fatalf("GET /me: response is not JSON: %v", err)
	}
	return out
}

func (p *portal) rows(rec *httptest.ResponseRecorder) []map[string]any {
	p.t.Helper()
	if rec.Code != http.StatusOK {
		p.t.Fatalf("got %d, want 200", rec.Code)
	}
	var body struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		p.t.Fatalf("response is not JSON: %v", err)
	}
	return body.Rows
}

func (p *portal) total(rec *httptest.ResponseRecorder) int64 {
	p.t.Helper()
	if rec.Code != http.StatusOK {
		p.t.Fatalf("got %d, want 200", rec.Code)
	}
	var body struct {
		Total int64 `json:"total_rupiah"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		p.t.Fatalf("response is not JSON: %v", err)
	}
	return body.Total
}

func (p *portal) exec(query string, args ...any) {
	p.t.Helper()
	if _, err := p.db.Exec(query, args...); err != nil {
		p.t.Fatalf("exec %q: %v", query, err)
	}
}

func (p *portal) queryRow(dest any, query string, args ...any) {
	p.t.Helper()
	if err := p.db.QueryRow(query, args...).Scan(dest); err != nil {
		p.t.Fatalf("query %q: %v", query, err)
	}
}

func (p *portal) queryRow2(a, b any, query string, args ...any) {
	p.t.Helper()
	if err := p.db.QueryRow(query, args...).Scan(a, b); err != nil {
		p.t.Fatalf("query %q: %v", query, err)
	}
}

func (p *portal) queryRow3(a, b, c any, query string, args ...any) {
	p.t.Helper()
	if err := p.db.QueryRow(query, args...).Scan(a, b, c); err != nil {
		p.t.Fatalf("query %q: %v", query, err)
	}
}
