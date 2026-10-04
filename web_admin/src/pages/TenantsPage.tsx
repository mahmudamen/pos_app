import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { ApiError, api } from '../lib/api'
import type { Tenant, TenantStatus } from '../types'
import { businessTypeLabel } from '../lib/billing'
import {
  EmptyState,
  ErrorBanner,
  Money,
  PageHead,
  TableSkeleton,
  TenantStatusBadge,
} from '../components/ui'

const LIMIT = 50

const STATUSES: { value: string; label: string }[] = [
  { value: 'active', label: 'Active' },
  { value: 'trial', label: 'Trial' },
  { value: 'suspended', label: 'Suspended' },
  { value: 'closed', label: 'Closed' },
  { value: 'disabled', label: 'Stopped' },
]

const BUSINESS_TYPES = [
  'restaurant',
  'book_store',
  'mobile_shop',
  'computer_shop',
  'grocery',
  'other',
]

/**
 * Usage against the plan's ceiling. `0` means unlimited on every plan, so an
 * unlimited tenant gets no meter and no misleading "7 / 0".
 */
function usageMeter(label: string, used: number, cap: number) {
  if (cap === 0) {
    return (
      <span className="small muted" title={`${label}: ${used} of unlimited`}>
        {used} · unlimited
      </span>
    )
  }
  const pct = Math.min(100, Math.round((used / cap) * 100))
  const atCap = used >= cap
  return (
    <div className="meter-row" style={{ minWidth: 108 }}>
      <span className="meter-value">
        {used}
        <span className="muted"> / {cap}</span>
      </span>
      <div
        className="meter-track"
        role="meter"
        aria-label={label}
        aria-valuenow={used}
        aria-valuemin={0}
        aria-valuemax={cap}
      >
        <div
          className="meter-fill"
          style={{ width: `${pct}%`, ['--meter' as string]: atCap ? 'var(--warn)' : 'var(--accent)' }}
        />
      </div>
    </div>
  )
}

