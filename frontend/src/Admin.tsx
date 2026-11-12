import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api, errorText } from './api'
import type { Access, StaffRow } from './api'

// The admin screen: see one person's access, give or take away a page or
// "may see money", or mark the person as having left. It decides nothing itself.
// Every button calls the admin API, and the server's access decision answers.

// pages is the list of pages a grant can name. It comes from the server (GET /me).
function Admin({ pages }: { pages: string[] }) {
  const [staff, setStaff] = useState<StaffRow[]>([])
  const [access, setAccess] = useState<Access | null>(null)
  const [branch, setBranch] = useState('')
  const [department, setDepartment] = useState('')
  const [message, setMessage] = useState('')

  // run wraps every server call so one place shows what went wrong.
  async function run(action: () => Promise<void>) {
    setMessage('')
    try {
      await action()
    } catch (err) {
      setMessage(errorText(err))
    }
  }

  const loadStaff = () => run(async () => setStaff((await api('GET', '/admin/staff')).rows))
  const loadAccess = (id: string) => run(async () => setAccess(await api('GET', `/admin/staff/${id}/access`)))

  useEffect(() => {
    api('GET', '/admin/staff')
      .then((data) => setStaff(data.rows))
      .catch((err) => setMessage(errorText(err)))
  }, [])

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

  // The filters only shorten the list on screen. The server sent everyone.
  const branches = [...new Set(staff.map((s) => s.branch))].sort()
  const departments = [...new Set(staff.map((s) => s.department))].sort()
  const shown = staff.filter(
    (s) => (branch === '' || s.branch === branch) && (department === '' || s.department === department),
  )

  return (
    <>
      {message && <p className="message">{message}</p>}
      <div className="columns">
        <section className="card">
          <h2>Staff ({shown.length})</h2>
          <label>
            Branch
            <select value={branch} onChange={(e) => setBranch(e.target.value)}>
              <option value="">All branches</option>
              {branches.map((b) => (
                <option key={b}>{b}</option>
              ))}
            </select>
          </label>
          <label>
            Department
            <select value={department} onChange={(e) => setDepartment(e.target.value)}>
              <option value="">All departments</option>
              {departments.map((d) => (
                <option key={d}>{d}</option>
              ))}
            </select>
          </label>
          <ul className="staff">
            {shown.map((s) => (
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
                  {pages.map((p) => (
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
    </>
  )
}

export default Admin
