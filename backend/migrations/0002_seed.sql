-- 0002_seed: the access rules and fake demo data. Nothing here is real:
-- names are invented, emails use example.com, phone numbers are not valid.

INSERT INTO branches (id, name, is_head_office) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Head Office', true),
    ('00000000-0000-0000-0000-000000000002', 'Abra Kemang', false),
    ('00000000-0000-0000-0000-000000000003', 'Abra Menteng', false);

-- Rules M1-M4 live in the money column: doctors and the Front Desk can never see money.
INSERT INTO departments (id, name, money) VALUES
    ('management',   'Management',         'yes'),
    ('growth',       'Growth',             'yes'),
    ('finance',      'Finance',            'yes'),
    ('marketing',    'Marketing',          'no'),
    ('hr',           'HR',                 'no'),
    ('partnerships', 'Partnerships',       'no'),
    ('crm_sales',    'CRM & Sales',        'no'),
    ('clinic_ops',   'Clinic Operations',  'no'),
    ('consultants',  'Beauty Consultants', 'no'),
    ('front_desk',   'Front Desk',         'never'),
    ('doctor',       'Doctor',             'never');

INSERT INTO pages (path, shows_money) VALUES
    ('/bookings',       false),
    ('/patients',       false),
    ('/medical-record', false),
    ('/sales-entry',    false),
    ('/revenue',        true),
    ('/admin',          false);

-- The permission table. Management and Growth have a row for every page (M1).
-- /sales-entry is enter-only for everyone who has it: totals are on /revenue.
INSERT INTO department_pages (department_id, page_path, mode) VALUES
    ('management', '/bookings',       'read'),
    ('management', '/patients',       'read'),
    ('management', '/medical-record', 'read'),
    ('management', '/sales-entry',    'enter'),
    ('management', '/revenue',        'read'),
    ('management', '/admin',          'read'),
    ('management', '/admin',          'enter'),
    ('growth',     '/bookings',       'read'),
    ('growth',     '/patients',       'read'),
    ('growth',     '/medical-record', 'read'),
    ('growth',     '/sales-entry',    'enter'),
    ('growth',     '/revenue',        'read'),
    ('growth',     '/admin',          'read'),
    ('growth',     '/admin',          'enter'),
    ('finance',     '/revenue',        'read'),
    ('crm_sales',   '/bookings',       'read'),
    ('clinic_ops',  '/bookings',       'read'),
    ('clinic_ops',  '/patients',       'read'),
    ('front_desk',  '/bookings',       'read'),
    ('front_desk',  '/patients',       'read'),
    ('front_desk',  '/sales-entry',    'enter'),
    ('consultants', '/patients',       'read'),
    ('doctor',      '/patients',       'read'),
    ('doctor',      '/medical-record', 'read');

-- Twelve fake staff, one or more per department. They all share one demo password,
-- stored only as a hash made with: go run ./cmd/hashpw '<the demo password in the README>'
INSERT INTO staff (id, email, display_name, department_id, branch_id, password_hash) VALUES
    ('00000000-0000-0000-0000-000000000101', 'management@example.com',   'Demo Management',     'management',   '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000102', 'growth@example.com',       'Demo Growth',         'growth',       '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000103', 'finance@example.com',      'Demo Finance',        'finance',      '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000104', 'marketing@example.com',    'Demo Marketing',      'marketing',    '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000105', 'hr@example.com',           'Demo HR',             'hr',           '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000106', 'partnerships@example.com', 'Demo Partnerships',   'partnerships', '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000107', 'crm@example.com',          'Demo CRM Agent',      'crm_sales',    '00000000-0000-0000-0000-000000000001', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000108', 'clinic-lead@example.com',  'Demo Clinic Lead',    'clinic_ops',   '00000000-0000-0000-0000-000000000002', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000109', 'frontdesk@example.com',    'Demo Front Desk',     'front_desk',   '00000000-0000-0000-0000-000000000002', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000110', 'consultant@example.com',   'Demo Consultant',     'consultants',  '00000000-0000-0000-0000-000000000002', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000111', 'doctor@example.com',       'Demo Doctor',         'doctor',       '00000000-0000-0000-0000-000000000002', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA'),
    ('00000000-0000-0000-0000-000000000112', 'frontdesk-menteng@example.com', 'Demo Front Desk Menteng', 'front_desk', '00000000-0000-0000-0000-000000000003', 'pbkdf2-sha256$600000$ezRVAiHf6YnuZ7VopT3bFQ$/vFjvbZtFf5g8o5Ool/SmndDCsK9R4HePM8HQMLXIdA');

-- Fake customers: two per clinic.
INSERT INTO patients (id, branch_id, display_name, phone) VALUES
    ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000002', 'Sample Customer K1', '+62 000 0000 0001'),
    ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000002', 'Sample Customer K2', '+62 000 0000 0002'),
    ('00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000003', 'Sample Customer M1', '+62 000 0000 0003'),
    ('00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000003', 'Sample Customer M2', '+62 000 0000 0004');

INSERT INTO bookings (branch_id, patient_id, treatment, starts_at, status) VALUES
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000201', 'Facial',        '2026-11-14 10:00+07', 'confirmed'),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000202', 'Laser session', '2026-11-14 13:00+07', 'confirmed'),
    ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000203', 'Consultation',  '2026-11-14 11:00+07', 'confirmed'),
    ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000204', 'Facial',        '2026-11-15 09:00+07', 'waiting');

INSERT INTO medical_records (branch_id, patient_id, note) VALUES
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000201', 'Sample note: sensitive skin, no known allergies.'),
    ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000203', 'Sample note: follow-up in four weeks.');

-- Fake sales: Kemang totals 2,250,000 and Menteng 900,000.
INSERT INTO sales (branch_id, entered_by, item, amount_rupiah) VALUES
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000109', 'Facial',          750000),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000109', 'Laser session',  1500000),
    ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000112', 'Facial package',  900000);
