// Types of the server's answers, and the one function that calls the server.

export type MyPage = { page: string; shows_money: boolean; read: boolean; enter: boolean }
export type Me = {
  name: string
  department: string
  branch: string
  head_office: boolean
  pages: MyPage[]
}

export type StaffRow = { id: string; name: string; department: string; branch: string; active: boolean }
export type Grant = {
  id: string
  kind: 'page' | 'money'
  page?: string
  mode?: string
  reason: string
  granted_by: string
  granted_at: string
  revoked_at?: string
}
export type Access = StaffRow & {
  money: string
  department_pages: { page: string; mode: string }[]
  grants: Grant[]
}

// api calls the Go service through the dev proxy and turns a refusal into a message.
export async function api(method: string, path: string, body?: unknown) {
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

// errorText gives the message to show for anything a server call threw.
export function errorText(err: unknown) {
  return err instanceof Error ? err.message : 'Something went wrong.'
}
