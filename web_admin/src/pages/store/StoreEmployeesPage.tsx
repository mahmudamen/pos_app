import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { canManage, isOwner, ROLE_LABELS } from '../../auth/roles'
import { ApiError, api } from '../../lib/api'
import type { StoreUser, UserRole } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

type InviteForm = {
  email: string
  display_name: string
  password: string
  role: UserRole
}

export function StoreEmployeesPage() {
  const { role } = useAuth()
  const manageable = canManage(role)
  const owner = isOwner(role)

  const [users, setUsers] = useState<StoreUser[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const limit = 50
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [form, setForm] = useState<InviteForm>({
    email: '',
    display_name: '',
    password: '',
    role: 'cashier',
  })

  async function load(p: number) {
    setError(null)
    try {
      const res = await api.listUsers(p, limit)
      setUsers(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load users')
      }
    }
  }

  useEffect(() => {
    void load(1)
  }, [])

  async function invite() {
    setError(null)
    if (!form.email.trim() || !form.display_name.trim()) {
      setError('Email and display name are required')
      return
    }
    if (form.password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }
    setBusy(true)
    try {
      await api.createUser({
        email: form.email.trim(),
        display_name: form.display_name.trim(),
        password: form.password,
        role: form.role,
      })
      setForm({ email: '', display_name: '', password: '', role: 'cashier' })
      await load(page)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Invite failed')
      }
    } finally {
      setBusy(false)
    }
  }

  const roleChoices: UserRole[] = owner
    ? ['owner', 'manager', 'cashier']
    : ['manager', 'cashier']

  return (
    <div className="page">
      <header className="page-head">
        <h2>Employees</h2>
        <span className="muted">{total} total</span>
      </header>
      <ErrorBanner error={error} />

      {manageable ? (
        <section className="card plan-form">
          <h3>Invite employee</h3>
          <div className="form-grid">
            <label>
              Display name
              <input
                type="text"
                value={form.display_name}
                onChange={(e) =>
                  setForm({ ...form, display_name: e.target.value })
                }
              />
            </label>
            <label>
              Email
              <input
                type="email"
                value={form.email}
                autoComplete="off"
                onChange={(e) => setForm({ ...form, email: e.target.value })}
              />
            </label>
            <label>
              Password (min 8 chars)
              <input
                type="password"
                value={form.password}
                autoComplete="new-password"
                onChange={(e) => setForm({ ...form, password: e.target.value })}
              />
            </label>
            <label>
              Role
              <select
                value={form.role}
                onChange={(e) =>
                  setForm({ ...form, role: e.target.value as UserRole })
                }
              >
                {roleChoices.map((r) => (
                  <option key={r} value={r}>
                    {ROLE_LABELS[r]}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="form-actions">
            <button
              className="btn primary"
              disabled={busy}
              onClick={() => void invite()}
            >
              {busy ? 'Inviting…' : 'Invite employee'}
            </button>
          </div>
        </section>
      ) : null}

      {!users.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Name</th>
              <th>Email</th>
              <th>Role</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} className={u.is_active ? '' : 'muted'}>
                <td>{u.display_name}</td>
                <td>{u.email}</td>
                <td>
                  <span className="badge info">
                    {ROLE_LABELS[u.role] ?? u.role}
                  </span>
                </td>
                <td>
                  {u.is_active ? (
                    <span className="badge ok">active</span>
                  ) : (
                    <span className="badge muted">inactive</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {Math.ceil(total / limit) > 1 ? (
        <div className="pager">
          <button
            className="btn"
            disabled={page <= 1}
            onClick={() => void load(page - 1)}
          >
            Prev
          </button>
          <span>
            Page {page} of {Math.ceil(total / limit)}
          </span>
          <button
            className="btn"
            disabled={page >= Math.ceil(total / limit)}
            onClick={() => void load(page + 1)}
          >
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}