export function TenantsPage() {
  // Filter state lives in the URL so a filtered view is shareable, survives a
  // reload, and can be linked from a Slack thread during an incident.
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const status = params.get('status') ?? ''
  const businessType = params.get('business_type') ?? ''
  const plan = params.get('plan') ?? ''
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1)

  const [tenants, setTenants] = useState<Tenant[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState(q)

  // Keep the input in sync when the URL changes underneath us (chips, reset).
  useEffect(() => {
    setSearch(q)
  }, [q])

  const filters = useMemo(
    () => ({
      q: q || undefined,
      status: status || undefined,
      business_type: businessType || undefined,
      plan: plan || undefined,
    }),
    [q, status, businessType, plan],
  )

  const update = useCallback(
    (changes: Record<string, string>, resetPage = true) => {
      const next = new URLSearchParams(params)
      for (const [key, value] of Object.entries(changes)) {
        if (value) {
          next.set(key, value)
        } else {
          next.delete(key)
        }
      }
      if (resetPage) {
        next.delete('page')
      }
      setParams(next, { replace: true })
    },
    [params, setParams],
  )

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api
      .listTenants(page, LIMIT, filters)
      .then((res) => {
        if (cancelled) {
          return
        }
        setTenants(res.data)
        setTotal(res.meta.total)
        setError(null)
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return
        }
        setError(err instanceof ApiError ? err.message : 'Failed to load tenants')
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [page, filters])

  const pages = Math.max(1, Math.ceil(total / LIMIT))
  const from = total === 0 ? 0 : (page - 1) * LIMIT + 1
  const to = Math.min(page * LIMIT, total)
  const filtered = Boolean(q || status || businessType || plan)
  const activeChips: { key: string; label: string }[] = [
    ...(q ? [{ key: 'q', label: q }] : []),
    ...(status
      ? [{ key: 'status', label: STATUSES.find((s) => s.value === status)?.label ?? status }]
      : []),
    ...(businessType ? [{ key: 'business_type', label: businessTypeLabel(businessType) }] : []),
    ...(plan ? [{ key: 'plan', label: plan }] : []),
  ]

  return (
    <div className="page">
      <PageHead
        title="Tenants"
        sub={`${total} ${total === 1 ? 'merchant' : 'merchants'} on the platform`}
      />

      <div className="filterbar">
        <input
          className="search"
          type="search"
          placeholder="Search name, slug or email"
          aria-label="Search tenants"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              update({ q: search.trim() })
            }
          }}
          onBlur={() => update({ q: search.trim() })}
        />
        <select
          aria-label="Status"
          value={status}
          onChange={(e) => update({ status: e.target.value })}
        >
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <select
          aria-label="Business type"
          value={businessType}
          onChange={(e) => update({ business_type: e.target.value })}
        >
          <option value="">All types</option>
          {BUSINESS_TYPES.map((t) => (
            <option key={t} value={t}>
              {businessTypeLabel(t)}
            </option>
          ))}
        </select>
        <input
          type="search"
          placeholder="Plan code"
          aria-label="Plan code"
          style={{ width: 132 }}
          value={plan}
          onChange={(e) => update({ plan: e.target.value.trim() })}
        />
      </div>

      {activeChips.length > 0 ? (
        <div className="chips" style={{ marginBottom: 'var(--s3)' }}>
          <span className="small muted">Filtering by</span>
          {activeChips.map((chip) => (
            <span className="chip" key={chip.key}>
              <span className="chip-key">{chip.key.replace('_', ' ')}</span>
              {chip.label}
              <button aria-label={`Clear ${chip.key} filter`} onClick={() => update({ [chip.key]: '' })}>
                ×
              </button>
            </span>
          ))}
          <button
            className="btn small"
            onClick={() => setParams(new URLSearchParams(), { replace: true })}
          >
            Reset
          </button>
        </div>
      ) : null}

      <ErrorBanner error={error} />

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Tenant</th>
              <th>Status</th>
              <th>Type</th>
              <th>Plan</th>
              <th>Users</th>
              <th>Products</th>
              <th className="num">Sales</th>
              <th className="num">Revenue</th>
            </tr>
          </thead>
          {loading ? (
            <TableSkeleton rows={6} cols={8} />
          ) : (
            <tbody>
              {tenants.map((t) => (
                <tr key={t.id}>
                  <td>
                    <Link className="table-link" to={`/tenants/${t.id}`}>
                      {t.name}
                    </Link>
                    <div className="muted small">{t.slug}</div>
                  </td>
                  <td>
                    {t.status ? (
                      <TenantStatusBadge status={t.status as TenantStatus} />
                    ) : (
                      <span className="badge muted">unknown</span>
                    )}
                    {t.subscription_status && t.status !== t.subscription_status ? (
                      <div className="small muted" style={{ marginTop: 3 }}>
                        {t.subscription_status}
                      </div>
                    ) : null}
                  </td>
                  <td>{businessTypeLabel(t.business_type)}</td>
                  <td>
                    <span className={t.plan ? 'badge accent' : 'badge muted'}>{t.plan || '—'}</span>
                  </td>
                  <td>{usageMeter('users', t.users, t.max_users)}</td>
                  <td>{usageMeter('products', t.products, t.max_products)}</td>
                  <td className="num">{t.total_sales ?? '—'}</td>
                  <td className="num">
                    {t.revenue_minor === undefined ? (
                      '—'
                    ) : (
                      <Money minor={t.revenue_minor} currency={t.currency_code} />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          )}
        </table>
        {!loading && tenants.length === 0 ? (
          <EmptyState
            title={filtered ? 'No tenants match these filters' : 'No tenants yet'}
            hint={
              filtered
                ? 'The API searched every tenant, not just this page — try a broader search or clear the filters.'
                : 'Tenants appear here once a merchant signs up or an operator creates one.'
            }
            action={
              filtered ? (
                <button
                  className="btn"
                  onClick={() => setParams(new URLSearchParams(), { replace: true })}
                >
                  Clear filters
                </button>
              ) : undefined
            }
          />
        ) : null}
      </div>

      <div className="pager">
        <span>
          {total === 0 ? 'No results' : `${from}–${to} of ${total}`}
        </span>
        <div className="pager-controls">
          <button
            className="btn"
            disabled={page <= 1 || loading}
            onClick={() => update({ page: String(page - 1) }, false)}
          >
            Prev
          </button>
          <span className="num">
            {page} / {pages}
          </span>
          <button
            className="btn"
            disabled={page >= pages || loading}
            onClick={() => update({ page: String(page + 1) }, false)}
          >
            Next
          </button>
        </div>
      </div>
    </div>
  )
}