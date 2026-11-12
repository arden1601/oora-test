# Abra staff portal

**Abra Skin Clinic** — *Your natural shimmer.*

A mini staff portal for Abra, an invented Jakarta skin clinic: who can sign in, and what each
person can see and do. Built for the OORA backend engineer test; written answers are in
`docs/answers.md`. Go + PostgreSQL, with an optional React screen (portal and admin).

> **Status.** Built and tested: sign-in and sessions, the five pages, the access rules as
> data, grants, the audit of refusals, branch scoping across Head Office and ten clinics, seed
> data (47 people, one per department per branch) and a screen for every account with an admin
> part for grants. The plan for running it for real is at the end. Designed but not built:
> 2FA, lockout, exports and the AI assistant (see "What I would build next").

## Run it

Needs Docker.

```
docker compose up
```

Backend: http://localhost:8080 (`GET /healthz` returns `ok`). The database is created,
migrated and seeded on the first start.

**API docs you can click:** open http://localhost:8080/docs. It is Swagger UI over
`backend/api/openapi.json`. Run `POST /login` there first with one of the accounts below; the
browser keeps the session cookie, so every later "Try it out" call is made as that person.
Run `POST /logout` to switch account. The page loads Swagger UI from a public CDN, so it needs
internet access; set `API_DOCS=false` to switch it off.

Or from a terminal:

```
# sign in as the Front Desk and keep the session cookie
curl -i -c jar -H 'Content-Type: application/json' \
  -d '{"email":"frontdesk@example.com","password":"abra-demo-2026"}' localhost:8080/login

curl -b jar localhost:8080/bookings      # 200: own branch only
curl -b jar localhost:8080/revenue       # 403, and one audit row
curl -b jar -H 'Content-Type: application/json' \
  -d '{"item":"Facial","amount_rupiah":750000}' localhost:8080/sales-entry   # 201, id only
```

| Route | What |
|---|---|
| `POST /login`, `POST /logout` | sign in with email and password, sign out |
| `GET /me` | who is signed in, and which pages they may read or enter on |
| `GET /bookings`, `/patients`, `/medical-record`, `/revenue` | the read pages |
| `POST /sales-entry` | enter one sale; the answer is the new ID and nothing else |
| `GET /admin/staff`, `GET /admin/staff/{id}/access` | staff list; one person's access and grant history |
| `POST /admin/grants`, `DELETE /admin/grants/{id}` | give one page or "may see money"; take it away |
| `POST /admin/staff/{id}/deactivate` | the person has left |

If port 8080 or 5433 is already in use on your machine, copy `.env.example` to `.env` and change
`BACKEND_HOST_PORT` / `DB_HOST_PORT`.

## Test it

```
make test
```

It starts PostgreSQL with Docker and runs every test. The end-to-end tests use their own
schema (`portal_test`), so they never touch the data you see when you run the service. Needs
Go 1.27 and Docker.

- `backend/internal/access/decide_test.go`: the decision function, one test per rule.
- `backend/cmd/server/portal_test.go`: every rule again through the real HTTP routes and a
  real database, including every cell of the permission table.
- `backend/cmd/server/docs_test.go`: the OpenAPI file and the routes must match, both ways.

## Layout

| Path | What |
|---|---|
| `backend/` | Go service, standard library `net/http`, PostgreSQL |
| `frontend/` | React + Vite + TypeScript, optional screen: portal for every account, admin part for grants |
| `docs/answers.md` | the written answers (warm-up, Question 1, Question 3) |
| `docs/test-brief/` | the test as received |
| `docs/pdp-checklist.md` | the UU PDP gate every change passes |
| `AGENTS.md` | the instruction file the AI coding tools worked from |

## Access rules and why

The rules are rows in the database, written in `backend/migrations/0002_seed.sql`. Nothing in
the Go code names a department. Three pieces of data decide access:

- **Permission table** (`department_pages`): department, page, mode. `read` opens a page;
  `enter` submits to it and never reads back.
- **Money setting** (`departments.money`): `yes`, `no` or `never`. A page that shows money
  (`pages.shows_money`) needs `yes`. A grant can turn `no` into `yes` for one person; nothing
  changes `never`.
- **Grants** (`grants`): one named person gets one page or "may see money".

