import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, api } from '../lib/api'
import type { Tenant } from '../types'
import { businessTypeLabel } from '../lib/billing'
import { ErrorBanner, Spinner } from '../components/ui'

export function TenantsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const limit = 50
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')

  async function load(p: number) {
    setError(null)
    try {
      const res = await api.listTenants(p, limit)
      setTenants(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load tenants')
      }
    }
  }

  useEffect(() => {
    void load(1)
  }, [limit])

  const pages = Math.max(1, Math.ceil(total / limit))
  const filtered = query
    ? tenants.filter((t) =>
        `${t.name} ${t.slug} ${t.business_type} ${t.plan}`
          .toLowerCase()
          .includes(query.toLowerCase()),
      )
    : tenants

  return (
    <div className="page">
      <header className="page-head">
        <h2>Tenants</h2>
        <input
          className="search"
          type="search"
          placeholder="Filter name / slug / type"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </header>
      <ErrorBanner error={error} />
      {!tenants.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Tenant</th>
              <th>Type</th>
              <th>Plan</th>
              <th>Limits</th>
              <th>Usage</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((t) => (
              <tr key={t.id}>
                <td>
                  <Link className="table-link" to={`/tenants/${t.id}`}>
                    {t.name}
                  </Link>
                  <div className="muted small">{t.slug}</div>
                </td>
                <td>{businessTypeLabel(t.business_type)}</td>
                <td>
                  <span className="badge info">{t.plan || '—'}</span>
                </td>
                <td className="muted">
                  {t.max_users === 0 ? '∞' : t.max_users} users ·{' '}
                  {t.max_products === 0 ? '∞' : t.max_products} products
                </td>
                <td className="num">
                  {t.users} / {t.products}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {pages > 1 ? (
        <div className="pager">
          <button
            className="btn"
            disabled={page <= 1}
            onClick={() => void load(page - 1)}
          >
            Prev
          </button>
          <span>
            Page {page} of {pages}
          </span>
          <button
            className="btn"
            disabled={page >= pages}
            onClick={() => void load(page + 1)}
          >
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}