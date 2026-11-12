# Staff portal screen (optional part of the test)

A plain React screen on top of the API. Any account can sign in. The screen asks the server
`GET /me` and draws one tab for each page that person may open:

- `/bookings`, `/patients`, `/medical-record`, `/revenue`: the rows the server sends, as a table.
- `/sales-entry`: a form that enters one sale and only says "saved".
- `/admin` (Management and Growth): filter the staff list by branch and department, pick a
  person, see their access (what the department gives, every grant with its history), give one
  page or "may see money" with a reason, take a grant away, and mark a person as having left.

```
pnpm install
pnpm dev        # http://localhost:5173, forwards /api to the Go service
```

If the Go service is not on port 8080, start it with the same `BACKEND_HOST_PORT` you set in
the root `.env`, for example `BACKEND_HOST_PORT=8091 pnpm dev`.

| File | What |
|---|---|
| `src/api.ts` | the one function that calls the server, and the types of its answers |
| `src/App.tsx` | sign-in, `GET /me`, the tabs |
| `src/PageView.tsx` | one view for every read page |
| `src/SalesEntry.tsx` | the sales entry form |
| `src/Admin.tsx` | the admin part |

The screen decides nothing. The tabs are the server's answer to `GET /me`, every page request
is checked again by the server's access decision, and a `403` is shown as a message. The demo
accounts are listed in the root `README.md`.
