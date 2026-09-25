import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { api } from '../../lib/api'
import type { NationalMember } from '../../types'
import { ErrorBanner, Spinner } from '../../components/ui'

type HubTile = {
  to: string
  title: string
  desc: string
}

const TILES: HubTile[] = [
  {
    to: '/community/profiles',
    title: 'Profiles',
    desc: 'Staff profiles, chief flag, skills and experience across the store community.',
  },
  {
    to: '/community/companies',
    title: 'Companies',
    desc: 'The employers behind the community — who they hire and what they do.',
  },
  {
    to: '/community/jobs',
    title: 'Jobs',
    desc: 'Job offers posted by community companies, with one-click applications.',
  },
  {
    to: '/community/national',
    title: 'Egypt community',
    desc: 'The Egypt-wide national network — join with an invite code and connect with members.',
  },
]

export function CommunityHubPage() {
  const [member, setMember] = useState<NationalMember | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const me = await api.nationalMe()
        if (!cancelled) {
          setMember(me)
        }
      } catch {
        if (!cancelled) {
          setError('Failed to load community status')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="page">
      <header className="page-head">
        <h2>Community</h2>
        <span className="muted">Profiles, companies, jobs &amp; the Egypt network</span>
      </header>
      <ErrorBanner error={error} />
      {loading ? (
        <Spinner />
      ) : (
        <div className="stat-grid inner">
          <div className="stat card">
            <div className="stat-label">National member</div>
            <div className="stat-value">
              {member ? (
                <span className={`badge ${member.status === 'suspended' ? 'bad' : 'ok'}`}>
                  {member.role}
                </span>
              ) : (
                <span className="badge muted">not joined</span>
              )}
            </div>
            {member ? (
              <div className="stat-sub">
                Level {member.level} · {member.expertise_score} expertise
              </div>
            ) : (
              <div className="stat-sub">Join the Egypt network with an invite code</div>
            )}
          </div>
        </div>
      )}
      <div className="comm-grid">
        {TILES.map((t) => (
          <NavLink key={t.to} to={t.to} className="comm-card card">
            <h3>{t.title}</h3>
            <p>{t.desc}</p>
            <span className="comm-arrow">Open →</span>
          </NavLink>
        ))}
      </div>
    </div>
  )
}