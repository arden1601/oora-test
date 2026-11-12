# Abra staff portal

**Abra Skin Clinic** — *Your natural shimmer.*

A mini staff portal for Abra, an invented Jakarta skin clinic: who can sign in, and what each
person can see and do. Built for the OORA backend engineer test; written answers are in
`docs/answers.md`. Go + PostgreSQL, with an optional React admin screen.

> **Status: scaffold only.** The service starts and answers `GET /healthz`. Sign-in, pages,
> grants and audit are not built yet. Sections marked _to be written_ are filled in as each part
> of the test is done.

## Run it

Needs Docker.

```
docker compose up
```

Backend: http://localhost:8080 (`GET /healthz` returns `ok`).

If port 8080 or 5433 is already in use on your machine, copy `.env.example` to `.env` and change
`BACKEND_HOST_PORT` / `DB_HOST_PORT`.

## Test it

```
make test
```

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

_To be written (Question 2, item 3)._

## Sign in as

_To be written: the seeded fake staff and the demo password._

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

_More to be added as the build goes on._

## What I would build next

_To be written._

## Optional frontend

_To be written: built, or described._

## How I would run this for real (plan only)

### Where the server lives

Everything is deployed to **one VPS at Sumopod, Jakarta region**. It is cheap (about
Rp 85,000–150,000 a month for 2 cores and 4–8 GB), close to the users, and keeps personal and
health data inside Indonesia, which avoids the cross-border transfer question in the PDP law.
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
restore is tested. Moving the database to a managed service is the first step when the second
branch opens. Tailscale needs a paid plan for company use (the free plan is non-commercial).
Sumopod resells capacity from other clouds, so the Jakarta region must be chosen at order time.

### Customer data
_To be written._

### Company data
_To be written._

### Running it
_To be written._

### Getting there
_To be written._
