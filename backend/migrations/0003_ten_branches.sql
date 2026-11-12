-- 0003_ten_branches: the full size the portal is built for. Head Office plus ten
-- clinics, one person per department per branch, and the "/me" page.
-- Nothing here is real: names are invented, emails use example.com, phones are not valid.

-- Eight more clinics, next to Kemang and Menteng from 0002.
INSERT INTO branches (id, name) VALUES
    ('00000000-0000-0000-0000-000000000004', 'Abra Senopati'),
    ('00000000-0000-0000-0000-000000000005', 'Abra Pondok Indah'),
    ('00000000-0000-0000-0000-000000000006', 'Abra Kelapa Gading'),
    ('00000000-0000-0000-0000-000000000007', 'Abra Pantai Indah Kapuk'),
    ('00000000-0000-0000-0000-000000000008', 'Abra Puri Indah'),
    ('00000000-0000-0000-0000-000000000009', 'Abra Tebet'),
    ('00000000-0000-0000-0000-000000000010', 'Abra Kuningan'),
    ('00000000-0000-0000-0000-000000000011', 'Abra Cempaka Putih');

-- "/me" tells the signed-in person who they are and which pages they may open.
-- It is a page like any other, and every department may read it.
INSERT INTO pages (path, shows_money) VALUES ('/me', false);
INSERT INTO department_pages (department_id, page_path, mode)
SELECT id, '/me', 'read' FROM departments;

-- One person in each clinic department in every clinic. People seeded in 0002 are
-- kept, so this adds three in Menteng and four in each new clinic. The email is
-- <role>-<clinic>@example.com, for example doctor-pondok-indah@example.com.
-- Everyone has the same demo password; its hash is copied from an existing row.
INSERT INTO staff (email, display_name, department_id, branch_id, password_hash)
SELECT r.email || '-' || lower(replace(replace(b.name, 'Abra ', ''), ' ', '-')) || '@example.com',
       'Demo ' || r.title || ' ' || replace(b.name, 'Abra ', ''),
       r.department_id,
       b.id,
       (SELECT password_hash FROM staff WHERE email = 'management@example.com')
FROM branches b
CROSS JOIN (VALUES
    ('clinic_ops',  'clinic-lead', 'Clinic Lead'),
    ('front_desk',  'frontdesk',   'Front Desk'),
    ('consultants', 'consultant',  'Consultant'),
    ('doctor',      'doctor',      'Doctor')
) AS r (department_id, email, title)
WHERE NOT b.is_head_office
  AND NOT EXISTS (SELECT 1 FROM staff s
                  WHERE s.branch_id = b.id AND s.department_id = r.department_id);

-- Fake customers: two in each clinic that has none yet.
INSERT INTO patients (branch_id, display_name, phone)
SELECT b.id, 'Sample Customer ' || replace(b.name, 'Abra ', '') || ' ' || n, '+62 000 0000 0000'
FROM branches b
CROSS JOIN generate_series(1, 2) AS n
WHERE NOT b.is_head_office
  AND NOT EXISTS (SELECT 1 FROM patients p WHERE p.branch_id = b.id);

-- One booking in each clinic that has none yet, for one of its customers.
INSERT INTO bookings (branch_id, patient_id, treatment, starts_at, status)
SELECT p.branch_id, p.id, 'Facial', '2026-11-16 10:00+07', 'confirmed'
FROM patients p
WHERE p.display_name LIKE 'Sample Customer % 1'
  AND NOT EXISTS (SELECT 1 FROM bookings k WHERE k.branch_id = p.branch_id);

-- One medical record in each clinic that has none yet.
INSERT INTO medical_records (branch_id, patient_id, note)
SELECT p.branch_id, p.id, 'Sample note: routine check, nothing to report.'
FROM patients p
WHERE p.display_name LIKE 'Sample Customer % 1'
  AND NOT EXISTS (SELECT 1 FROM medical_records m WHERE m.branch_id = p.branch_id);

-- One sale of Rp 500,000 in each clinic that has none yet, entered by its Front Desk.
INSERT INTO sales (branch_id, entered_by, item, amount_rupiah)
SELECT s.branch_id, s.id, 'Facial', 500000
FROM staff s
WHERE s.department_id = 'front_desk'
  AND NOT EXISTS (SELECT 1 FROM sales x WHERE x.branch_id = s.branch_id);
