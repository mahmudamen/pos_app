import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from '../../lib/api'
import type { StaffProfile } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

export function CommunityProfilesPage() {
  const [profiles, setProfiles] = useState<StaffProfile[]>([])
  const [me, setMe] = useState<StaffProfile | null>(null)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [q, setQ] = useState('')
  const [chief, setChief] = useState(false)
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const [headline, setHeadline] = useState('')
  const [bio, setBio] = useState('')
  const [location, setLocation] = useState('')
  const [years, setYears] = useState('0')
  const [isChief, setIsChief] = useState(false)
  const [skills, setSkills] = useState('')

  async function load(p: number, search: string, onlyChief: boolean) {
    setError(null)
    try {
      const res = await api.listProfiles(p, 50, search, onlyChief)
      setProfiles(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch {
      setError('Failed to load profiles')
    }
  }

  useEffect(() => {
    void load(1, q, chief)
    void (async () => {
      try {
        const mine = await api.myProfile()
        setMe(mine)
        if (mine) {
          setHeadline(mine.headline ?? '')
          setBio(mine.bio ?? '')
          setLocation(mine.location ?? '')
          setYears(String(mine.years_experience ?? 0))
          setIsChief(mine.is_chief ?? false)
          setSkills((mine.skills ?? []).join(', '))
        }
      } catch {
        // directory still works without the profile
      }
    })()
  }, [])

  function onSearch(e: FormEvent) {
    e.preventDefault()
    void load(1, q, chief)
  }

  function toggleChief() {
    const next = !chief
    setChief(next)
    void load(1, q, next)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    setSaved(false)
    try {
      const yearsNum = parseInt(years, 10)
      const skillList = skills
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      const savedProfile = await api.upsertMyProfile({
        headline,
        bio,
        location,
        years_experience: Number.isNaN(yearsNum) ? 0 : yearsNum,
        is_chief: isChief,
        skills: skillList,
      })
      setMe(savedProfile)
      setEditing(false)
      setSaved(true)
    } catch {
      setError('Failed to save your profile')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Profiles</h2>
        <span className="muted">{total} staff profiles</span>
      </header>
      <ErrorBanner error={error} />

      <div className="card comm-card form-card" style={{ marginBottom: 18 }}>
        <div className="comm-card-head">
          <div>
            <h3>{editing ? 'Edit your profile' : me ? 'Your profile' : 'Create your community profile'}</h3>
            {!editing && me ? (
              <p className="muted small">
                {me.headline || 'No headline yet'} · {me.location || 'No location'} ·{' '}
                {me.years_experience} yrs · {me.skills.length} skills
                {me.is_chief ? ' · Chief' : ''}
              </p>
            ) : !editing ? (
              <p className="muted small">
                Introduce yourself — your headline, skills and experience help companies find you.
              </p>
            ) : null}
          </div>
          {!editing ? (
            <button className="btn small" onClick={() => setEditing(true)}>
              {me ? 'Edit' : 'Create'}
            </button>
          ) : null}
        </div>
        {editing ? (
          <form className="form-grid" onSubmit={onSubmit} style={{ marginTop: 6 }}>
            <div className="form-row span-2">
              <label>
                Headline
                <input value={headline} onChange={(e) => setHeadline(e.target.value)} placeholder="e.g. Shop owner since 2019" />
              </label>
            </div>
            <div className="form-row span-2">
              <label>
                Bio
                <textarea value={bio} onChange={(e) => setBio(e.target.value)} rows={3} placeholder="What do you do?" />
              </label>
            </div>
            <div className="form-row">
              <label>
                Location
                <input value={location} onChange={(e) => setLocation(e.target.value)} placeholder="City, Egypt" />
              </label>
            </div>
            <div className="form-row">
              <label>
                Years of experience
                <input type="number" min={0} value={years} onChange={(e) => setYears(e.target.value)} />
              </label>
            </div>
            <div className="form-row span-2">
              <label>
                Skills (comma separated)
                <input value={skills} onChange={(e) => setSkills(e.target.value)} placeholder="retail, inventory, payments" />
              </label>
            </div>
            <div className="form-row span-2">
              <label className="check">
                <input type="checkbox" checked={isChief} onChange={(e) => setIsChief(e.target.checked)} /> Chief — I own or lead the store
              </label>
            </div>
            <div className="form-actions span-2">
              <button className="btn primary" type="submit" disabled={saving}>
                {saving ? 'Saving…' : 'Save profile'}
              </button>
              <button className="btn" type="button" onClick={() => setEditing(false)}>
                Cancel
              </button>
            </div>
            {saved ? <p className="ok-text span-2">Profile saved.</p> : null}
          </form>
        ) : null}
      </div>

      <div className="head-actions" style={{ marginBottom: 12 }}>
        <form onSubmit={onSearch}>
          <input
            className="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search by name, headline or skill"
          />
        </form>
        <button
          className={`btn small${chief ? ' primary' : ''}`}
          onClick={toggleChief}
          title="Only show chief profiles"
        >
          {chief ? 'Chief ✓' : 'Chief'}
        </button>
      </div>

      {!profiles.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Name</th>
              <th>Headline</th>
              <th>Location</th>
              <th className="num">Years</th>
              <th>Skills</th>
              <th>Role</th>
            </tr>
          </thead>
          <tbody>
            {profiles.map((p) => (
              <tr key={p.user_id}>
                <td>
                  {p.display_name}
                  {p.has_profile ? null : <span className="small muted"> (no profile)</span>}
                </td>
                <td className="muted">{p.headline || '—'}</td>
                <td className="muted">{p.location || '—'}</td>
                <td className="num">{p.years_experience}</td>
                <td className="small">
                  {(p.skills ?? []).slice(0, 4).join(', ') || '—'}
                </td>
                <td>{p.is_chief ? <span className="badge info">Chief</span> : <span className="badge muted">Staff</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {total > 50 ? (
        <div className="pager">
          <button className="btn" disabled={page <= 1} onClick={() => void load(page - 1, q, chief)}>
            Prev
          </button>
          <span>
            Page {page} of {Math.ceil(total / 50)}
          </span>
          <button className="btn" disabled={page >= Math.ceil(total / 50)} onClick={() => void load(page + 1, q, chief)}>
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}