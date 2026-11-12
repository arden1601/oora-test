import { useState } from 'react'
import type { FormEvent } from 'react'
import { api, errorText } from './api'

// The sales entry form. It sends one sale and shows that it was saved. The
// server answers with the new sale's ID only, so there is no total or list of
// sales here to show: entering a sale never reads sales back.

function SalesEntry() {
  const [message, setMessage] = useState('')
  const [saved, setSaved] = useState('')

  function enter(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const formEl = e.currentTarget
    const form = new FormData(formEl)
    setMessage('')
    setSaved('')
    api('POST', '/sales-entry', { item: form.get('item'), amount_rupiah: Number(form.get('amount')) })
      .then((data) => {
        setSaved(data.id)
        formEl.reset()
      })
      .catch((err) => setMessage(errorText(err)))
  }

  return (
    <section className="card">
      <h2>/sales-entry</h2>
      <form onSubmit={enter} className="grant">
        <label>
          Item
          <input name="item" required />
        </label>
        <label>
          Amount (whole rupiah)
          <input name="amount" type="number" min="1" step="1" required />
        </label>
        <button type="submit">Enter sale</button>
      </form>
      {saved && <p>Saved. Sale ID: {saved}</p>}
      {message && <p className="message">{message}</p>}
    </section>
  )
}

export default SalesEntry
