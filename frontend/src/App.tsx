import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'

// The admin screen: see one person's access, give or take away a page or
// "may see money". It decides nothing itself. Every button calls the server,
// and the server's access decision answers. A 403 is simply shown.

type StaffRow = { id: string; name: string; department: string; branch: string; active: boolean }
type Grant = {
  id: string
  kind: 'page' | 'money'
  page?: string
  mode?: string
  reason: string
  granted_by: string
  granted_at: string
  revoked_at?: string
}
type Access = StaffRow & {
  money: string
  department_pages: { page: string; mode: string }[]
  grants: Grant[]
}

// Only a convenience for the form. The server checks the page exists.
const PAGES = ['/bookings', '/patients', '/medical-record', '/sales-entry', '/revenue']

// api calls the Go service through the dev proxy and turns a refusal into a message.
async function api(method: string, path: string, body?: unknown) {
  const res = await fetch('/api' + path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) throw new Error('Please sign in.')
  if (res.status === 403) throw new Error('The server refused: this account may not do that.')
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.error ?? `Request failed (${res.status})`)
  }
  return res.status === 204 ? null : res.json()
}

function App() {
  const [signedIn, setSignedIn] = useState(false)
  const [staff, setStaff] = useState<StaffRow[]>([])
  const [access, setAccess] = useState<Access | null>(null)
  const [message, setMessage] = useState('')

  // run wraps every server call so one place shows what went wrong.
  async function run(action: () => Promise<void>) {
    setMessage('')
    try {
      await action()
    } catch (err) {
      setMessage(err instanceof Error ? err.message : 'Something went wrong.')
    }
  }

  const loadStaff = () =>
    run(async () => {
      const data = await api('GET', '/admin/staff')
      setStaff(data.rows)
      setSignedIn(true)
    })

  const loadAccess = (id: string) =>
    run(async () => setAccess(await api('GET', `/admin/staff/${id}/access`)))

  // On first load, try the staff list: it works if a session cookie is still valid.
  useEffect(() => {
    api('GET', '/admin/staff')
      .then((data) => {
        setStaff(data.rows)
        setSignedIn(true)
      })
      .catch(() => setSignedIn(false))
  }, [])

  function signIn(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    run(async () => {
      await api('POST', '/login', { email: form.get('email'), password: form.get('password') })
      await loadStaff()
    })
  }

  function signOut() {
    run(async () => {
      await api('POST', '/logout')
      setSignedIn(false)
      setStaff([])
      setAccess(null)
    })
  }

  function giveGrant(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!access) return
    const formEl = e.currentTarget
    const form = new FormData(formEl)
    const kind = form.get('kind')
    run(async () => {
      await api('POST', '/admin/grants', {
        staff_id: access.id,
        kind,
        page: kind === 'page' ? form.get('page') : '',
        mode: kind === 'page' ? form.get('mode') : '',
        reason: form.get('reason'),
      })
      formEl.reset()
      await loadAccess(access.id)
    })
  }

  function takeAway(grantId: string) {
    if (!access) return
    run(async () => {
      await api('DELETE', `/admin/grants/${grantId}`)
      await loadAccess(access.id)
    })
  }

  function deactivate() {
    if (!access) return
    run(async () => {
      await api('POST', `/admin/staff/${access.id}/deactivate`)
      await loadAccess(access.id)
      await loadStaff()
    })
  }

  if (!signedIn) {
    return (
      <main>
        <h1>Abra staff portal: admin</h1>
        <form onSubmit={signIn} className="card">
          <label>
            Email
            <input name="email" type="email" required autoComplete="username" />
          </label>
          <label>
            Password
            <input name="password" type="password" required autoComplete="current-password" />
          </label>
          <button type="submit">Sign in</button>
        </form>
        {message && <p className="message">{message}</p>}
      </main>
    )
  }

  return (
    <main>
      <header>
        <h1>Abra staff portal: admin</h1>
        <button type="button" onClick={signOut}>
          Sign out
        </button>
      </header>
      {message && <p className="message">{message}</p>}

      <div className="columns">
        <section className="card">
          <h2>Staff</h2>
          <ul className="staff">
            {staff.map((s) => (
              <li key={s.id}>
                <button
                  type="button"
                  className={access?.id === s.id ? 'selected' : ''}
                  onClick={() => loadAccess(s.id)}
                >
                  {s.name}
                  <small>
                    {s.department}, {s.branch}
                    {s.active ? '' : ' (left)'}
                  </small>
                </button>
              </li>
            ))}
          </ul>
        </section>

        {access && (
          <section className="card">
            <h2>{access.name}</h2>
            <p>
              {access.department}, {access.branch}. Money setting: <b>{access.money}</b>.{' '}
              {access.active ? 'Active.' : 'Has left: every request is refused.'}
            </p>

            <h3>From the department</h3>
            {access.department_pages.length === 0 ? (
              <p>No pages.</p>
            ) : (
              <ul>
                {access.department_pages.map((p) => (
                  <li key={p.page + p.mode}>
                    <code>{p.page}</code> ({p.mode})
                  </li>
                ))}
              </ul>
            )}

            <h3>Grants to this person</h3>
            {access.grants.length === 0 ? (
              <p>None.</p>
            ) : (
              <table>
                <thead>
                  <tr>
                    <th>What</th>
                    <th>Why</th>
                    <th>Given</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {access.grants.map((g) => (
                    <tr key={g.id}>
                      <td>{g.kind === 'money' ? 'May see money' : `${g.page} (${g.mode})`}</td>
                      <td>{g.reason}</td>
                      <td>{new Date(g.granted_at).toLocaleString()}</td>
                      <td>
                        {g.revoked_at ? (
                          `Taken away ${new Date(g.revoked_at).toLocaleString()}`
                        ) : (
                          <button type="button" onClick={() => takeAway(g.id)}>
                            Take away
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            <h3>Give access</h3>
            <form onSubmit={giveGrant} className="grant">
              <label>
                What
                <select name="kind" defaultValue="page">
                  <option value="page">One page</option>
                  <option value="money">May see money</option>
                </select>
              </label>
              <label>
                Page (for "one page")
                <select name="page">
                  {PAGES.map((p) => (
                    <option key={p}>{p}</option>
                  ))}
                </select>
              </label>
              <label>
                Mode
                <select name="mode">
                  <option value="read">read</option>
                  <option value="enter">enter</option>
                </select>
              </label>
              <label>
                Why (required)
                <input name="reason" required />
              </label>
              <button type="submit">Give</button>
            </form>

            {access.active && (
              <p>
                <button type="button" className="danger" onClick={deactivate}>
                  This person has left
                </button>
              </p>
            )}
          </section>
        )}
      </div>
    </main>
  )
}

export default App
