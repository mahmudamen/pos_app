import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { CommunityHubPage } from './CommunityHubPage'
import { CommunityNationalPage } from './CommunityNationalPage'
import { CommunityProfilesPage } from './CommunityProfilesPage'

const OK = (body: unknown, status = 200) =>
  Promise.resolve(
    new Response(JSON.stringify({ data: body }), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )

const ERR = (code: string, message: string, status: number) =>
  Promise.resolve(
    new Response(JSON.stringify({ error: { code, message, request_id: 'r' } }), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )

describe('community pages', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    localStorage.clear()
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('hub shows community status and links to the district sections', async () => {
    fetchMock
      .mockResolvedValueOnce(ERR('not_found', 'no membership', 404))
      .mockResolvedValueOnce(OK({ data: [], meta: { page: 1, limit: 50, total: 0 } }))
      .mockResolvedValueOnce(ERR('profile_not_found', 'no profile', 404))
    render(
      <MemoryRouter initialEntries={['/community']}>
        <Routes>
          <Route path="/community" element={<CommunityHubPage />} />
          <Route path="/community/profiles" element={<CommunityProfilesPage />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('Egypt community')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText('not joined')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('Profiles'))
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Profiles' })).toBeInTheDocument()
    })
  })

  it('national page joins with an invite code and shows the membership', async () => {
    fetchMock
      .mockResolvedValueOnce(ERR('not_found', 'no membership', 404))
      .mockResolvedValueOnce(
        OK({
          user_id: 'u1',
          display_name: 'Ali',
          role: 'member',
          status: 'active',
          level: 'bronze',
          expertise_score: 10,
          joined_at: '2026-01-01T00:00:00Z',
        }),
      )
      .mockResolvedValueOnce(
        OK({
          data: [
            {
              user_id: 'u2',
              display_name: 'Omar',
              role: 'moderator',
              status: 'active',
              level: 'silver',
              expertise_score: 40,
              joined_at: '2026-01-02T00:00:00Z',
            },
          ],
          meta: { page: 1, limit: 20, total: 1 },
        }),
      )

    render(
      <MemoryRouter>
        <CommunityNationalPage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByPlaceholderText('EG-…')).toBeInTheDocument()
    })
    fireEvent.change(screen.getByPlaceholderText('EG-…'), {
      target: { value: 'EG-ABCDEFGHJKLM' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Join' }))

    await waitFor(() => {
      expect(screen.getByText('Membership')).toBeInTheDocument()
    })
    const joinCall = fetchMock.mock.calls.find((c) => (c[0] as string).endsWith('/join'))
    expect(joinCall).toBeDefined()
    const [, init] = joinCall as [string, RequestInit]
    expect(JSON.parse(String(init.body))).toEqual({ code: 'EG-ABCDEFGHJKLM' })
    expect(screen.getByText('Omar')).toBeInTheDocument()
  })

  it('profiles page lists the directory and searches', async () => {
    fetchMock
      .mockResolvedValueOnce(
        OK({
          data: [
            {
              user_id: 'u1',
              display_name: 'Ali',
              email: 'ali@demo.test',
              headline: 'Shop owner',
              location: 'Cairo',
              years_experience: 8,
              is_chief: true,
              skills: ['retail', 'inventory'],
              has_profile: true,
            },
          ],
          meta: { page: 1, limit: 50, total: 1 },
        }),
      )
      .mockResolvedValueOnce(ERR('profile_not_found', 'no profile', 404))

    render(
      <MemoryRouter>
        <CommunityProfilesPage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('Ali')).toBeInTheDocument()
    })
    expect(screen.getByText('Shop owner')).toBeInTheDocument()
    expect(screen.getAllByText('Chief').length).toBeGreaterThan(0)

    const search = screen.getByPlaceholderText('Search by name, headline or skill')
    fireEvent.change(search, { target: { value: 'retail' } })
    fireEvent.submit(search)
    await waitFor(() => {
      const url = fetchMock.mock.calls[0][0] as string
      expect(url).toContain('profiles?')
    })
  })
})