| Department | /bookings | /patients | /medical-record | /sales-entry | /revenue | Money |
|---|---|---|---|---|---|---|
| Management, Growth | R | R | R | E | R | yes |
| Finance | | | | | R | yes |
| CRM & Sales | R | | | | | no |
| Clinic Operations | R | R | | | | no |
| Front Desk | R | R | | E | | never |
| Beauty Consultants | | R | | | | no |
| Doctor | | R | R | | | never |
| Marketing, HR, Partnerships | | | | | | no |

`R` read, `E` enter only. Management and Growth also have the `/admin` page. Every department
reads `/me`, the page that tells a person what they may open.

Why these choices:

- **Each department gets the pages it works with, and no more.** Marketing, HR and
  Partnerships own none of the five pages, so they open nothing until someone is given a grant.
- **Front Desk and the clinic read `/patients`** because check-in and treatment need the
  patient card. The medical record is a separate page, open only to doctors.
- **`/sales-entry` is enter-only for everyone.** Totals live on `/revenue`. That is how the
  Front Desk enters sales without seeing them.
- **Doctors and the Front Desk are `never`**, so even an admin's mistake cannot show them money.
- **A clinic lead sees revenue by two grants** (the page and money), and then only for their
  own branch.

One function decides every request: `Decide` in `backend/internal/access/decide.go`. The
person must be active, pass the money rule, and have the page by table or grant. The guard in
`backend/internal/httpx/guard.go` calls it for every route except sign-in, loading the person
and their grants fresh each time. A refusal is a `403` and one row in `audit_log`.

Departments are given access by adding rows to `department_pages` in a new migration, which
is reviewed like code. One person is given access with a grant through the admin API.

## Sign in as

47 fake people, all with the password `abra-demo-2026` (a demo password for fake accounts;
only its hash is stored). There is one person in each department in each branch: seven at Head
Office, and four in each of the ten clinics.

Head Office (sees all branches):

| Email | Department |
|---|---|
| `management@example.com` | Management |
| `growth@example.com` | Growth |
| `finance@example.com` | Finance |
| `marketing@example.com` | Marketing |
| `hr@example.com` | HR |
| `partnerships@example.com` | Partnerships |
| `crm@example.com` | CRM & Sales |

Clinics (each sees only its own branch). The email is `<role>-<clinic>@example.com`:

| Role in the email | Department |
|---|---|
| `clinic-lead` | Clinic Operations |
| `frontdesk` | Front Desk |
| `consultant` | Beauty Consultants |
| `doctor` | Doctor |

| Clinic | In the email | Example |
|---|---|---|
| Abra Kemang | *(none)* | `frontdesk@example.com`, `doctor@example.com` |
| Abra Menteng | `menteng` | `frontdesk-menteng@example.com` |
| Abra Senopati | `senopati` | `doctor-senopati@example.com` |
| Abra Pondok Indah | `pondok-indah` | `clinic-lead-pondok-indah@example.com` |
| Abra Kelapa Gading | `kelapa-gading` | `consultant-kelapa-gading@example.com` |
| Abra Pantai Indah Kapuk | `pantai-indah-kapuk` | `frontdesk-pantai-indah-kapuk@example.com` |
| Abra Puri Indah | `puri-indah` | `doctor-puri-indah@example.com` |
| Abra Tebet | `tebet` | `frontdesk-tebet@example.com` |
| Abra Kuningan | `kuningan` | `clinic-lead-kuningan@example.com` |
| Abra Cempaka Putih | `cempaka-putih` | `consultant-cempaka-putih@example.com` |

Kemang was the first clinic, so its four people have no clinic name in the email.

## Assumptions

The brief does not settle these, so they are written down here and used throughout.

- **A money page is any page that shows a money figure.** Of the five pages here only
  `/revenue` is one. This keeps the doctor rule ("revenue or any money figure") true for pages
  added later.
- **The Front Desk can never be granted money**, like doctors. The rule says the Front Desk
  "never sees the totals", so I read it as absolute rather than as a default a grant can lift.
- **An admin is anyone in Management or Growth.** They are the people who already see
  everything, so they are the ones who create and deactivate accounts and give and take away
  grants, all on one admin screen. HR has no special path; if HR should do onboarding, it gets
  the admin page by a grant like anyone else.
- **Management and Growth may open the medical record**, because the rule says they see
  everything. This sits badly with "health data only to those who need it"; in a real clinic I
  would ask management to give that up and use named grants instead.

- **A request with no valid session is a `401`, and is audited with an empty "who".** Failed
  sign-ins are audited the same way. The email typed is never stored or logged.
