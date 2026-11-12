# Abra staff portal

**Abra Skin Clinic** — *Your natural shimmer.*

A mini staff portal for Abra, an invented Jakarta skin clinic: who can sign in, and what each
person can see and do. Built for the OORA backend engineer test; written answers are in
`docs/answers.md`. Go + PostgreSQL, with an optional React admin screen.

> **Status.** Built and tested: sign-in and sessions, the five pages, the access rules as
> data, grants, the audit of refusals, branch scoping, seed data and the admin screen. Designed
> but not built: 2FA, lockout, exports and the AI assistant (see "What I would build next").

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
| `frontend/` | React + Vite + TypeScript, optional admin screen |
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

`R` read, `E` enter only. Management and Growth also have the `/admin` page.

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

Twelve fake people, all with the password `abra-demo-2026` (a demo password for fake accounts;
only its hash is stored).

| Email | Department | Branch |
|---|---|---|
| `management@example.com` | Management | Head Office |
| `growth@example.com` | Growth | Head Office |
| `finance@example.com` | Finance | Head Office |
| `marketing@example.com` | Marketing | Head Office |
| `hr@example.com` | HR | Head Office |
| `partnerships@example.com` | Partnerships | Head Office |
| `crm@example.com` | CRM & Sales | Head Office |
| `clinic-lead@example.com` | Clinic Operations | Abra Kemang |
| `frontdesk@example.com` | Front Desk | Abra Kemang |
| `consultant@example.com` | Beauty Consultants | Abra Kemang |
| `doctor@example.com` | Doctor | Abra Kemang |
| `frontdesk-menteng@example.com` | Front Desk | Abra Menteng |

Head office sees both clinics; clinic staff see only their own.

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

Built, plain: `frontend/src/App.tsx`, one file.

```
cd frontend && pnpm install && pnpm dev     # http://localhost:5173
```

Sign in as `management@example.com` or `growth@example.com`. Pick a person to see what their
department gives them and every grant with its history. Give one page or "may see money" with
a reason, take a grant away, or mark the person as having left.

The screen decides nothing. It calls the admin API, the server's access decision answers, and
a `403` is shown as a message. Hiding a button is not a control, so there is no admin check in
the browser at all.

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
_To be written._

### Company data
_To be written._

### Running it
_To be written._

### Getting there
_To be written._
