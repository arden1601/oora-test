import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'
import Admin from './Admin'
import PageView from './PageView'
import SalesEntry from './SalesEntry'
import { api, errorText } from './api'
import type { Me } from './api'

// The staff portal screen. Anyone can sign in. The tabs come from the server's
// answer to GET /me, which is the access decision's own answer for this person.
// This screen decides nothing: a tab is only a shortcut, and the server checks
// every page again when it is opened. A 403 is simply shown.

// tabsOf turns the /me answer into the tabs to draw.
function tabsOf(me: Me) {
  const tabs: string[] = []
  for (const p of me.pages) {
    if (p.page === '/me') continue // this screen itself
    // /sales-entry is a form, so it needs "enter"; every other page is opened with "read".
    const usable = p.page === '/sales-entry' ? p.enter : p.read
    if (usable) tabs.push(p.page)
  }
  return tabs
}

function App() {
  const [me, setMe] = useState<Me | null>(null)
  const [tab, setTab] = useState('')
  const [message, setMessage] = useState('')

  // showMe keeps the server's answer and picks the tab to open.
  function showMe(data: Me) {
    setMe(data)
    // Keep the open tab if it is still allowed, else open the first one.
    const tabs = tabsOf(data)
    setTab((old) => (tabs.includes(old) ? old : (tabs[0] ?? '')))
  }

  // loadMe asks the server who is signed in and what they may open.
  const loadMe = () => api('GET', '/me').then(showMe)

  // On first load, try /me: it works if a session cookie is still valid.
  useEffect(() => {
    api('GET', '/me')
      .then(showMe)
      .catch(() => setMe(null))
  }, [])

  function signIn(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    setMessage('')
    api('POST', '/login', { email: form.get('email'), password: form.get('password') })
      .then(loadMe)
      .catch((err) => setMessage(err.message === 'Please sign in.' ? 'Wrong email or password.' : errorText(err)))
  }

  function signOut() {
    setMessage('')
    api('POST', '/logout')
      .then(() => setMe(null))
      .catch((err) => setMessage(errorText(err)))
  }

  function reload() {
    setMessage('')
    loadMe().catch((err) => {
      setMe(null)
      setMessage(errorText(err))
    })
  }

  if (!me) {
    return (
      <main>
        <h1>Abra staff portal</h1>
        <form onSubmit={signIn} className="card signin">
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
        <p className="hint">The demo accounts and their password are listed in the README.</p>
      </main>
    )
  }

  const tabs = tabsOf(me)

  return (
    <main>
      <header>
        <div>
          <h1>Abra staff portal</h1>
          <p className="who">
            {me.name}: {me.department}, {me.branch}.{' '}
            {me.head_office ? 'Sees all branches.' : 'Sees this branch only.'}
          </p>
        </div>
        <div className="actions">
          <button type="button" onClick={reload}>
            Reload my access
          </button>
          <button type="button" onClick={signOut}>
            Sign out
          </button>
        </div>
      </header>
      {message && <p className="message">{message}</p>}

      {tabs.length === 0 ? (
        <p className="card">This account has no pages. An admin can give it one.</p>
      ) : (
        <nav className="tabs">
          {tabs.map((t) => (
            <button key={t} type="button" className={t === tab ? 'selected' : ''} onClick={() => setTab(t)}>
              {t}
            </button>
          ))}
        </nav>
      )}

      {/* key={tab} starts each page fresh, so nothing of one page is left on the next. */}
      {tab === '/admin' && <Admin key={tab} pages={me.pages.map((p) => p.page).filter((p) => p !== '/me')} />}
      {tab === '/sales-entry' && <SalesEntry key={tab} />}
      {tab !== '' && tab !== '/admin' && tab !== '/sales-entry' && <PageView key={tab} path={tab} />}
    </main>
  )
}

export default App
