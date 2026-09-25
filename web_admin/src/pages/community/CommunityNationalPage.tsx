import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from '../../lib/api'
import { formatDate } from '../../lib/billing'
import type { NationalMember } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

const LEVELS = ['', 'bronze', 'silver', 'gold', 'platinum']

const LEVEL_COLOR: Record<string, string> = {
  bronze: 'muted',
  silver: 'info',
  gold: 'warn',
  platinum: 'ok',
}

function LevelBadge({ level }: { level: string }) {
  return <span className={`badge ${LEVEL_COLOR[level] ?? 'muted'}`}>{level}</span>
}

export function CommunityNationalPage() {
  const [me, setMe] = useState<NationalMember | null>(null)
  const [code, setCode] = useState('')
  const [joinError, setJoinError] = useState<string | null>(null)
  const [members, setMembers] = useState<NationalMember[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [q, setQ] = useState('')
  const [level, setLevel] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function loadMe() {
    try {
      const member = await api.nationalMe()
      setMe(member)
      setLevel('')
      if (member) {
        await load(1, '', '')
      }
    } catch {
      setError('Failed to load membership')
    }
  }

  async function load(p: number, search: string, lvl: string) {
    setError(null)
    try {
      const res = await api.listNationalMembers(p, 20, search, lvl)
      setMembers(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch (err) {
      if (err instanceof Error && /not_a_member/i.test(err.message)) {
        return
      }
      setError('Failed to load members')
    }
  }

  useEffect(() => {
    void loadMe()
  }, [])

  function onSearch(e: FormEvent) {
    e.preventDefault()
    void load(1, q, level)
  }

  function onChangeLevel(l: string) {
    setLevel(l)
    void load(1, q, l)
  }

  async function onJoin(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setJoinError(null)
    setError(null)
    try {
      const member = await api.nationalJoin(code.trim())
      setMe(member)
      setCode('')
      await load(1, q, level)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to join'
      const codeKey =
        /invalid/i.test(message) ? 'Invalid code' : /expired/i.test(message) ? 'Code expired' : message
      setJoinError(codeKey)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Egypt community</h2>
        <span className="muted">{total} national members</span>
      </header>
      <ErrorBanner error={error} />

      {me ? (
        <div className="stat-grid inner">
          <div className="stat card">
            <div className="stat-label">Membership</div>
            <div className="stat-value">
              <LevelBadge level={me.level} /> <span className="muted small">{me.role}</span>
            </div>
            <div className="stat-sub">
              {me.expertise_score} expertise · active since {formatDate(me.joined_at)}
            </div>
          </div>
        </div>
      ) : (
        <div className="card comm-card form-card" style={{ marginBottom: 18 }}>
          <div className="comm-card-head">
            <div>
              <h3>Join the Egypt-wide community</h3>
              <p className="muted small">
                Members connect across stores, share expertise and moderate the national directory.
                You need an invite code from an existing member.
              </p>
            </div>
          </div>
          <form className="form-row" onSubmit={onJoin} style={{ marginTop: 6 }}>
            <label>
              Invite code
              <input
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="EG-…"
                style={{ minWidth: 220 }}
              />
            </label>
            <button className="btn primary" type="submit" disabled={busy || !code.trim()}>
              {busy ? 'Joining…' : 'Join'}
            </button>
          </form>
          {joinError ? <p className="error-banner inline">{joinError}</p> : null}
        </div>
      )}

      <div className="head-actions" style={{ marginBottom: 12 }}>
        <form onSubmit={onSearch}>
          <input
            className="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search members by name"
          />
        </form>
        <select
          className="status-filter"
          value={level}
          onChange={(e) => onChangeLevel(e.target.value)}
          aria-label="Expertise level"
        >
          {LEVELS.map((l) => (
            <option key={l} value={l}>
              {l ? l.toUpperCase() : 'All levels'}
            </option>
          ))}
        </select>
      </div>

      {!members.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Name</th>
              <th>Level</th>
              <th>Role</th>
              <th className="num">Expertise</th>
              <th>Joined</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <tr key={m.user_id}>
                <td>
                  {m.display_name}
                  {m.invited_by_name ? (
                    <div className="small muted">invited by {m.invited_by_name}</div>
                  ) : null}
                </td>
                <td>
                  <LevelBadge level={m.level} />
                </td>
                <td>{m.role}</td>
                <td className="num">{m.expertise_score}</td>
                <td className="muted">{formatDate(m.joined_at)}</td>
                <td>
                  {m.status === 'active' ? (
                    <span className="badge ok">active</span>
                  ) : (
                    <span className="badge bad">{m.status}</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {total > 20 ? (
        <div className="pager">
          <button className="btn" disabled={page <= 1} onClick={() => void load(page - 1, q, level)}>
            Prev
          </button>
          <span>
            Page {page} of {Math.ceil(total / 20)}
          </span>
          <button className="btn" disabled={page >= Math.ceil(total / 20)} onClick={() => void load(page + 1, q, level)}>
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}