- **A deactivated person keeps their session row but is refused** with a `403` on the next
  request, because the staff row is read every time.
- **A grant to a `never` department is accepted but has no effect.** The decision refuses it
  on every request; the admin API does not second-guess the decision.
- **`/me` is a page, and every department may read it.** It is a row in the permission table
  like the rest, so it goes through the same guard: a person who has left is refused there too.
  Its list of pages is the access decision's own answer for each page. It only helps a screen
  draw its menu; every page still checks access when it is opened.
- **One person per department per branch means 47 people.** The brief puts seven departments
  at head office and four at the clinic, so the seed has seven people at Head Office and four
  in each of ten clinics. Head-office departments are not repeated in every clinic.
- **Pages answer in JSON.** The brief asks for "a simple page with fake data"; JSON keeps the
  service small and is what the admin screen reads.

## What I would build next

In this order:

1. **2FA and lockout.** The authenticator code and "five wrong passwords lock the account for
   15 minutes" are in the design (Question 1) and not in this build.
2. **Separate database roles.** Today the service migrates and runs with one role. The app's
   role should not be able to change tables or delete audit and grant rows; migrations should
   run under their own role.
3. **Audit of reads of health data,** not only refusals, and an admin view of the audit log.
4. **Exports of customer data** with approval in the team chat.
5. **The AI assistant,** with every tool going through the same guard as a page.
6. **An expiry date on grants,** for exceptions that are meant to be short.

## Optional frontend

Built, plain: five small files in `frontend/src/`.

```
cd frontend && pnpm install && pnpm dev     # http://localhost:5173
```

Any of the 47 accounts can sign in. The screen asks the server `GET /me` and draws one tab for
each page that person may open: the read pages as a table of the fake data, `/sales-entry` as
a form that only says "saved". An account with no pages (Marketing, HR, Partnerships) is told
so.

Management and Growth also get the **`/admin`** tab: filter the staff list by branch and
department, pick a person, see what their department gives them and every grant with its
history, give one page or "may see money" with a reason, take a grant away, or mark the person
as having left.

To watch a grant work: as `management@example.com` give `marketing@example.com` the
`/bookings` page. Sign in as marketing in another browser (or after signing out) and the tab is
there. Take the grant away, press "Reload my access", and the tab is gone; opening the page
anyway answers `403`.

The screen decides nothing. The tabs are the server's answer, every page request is checked
again by the server, and a `403` is shown as a message. Hiding a button is not a control, so
there is no access check in the browser at all.

## How I would run this for real (plan only)

### Two sizes

There are two designs. The first is what this test really uses. The second is an estimate for
10 branches, made by multiplying the test workload by 10. It is a plan, not something built.

Starting point: the brief's one clinic, 44 staff, 14 of them at the clinic.

| Measure | Test size (1 clinic) | × 10 (10 branches) | How it is estimated |
|---|---|---|---|
| Staff accounts | 44 | about 440 | straight × 10, an upper bound (head office will not grow 10 times) |
| Signed in at the busiest moment | about 25 | about 250 | a little over half of staff |
| Requests per second at peak | about 5 | about 50 | one request every 5 seconds per active person |
| Database queries per second | about 20 | about 200 | 3–4 small indexed queries per request |
| Visits per day | about 40 | about 400 | one clinic's rooms and staff |
| Table data per year | well under 1 GB | a few GB | bookings, visits, sales and audit rows are small |
| Photos per year | about 90 GB | about 900 GB | 2 photos of 3 MB per visit |

What the numbers say: 50 requests a second and a few GB of rows is still a small system. One
modest PostgreSQL server handles thousands of queries a second, so × 10 does not call for a
different architecture, a queue, a cache or more services. Two things do change:

- **Failure costs more.** One dead server would stop 10 clinics instead of one, so no single
  server may be able to stop the service.
- **Photos become the bulk of the data.** About 900 GB a year does not belong on a server
  disk or in the database.

