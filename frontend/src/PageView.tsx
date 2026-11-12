import { useEffect, useState } from 'react'
import { api, errorText } from './api'

// One view for every read page (/bookings, /patients, /medical-record, /revenue).
// It asks the server for the page and draws whatever rows come back. If the
// server refuses, the refusal is shown and nothing else.

type Row = Record<string, unknown>
type PageData = { rows: Row[]; total_rupiah?: number }

const rupiah = (n: number) => 'Rp ' + n.toLocaleString('id-ID')

// show formats one cell by the name of its field.
function show(field: string, value: unknown) {
  if (field.endsWith('_rupiah') && typeof value === 'number') return rupiah(value)
  if (field.endsWith('_at') && typeof value === 'string') return new Date(value).toLocaleString()
  return String(value)
}

function PageView({ path }: { path: string }) {
  const [data, setData] = useState<PageData | null>(null)
  const [message, setMessage] = useState('')

  useEffect(() => {
    api('GET', path)
      .then(setData)
      .catch((err) => setMessage(errorText(err)))
  }, [path])

  if (message) return <p className="message">{message}</p>
  if (!data) return <p>Loading…</p>

  // The columns are the fields of the first row, without the internal ID.
  const fields = data.rows.length > 0 ? Object.keys(data.rows[0]).filter((f) => f !== 'id') : []

  return (
    <section className="card">
      <h2>{path}</h2>
      {data.rows.length === 0 ? (
        <p>No rows.</p>
      ) : (
        <table>
          <thead>
            <tr>
              {fields.map((f) => (
                <th key={f}>{f.replaceAll('_', ' ')}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.rows.map((row, i) => (
              <tr key={i}>
                {fields.map((f) => (
                  <td key={f}>{show(f, row[f])}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {data.total_rupiah !== undefined && (
        <p>
          <b>Total: {rupiah(data.total_rupiah)}</b>
        </p>
      )}
    </section>
  )
}

export default PageView
