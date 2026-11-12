-- 0001_init: tables for the staff portal. All data in this project is fake.

-- Head office is a branch with the flag set; it sees all branches.
CREATE TABLE branches (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name           text NOT NULL UNIQUE,
    is_head_office boolean NOT NULL DEFAULT false
);

-- money is the department's money setting (rules M1-M4 as data):
--   yes   may open pages that show money
--   no    may not, unless one named person holds a money grant
--   never may not, and no grant can change it
CREATE TABLE departments (
    id    text PRIMARY KEY,
    name  text NOT NULL UNIQUE,
    money text NOT NULL CHECK (money IN ('yes', 'no', 'never'))
);

-- Every person has exactly one department and one branch.
CREATE TABLE staff (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    display_name  text NOT NULL,
    department_id text NOT NULL REFERENCES departments (id),
    branch_id     uuid NOT NULL REFERENCES branches (id),
    is_active     boolean NOT NULL DEFAULT true
);

-- A page that shows any money figure is a money page.
CREATE TABLE pages (
    path        text PRIMARY KEY,
    shows_money boolean NOT NULL DEFAULT false
);

-- The permission table: which department has which page, in which mode.
--   read  opens the page
--   enter submits to it, never reads back
CREATE TABLE department_pages (
    department_id text NOT NULL REFERENCES departments (id),
    page_path     text NOT NULL REFERENCES pages (path),
    mode          text NOT NULL CHECK (mode IN ('read', 'enter')),
    PRIMARY KEY (department_id, page_path, mode)
);

-- Exceptions for one named person: one page, or "may see money".
-- Rows are never deleted. Taking a grant away fills in revoked_by and revoked_at.
CREATE TABLE grants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id   uuid NOT NULL REFERENCES staff (id),
    kind       text NOT NULL CHECK (kind IN ('page', 'money')),
    page_path  text REFERENCES pages (path),
    mode       text CHECK (mode IN ('read', 'enter')),
    reason     text NOT NULL CHECK (reason <> ''),
    granted_by uuid NOT NULL REFERENCES staff (id),
    granted_at timestamptz NOT NULL DEFAULT now(),
    revoked_by uuid REFERENCES staff (id),
    revoked_at timestamptz,
    -- a page grant names a page and a mode; a money grant names neither
    CHECK ((kind = 'page' AND page_path IS NOT NULL AND mode IS NOT NULL)
        OR (kind = 'money' AND page_path IS NULL AND mode IS NULL))
);
CREATE INDEX grants_staff_idx ON grants (staff_id) WHERE revoked_at IS NULL;

-- Server-side sessions. Only the hash of the cookie value is stored.
CREATE TABLE sessions (
    token_hash   text PRIMARY KEY,
    staff_id     uuid NOT NULL REFERENCES staff (id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

-- Every refused access: who, which page, when, and a reason code.
-- There is no free-text column, so customer data cannot end up here.
CREATE TABLE audit_log (
    id          bigserial PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    staff_id    uuid REFERENCES staff (id), -- empty when nobody was signed in
    page_path   text NOT NULL,
    action      text NOT NULL,
    reason_code text NOT NULL
);

-- Fake business data, just enough to show each page. Every row carries a branch.
CREATE TABLE patients (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id    uuid NOT NULL REFERENCES branches (id),
    display_name text NOT NULL,
    phone        text NOT NULL
);

CREATE TABLE bookings (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id  uuid NOT NULL REFERENCES branches (id),
    patient_id uuid NOT NULL REFERENCES patients (id),
    treatment  text NOT NULL,
    starts_at  timestamptz NOT NULL,
    status     text NOT NULL
);

CREATE TABLE medical_records (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id  uuid NOT NULL REFERENCES branches (id),
    patient_id uuid NOT NULL REFERENCES patients (id),
    note       text NOT NULL
);

-- Money is a whole number of rupiah.
CREATE TABLE sales (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id     uuid NOT NULL REFERENCES branches (id),
    entered_by    uuid NOT NULL REFERENCES staff (id),
    item          text NOT NULL,
    amount_rupiah bigint NOT NULL CHECK (amount_rupiah > 0),
    entered_at    timestamptz NOT NULL DEFAULT now()
);
