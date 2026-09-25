import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from '../../lib/api'
import { useAuth } from '../../auth/AuthContext'
import { canManage } from '../../auth/roles'
import { formatDate } from '../../lib/billing'
import type { Company, CompanyMember, CompanyDetail, StaffProfile } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

export function CommunityCompaniesPage() {
  const { role } = useAuth()
  const write = canManage(role)
  const [companies, setCompanies] = useState<Company[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [q, setQ] = useState('')
  const [selected, setSelected] = useState<CompanyDetail | null>(null)
  const [members, setMembers] = useState<CompanyMember[]>([])
  const [candidates, setCandidates] = useState<StaffProfile[]>([])
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [industry, setIndustry] = useState('')
  const [city, setCity] = useState('')
  const [website, setWebsite] = useState('')

  const [picker, setPicker] = useState<string>('')
  const [memberTitle, setMemberTitle] = useState('')

  async function load(p: number, search: string) {
    setError(null)
    try {
      const res = await api.listCompanies(p, 50, search)
      setCompanies(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch {
      setError('Failed to load companies')
    }
  }

  useEffect(() => {
    void load(1, '')
    void api.listProfiles(1, 100).then((res) => setCandidates(res.data)).catch(() => {})
  }, [])

  function onSearch(e: FormEvent) {
    e.preventDefault()
    void load(1, q)
  }

  async function openDetail(id: string) {
    setError(null)
    const detail = await api.getCompany(id)
    setSelected(detail)
    setMembers(detail?.members ?? [])
    setPicker('')
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      const created = await api.createCompany({
        name,
        industry: industry || undefined,
        city: city || undefined,
        website: website || undefined,
      })
      setCreating(false)
      setName('')
      setIndustry('')
      setCity('')
      setWebsite('')
      await load(1, q)
      await openDetail(created.id)
    } catch {
      setError('Failed to create company')
    }
  }

  async function addMember(e: FormEvent) {
    e.preventDefault()
    if (!selected || !picker) {
      return
    }
    setError(null)
    try {
      await api.addCompanyMember(selected.company.id, {
        user_id: picker,
        title: memberTitle || undefined,
      })
      setPicker('')
      setMemberTitle('')
      await openDetail(selected.company.id)
    } catch {
      setError('Failed to add member')
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Companies</h2>
        <span className="muted">{total} employers in the community</span>
      </header>
      <ErrorBanner error={error} />

      {write ? (
        <div className="card comm-card form-card" style={{ marginBottom: 18 }}>
          <div className="comm-card-head">
            <div>
              <h3>{creating ? 'New company' : 'Add your company'}</h3>
              <p className="muted small">
                Register the store you run so staff can join and companies can post jobs.
              </p>
            </div>
            {!creating ? (
              <button className="btn small primary" onClick={() => setCreating(true)}>
                New company
              </button>
            ) : null}
          </div>
          {creating ? (
            <form className="form-grid" onSubmit={onSubmit} style={{ marginTop: 6 }}>
              <div className="form-row">
                <label>
                  Name
                  <input required value={name} onChange={(e) => setName(e.target.value)} placeholder="XAMLtech" />
                </label>
              </div>
              <div className="form-row">
                <label>
                  Industry
                  <input value={industry} onChange={(e) => setIndustry(e.target.value)} placeholder="Retail / Software" />
                </label>
              </div>
              <div className="form-row">
                <label>
                  City
                  <input value={city} onChange={(e) => setCity(e.target.value)} placeholder="Cairo" />
                </label>
              </div>
              <div className="form-row">
                <label>
                  Website
                  <input value={website} onChange={(e) => setWebsite(e.target.value)} placeholder="https://…" />
                </label>
              </div>
              <div className="form-actions span-2">
                <button className="btn primary" type="submit">
                  Create
                </button>
                <button className="btn" type="button" onClick={() => setCreating(false)}>
                  Cancel
                </button>
              </div>
            </form>
          ) : null}
        </div>
      ) : null}

      <div className="head-actions" style={{ marginBottom: 12 }}>
        <form onSubmit={onSearch}>
          <input
            className="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search companies"
          />
        </form>
      </div>

      {!companies.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Name</th>
              <th>Industry</th>
              <th>City</th>
              <th className="num">Members</th>
              <th>Founded</th>
            </tr>
          </thead>
          <tbody>
            {companies.map((c) => (
              <tr key={c.id}>
                <td>
                  <button className="table-link" onClick={() => void openDetail(c.id)}>
                    {c.name}
                  </button>
                  {c.website ? (
                    <div className="muted small">
                      <a href={c.website} target="_blank" rel="noopener noreferrer">
                        {c.website}
                      </a>
                    </div>
                  ) : null}
                </td>
                <td>{c.industry || '—'}</td>
                <td className="muted">{c.city || '—'}</td>
                <td className="num">{c.members_count}</td>
                <td className="muted">{formatDate(c.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {total > 50 ? (
        <div className="pager">
          <button className="btn" disabled={page <= 1} onClick={() => void load(page - 1, q)}>
            Prev
          </button>
          <span>
            Page {page} of {Math.ceil(total / 50)}
          </span>
          <button className="btn" disabled={page >= Math.ceil(total / 50)} onClick={() => void load(page + 1, q)}>
            Next
          </button>
        </div>
      ) : null}

      {selected ? (
        <div className="card comm-card" style={{ marginTop: 18 }}>
          <div className="comm-card-head">
            <div>
              <h3>{selected.company.name}</h3>
              <p className="muted small">
                {selected.company.industry || '—'} · {selected.company.city || '—'} ·{' '}
                {(selected.company.description || 'No description').slice(0, 120)}
              </p>
            </div>
            <button className="btn small" onClick={() => setSelected(null)}>
              Close
            </button>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>Member</th>
                <th>Role</th>
                <th>Title</th>
                <th>Since</th>
              </tr>
            </thead>
            <tbody>
              {members.map((m) => (
                <tr key={m.id}>
                  <td>{m.display_name}</td>
                  <td>{m.role}</td>
                  <td className="muted">{m.title || '—'}</td>
                  <td className="muted">{formatDate(m.created_at)}</td>
                </tr>
              ))}
              {!members.length ? (
                <tr>
                  <td colSpan={4} className="muted">
                    No members yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
          {write ? (
            <form className="form-row" onSubmit={addMember} style={{ marginTop: 12 }}>
              <label>
                Add member
                <select
                  className="search"
                  value={picker}
                  onChange={(e) => setPicker(e.target.value)}
                >
                  <option value="">Pick a staff profile…</option>
                  {candidates.map((p) => (
                    <option key={p.user_id} value={p.user_id}>
                      {p.display_name} — {p.headline || p.email}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Title
                <input value={memberTitle} onChange={(e) => setMemberTitle(e.target.value)} placeholder="Founder" />
              </label>
              <button className="btn small primary" type="submit" disabled={!picker}>
                Add
              </button>
            </form>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}