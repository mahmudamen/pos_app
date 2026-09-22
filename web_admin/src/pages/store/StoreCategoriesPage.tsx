import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { canManage } from '../../auth/roles'
import { localizedName } from '../../i18n/ar'
import { ApiError, api } from '../../lib/api'
import type { Category } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

export function StoreCategoriesPage() {
  const { role } = useAuth()
  const manageable = canManage(role)

  const [categories, setCategories] = useState<Category[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [edit, setEdit] = useState<Category | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', name_ar: '' })

  async function load() {
    setError(null)
    try {
      setCategories(await api.listCategories())
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load categories')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [])

  function startCreate() {
    setEdit(null)
    setForm({ name: '', name_ar: '' })
    setShowForm(true)
  }

  function startEdit(c: Category) {
    setEdit(c)
    setForm({ name: c.name, name_ar: c.name_ar ?? '' })
    setShowForm(true)
  }

  async function save() {
    setError(null)
    if (!form.name.trim()) {
      setError('Name is required')
      return
    }
    const payload = { name: form.name.trim(), name_ar: form.name_ar.trim() }
    setBusy(true)
    try {
      if (edit) {
        await api.updateCategory(edit.id, payload)
      } else {
        await api.createCategory(payload)
      }
      setEdit(null)
      setForm({ name: '', name_ar: '' })
      setShowForm(false)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Save failed')
      }
    } finally {
      setBusy(false)
    }
  }

  async function toggle(c: Category) {
    setBusy(true)
    setError(null)
    try {
      await api.updateCategory(c.id, { is_active: !c.is_active })
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(false)
    }
  }

  async function remove(c: Category) {
    setBusy(true)
    setError(null)
    try {
      await api.deleteCategory(c.id)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Categories</h2>
        {manageable ? (
          <button className="btn primary" onClick={startCreate}>
            New category
          </button>
        ) : null}
      </header>
      <ErrorBanner error={error} />

      {manageable && showForm ? (
        <section className="card plan-form">
          <h3>{edit ? `Edit ${edit.name}` : 'New category'}</h3>
          <div className="form-grid">
            <label>
              Name
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </label>
            <label>
              Arabic name
              <input
                type="text"
                value={form.name_ar}
                onChange={(e) => setForm({ ...form, name_ar: e.target.value })}
              />
            </label>
          </div>
          <div className="form-actions">
            <button
              className="btn primary"
              disabled={busy}
              onClick={() => void save()}
            >
              {busy ? 'Saving…' : edit ? 'Save changes' : 'Create category'}
            </button>
            <button
              className="btn"
              onClick={() => {
                setEdit(null)
                setForm({ name: '', name_ar: '' })
                setShowForm(false)
              }}
            >
              Cancel
            </button>
          </div>
        </section>
      ) : null}

      {!categories.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>الاسم / Name</th>
              <th>English name</th>
              <th>Slug</th>
              <th>الحالة / Status</th>
              {manageable ? <th>إجراءات / Actions</th> : null}
            </tr>
          </thead>
          <tbody>
            {categories.map((c) => (
              <tr key={c.id} className={c.is_active ? '' : 'muted'}>
                <td>
                  <strong>{localizedName(c.name_ar, c.name)}</strong>
                </td>
                <td className="muted">{c.name_ar ? c.name : '—'}</td>
                <td className="muted">{c.slug}</td>
                <td>
                  {c.is_active ? (
                    <span className="badge ok">active</span>
                  ) : (
                    <span className="badge muted">inactive</span>
                  )}
                </td>
                {manageable ? (
                  <td className="actions">
                    <button
                      className="btn small"
                      disabled={busy}
                      onClick={() => startEdit(c)}
                    >
                      Edit
                    </button>
                    <button
                      className="btn small"
                      disabled={busy}
                      onClick={() => void toggle(c)}
                    >
                      {c.is_active ? 'Deactivate' : 'Activate'}
                    </button>
                    <button
                      className="btn small danger"
                      disabled={busy}
                      onClick={() => void remove(c)}
                    >
                      Delete
                    </button>
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}