import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from '../../lib/api'
import { useAuth } from '../../auth/AuthContext'
import { formatDate, formatMoney } from '../../lib/billing'
import type { JobOffer } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

const TYPES = ['', 'full_time', 'part_time', 'contract', 'internship']

function canApply(role: string | null | undefined): boolean {
  return role === 'owner' || role === 'manager' || role === 'saas_admin' || role === 'superadmin'
}

export function CommunityJobsPage() {
  const { role } = useAuth()
  const apply = canApply(role)
  const [jobs, setJobs] = useState<JobOffer[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [q, setQ] = useState('')
  const [type, setType] = useState('')
  const [mine, setMine] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState<Record<string, 'ok' | 'dup' | 'fail'>>({})

  async function load(p: number, search: string, t: string, onlyMine: boolean) {
    setError(null)
    try {
      const res = await api.listJobs(p, 50, search, t, onlyMine)
      setJobs(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch {
      setError('Failed to load jobs')
    }
  }

  useEffect(() => {
    void load(1, '', '', false)
  }, [])

  function onSearch(e: FormEvent) {
    e.preventDefault()
    void load(1, q, type, mine)
  }

  function onChangeType(t: string) {
    setType(t)
    void load(1, q, t, mine)
  }

  function onToggleMine() {
    const next = !mine
    setMine(next)
    void load(1, q, type, next)
  }

  async function applyTo(id: string) {
    if (!apply) {
      return
    }
    setError(null)
    try {
      await api.applyJob(id)
      setDone((prev) => ({ ...prev, [id]: 'ok' }))
      await load(page, q, type, mine)
    } catch (err) {
      const code = err instanceof Error ? err.message : ''
      if (/already|applied/i.test(code)) {
        setDone((prev) => ({ ...prev, [id]: 'dup' }))
      } else {
        setDone((prev) => ({ ...prev, [id]: 'fail' }))
      }
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Jobs</h2>
        <span className="muted">{total} offers from community companies</span>
      </header>
      <ErrorBanner error={error} />

      <div className="head-actions" style={{ marginBottom: 12 }}>
        <form onSubmit={onSearch}>
          <input
            className="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search by title or skill tag"
          />
        </form>
        <select
          className="status-filter"
          value={type}
          onChange={(e) => onChangeType(e.target.value)}
          aria-label="Employment type"
        >
          {TYPES.map((t) => (
            <option key={t} value={t}>
              {t ? t.replace('_', ' ') : 'All types'}
            </option>
          ))}
        </select>
        <button className={`btn small${mine ? ' primary' : ''}`} onClick={onToggleMine}>
          {mine ? 'My applications ✓' : 'My applications'}
        </button>
      </div>

      {!jobs.length && !error ? (
        <Spinner />
      ) : (
        <div className="job-list">
          {jobs.map((j) => {
            const state = done[j.id]
            return (
              <div key={j.id} className="card comm-card">
                <div className="comm-card-head">
                  <div>
                    <h3>{j.title}</h3>
                    <p className="muted small">
                      {j.company_name} · {j.location || 'Remote'}
                      {j.salary_minor > 0 ? ' · ' + formatMoney(j.salary_minor, j.salary_currency) : ''}
                      {j.closes_at ? ' · closes ' + formatDate(j.closes_at) : ''}
                    </p>
                  </div>
                  <div className="job-actions">
                    <span className="badge info">{j.employment_type || 'full_time'}</span>
                    {apply ? (
                      <button
                        className="btn small primary"
                        disabled={j.applied || state === 'ok' || state === 'dup'}
                        onClick={() => void applyTo(j.id)}
                      >
                        {j.applied || state === 'dup'
                          ? 'Applied'
                          : state === 'ok'
                            ? 'Applied ✓'
                            : 'Apply'}
                      </button>
                    ) : (
                      <span className="badge muted">cashier read-only</span>
                    )}
                  </div>
                </div>
                <p className="job-desc">{j.description || 'No description provided.'}</p>
                {j.skill_tags && j.skill_tags.length ? (
                  <div className="tags">
                    {j.skill_tags.map((s) => (
                      <span key={s} className="badge muted">
                        {s}
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>
            )
          })}
        </div>
      )}
      {total > 50 ? (
        <div className="pager">
          <button className="btn" disabled={page <= 1} onClick={() => void load(page - 1, q, type, mine)}>
            Prev
          </button>
          <span>
            Page {page} of {Math.ceil(total / 50)}
          </span>
          <button className="btn" disabled={page >= Math.ceil(total / 50)} onClick={() => void load(page + 1, q, type, mine)}>
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}