| Part | Test size (what we use) | Estimated production, 10 branches | Why it changes |
|---|---|---|---|
| App servers | 1 VPS runs everything | 2 small app VPSs behind a load balancer | Sessions live in the database, so a second copy is cheap and removes the single point of failure |
| Database | PostgreSQL on the same VPS | 1 database VPS plus 1 standby VPS that copies it continuously | A dead database server no longer means restoring from last night |
| Backups | nightly dump, copied off the box | continuous backup (restore to any minute) to object storage; monthly restore test | Losing a day of 10 clinics' records is too much |
| Photos and files | on the VPS disk | object storage in Jakarta, encrypted | about 900 GB a year |
| Background jobs | `jobs` table, run inside the service | same table, one separate worker process | Reminders and exports must not slow page requests; still no queue product |
| Sign-in | password + authenticator code in the app | single sign-on through the company's staff directory, 2FA enforced there | 440 accounts are too many to create and remove by hand |
| Secrets | owner-only `.env` on the VPS | one encrypted secrets file in the repo, decrypted on each server at deploy | Several servers must not drift apart |
| Deploys | GitHub Actions over Tailscale SSH | the same, one app server at a time, plus a staging VPS | A bad deploy never takes both app servers down |
| Logs and alerts | Docker logs, outside uptime check | one small monitoring VPS for logs and metrics, alerts to the team chat | Nobody reads logs on six servers by hand |
| Server hardening | four layers on one VPS | the same four layers on every VPS; the database only on the private network | Unchanged |
| Access model | permission table, money setting, grants, one decision function | unchanged; a new branch is one row | Branch-aware from day one |
| Rough cost / month | about $25 | about $80–110 | 6 VPSs, about 1 TB of object storage, Tailscale for more engineers |

Kept the same on purpose: one Go service, one PostgreSQL, the access decision, branch scoping
and the audit log. Nothing in the application is rewritten to go from one size to the other.

When to move: before the second branch opens, not at the tenth. The trigger is that an hour of
downtime costs more than a month of the bigger setup. The standby database and the second app
server come first.

The weak point of the bigger design is the standby database we run ourselves: switching to it
is a written manual step, not automatic. That is simpler to get right with one or two
engineers; a managed PostgreSQL in a Jakarta region is the alternative if it proves too much.

Costs are rough. Sumopod's 2-core plans are about Rp 85,000 (4 GB) and Rp 150,000 (8 GB) on
third-party pages; Tailscale Standard is $8 per user; the object storage price is not checked.

### Where the server lives

At the test size, everything is deployed to **one VPS at Sumopod, Jakarta region**. It is
cheap (about Rp 85,000–150,000 a month for 2 cores and 4–8 GB), close to the users, and keeps
personal and health data inside Indonesia, which avoids the cross-border transfer question in
the PDP law.
The VPS runs Docker Compose: a reverse proxy for HTTPS, the Go service and PostgreSQL. Only
ports 80 and 443 are open to the internet.

The server is locked down in four layers:

1. **No root login and no password login.** SSH accepts keys only, and each engineer has
   their own named user with `sudo`.
2. **SSH on a non-default port.** This only cuts noise from scanners; it is not a control on
   its own.
3. **UFW firewall.** Everything incoming is denied except 80 and 443. SSH is allowed only on
   the Tailscale interface, and PostgreSQL is not exposed at all.
4. **Tailscale.** Engineers and the deploy job reach SSH only through the private network.
   Removing a person from it removes their server access at once.

Trade-offs I accept at this size: one VPS is a single point of failure and we run PostgreSQL
ourselves, so the database is dumped every night, encrypted, and copied off the box, and a
restore is tested. What changes when more branches open is in "Two sizes" above. Tailscale
needs a paid plan for company use (the free plan is non-commercial). Sumopod resells capacity
from other clouds, so the Jakarta region must be chosen at order time.

### Customer data

- **Where it lives.** Patient records are rows in PostgreSQL on the Jakarta server, in their
  own schema (`clinical`). Photos are never in the database and never on a public link: they
  are encrypted files in private storage (the server disk at test size, object storage in
  Jakarta at 10 branches), and the service hands one out only after the access decision, by an
  ID that says nothing about the customer.
- **Who can read it.** Staff, through the portal only: the permission table decides the page,
  the branch decides the rows, and the medical record is open to doctors. Every read of a
  medical record or photo is logged (who, which record ID, when), so "who has seen my record"
  can be answered inside the law's 3 × 24 hours. Engineers do not read customer data as part
  of normal work.
- **Encryption.** HTTPS on every connection. The server disk is encrypted. Medical notes,
  allergies and photos are encrypted a second time by the service, with a key kept outside the
  database, so a stolen copy of the database alone shows no health data. Backups are encrypted
  before they leave the server, with a separate key held by two people.
- **Backups.** A nightly encrypted copy stored away from the server, kept 30 days; at 10
  branches, continuous backup that can restore to any minute. Photo storage keeps old versions.
