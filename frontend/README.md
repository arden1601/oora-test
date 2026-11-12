# Admin screen (optional part of the test)

A plain React screen on top of the admin API: sign in, pick a person, see their access (what
the department gives, every grant with its history), give one page or "may see money" with a
reason, take a grant away, and mark a person as having left.

```
pnpm install
pnpm dev        # http://localhost:5173, forwards /api to the Go service
```

If the Go service is not on port 8080, start it with the same `BACKEND_HOST_PORT` you set in
the root `.env`, for example `BACKEND_HOST_PORT=8091 pnpm dev`.

The screen decides nothing. Every button calls the server, the server's access decision
answers, and a `403` is shown as a message. Sign in as `management@example.com` or
`growth@example.com` to use it; any other account is refused by the server.