- **How long it is kept.** Medical records and photos for as long as health regulation
  requires (25 years after the last visit, to verify), then destroyed. Contact and booking data
  until two years after the last visit, then erased. A customer's request to erase is done
  within 3 × 24 hours for everything the law does not make us keep; copies in backups are gone
  when those backups expire, 30 days later.

### Company data

- **Kept apart by schema.** Sales, stock and attendance are in `company`; customers are in
  `clinical`; staff accounts, the permission table, grants and the audit log are in `access`.
  A sale points to a customer by ID only. No name and no treatment detail is copied into the
  ledger, so Finance can reconcile without seeing health data, and a doctor's page never
  touches a money table.
- **Kept apart by database role.** The service's role reads and writes rows but cannot change
  tables, and cannot change or delete audit and grant rows. Migrations run under their own
  role, used only by the deploy. Finance reporting gets a read-only role that sees `company`
  and nothing else. The backup job has its own role.
- **Who can see it.** Revenue and the ledger: Management, Growth, Finance, and people named by
  a grant. The Front Desk enters sales and reads none. Doctors never see money. Staff records
  and attendance: HR and Management. The audit log and grants: Management and Growth.
- **How long it is kept.** Sales records for ten years, as tax law requires (to verify). Staff
  records for the time of employment plus the legal minimum. Audit and grant rows are never
  deleted by the service.
- **Leaving the system.** An export of either kind of data needs the same approval in the team
  chat and is logged.

### Running it

- **How a change reaches production.** A change is a pull request. Tests and lint run
  automatically, and a person reviews it; AI tools never merge or deploy. After the merge,
  GitHub Actions builds the image and deploys it over Tailscale, runs the migrations, and
  checks the health endpoint. If the check fails, the previous image is put back.
- **Database changes and rollback.** Every migration must work with both the old and the new
  code: add first, remove in a later release. So rolling back is always "deploy the previous
  image", never "undo the database". A migration that removes or rewrites data gets a fresh
  backup taken just before it.
- **Who may access the servers and the database.** The one or two engineers, each with their
  own named user and SSH key, only through Tailscale. The database is not reachable from the
  internet. An engineer's own database role is read-only; write access is a separate role
  taken for a stated reason, for a limited time, and logged. When someone leaves, the same
  day: removed from Tailscale (access ends at once), their server user and database role
  dropped, removed from GitHub, and every shared secret they could read is changed.
- **Proving a backup restores.** Once a month a scheduled job restores the newest backup into
  an empty PostgreSQL on a temporary server, compares row counts with production, runs a few
  known queries, and records how long it took. A failed restore, or a backup older than 26
  hours, raises an alert. Once a quarter an engineer does it by hand from the written steps.
- **The booking page is very slow on a busy Saturday: the first hour.**
  1. *First 10 minutes.* Confirm it and size it: one page or all, one clinic or all. Tell the
     clinics, who switch to the paper booking sheet. One person leads; changes are written
     down as they are made.
  2. *To minute 25.* Look before touching: server CPU, memory and disk; database connections;
     which queries are running long or waiting on a lock; whether anything was deployed or a
     heavy job (export, report) started.
  3. *To minute 40.* Take the least risky fix that matches what was seen: put the previous
     image back if there was a deploy; stop the heavy job or the one blocking query; add the
     missing index without locking the table. Restarting the service is the last resort.
  4. *To minute 60.* Confirm the page is fast again from a clinic, keep watching, tell the
     clinics. Afterwards: a short written review and the lasting fix.
  What I would not do: change tables in a hurry, or switch off access checks to make it faster.

### Getting there

1. **Finish what guards real data**: the authenticator code and lockout, the separate
   database roles, and the audit of reads of health data.
2. **Split the seed.** Today the migrations hold both the access rules and the demo people and
   customers. Production loads the rules only; real accounts are created by an admin.
3. **Do the legal work before any real customer is entered**: the impact assessment the PDP
   law asks for when health data is processed, a named person responsible for data protection,
   and contracts with every vendor that will hold data.
4. **Build the server**: the four layers, HTTPS, secrets, backups, and one restore test that
   passes before go-live.
5. **Set up the deploy pipeline** and a staging copy that holds only fake data.
6. **Pilot in one clinic**, with bookings and the Front Desk first, next to the current way of
   working for two weeks. Then add pages one at a time, reading the audit log every week.
7. **Before the second branch opens**, move to the larger design in "Two sizes".
