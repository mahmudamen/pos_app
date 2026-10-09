import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { SupportInboxPage } from './SupportInboxPage'

function renderPage(initialEntry = '/support') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <SupportInboxPage />
    </MemoryRouter>,
  )
}

const OK = (body: unknown, status = 200) =>
  Promise.resolve(
    new Response(JSON.stringify({ data: body }), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )

const PAGE = (rows: unknown[], meta: Record<string, unknown> = {}) =>
  Promise.resolve(
    new Response(
      JSON.stringify({
        data: rows,
        meta: { page: 1, limit: 25, total: rows.length, request_id: 'r', ...meta },
      }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ),
  )

const inbound = {
  id: 'm1',
  thread_id: 't1',
  direction: 'inbound',
  mailbox: 'support',
  tenant_id: 'ten-1',
  tenant_name: 'Nile Grocers',
  message_id: '<x@y>',
  in_reply_to: null,
  from_email: 'owner@nile.example',
  from_name: 'Nadia',
  to_email: 'support@xamltech.com',
  subject: 'Refund request',
  body_text: 'Please refund order 42.',
  body_html: '',
  status: 'unread',
  error: null,
  received_at: '2026-10-01T09:00:00Z',
  sent_at: null,
  created_at: '2026-10-01T09:00:00Z',
}

const summary = { unread: 1, read: 4, archived: 0, total: 5 }

describe('SupportInboxPage', () => {
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

  it('shows the summary and marks an unread message read on open', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([inbound]))
      .mockResolvedValueOnce(OK(summary))
      .mockResolvedValueOnce(OK({ id: 'm1', status: 'read' }))
      .mockResolvedValueOnce(OK(summary))

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Refund request')).toBeInTheDocument()
    })
    expect(screen.getByText('Nile Grocers')).toBeInTheDocument()
    expect(screen.getByText('Unread')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Refund request'))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/v1/saas/support/messages/m1/read'),
        expect.objectContaining({ method: 'POST' }),
      )
    })
    await waitFor(() => {
      expect(screen.getByText('Please refund order 42.')).toBeInTheDocument()
    })
  })

  it('replies to the selected message and reloads the queue', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([inbound]))
      .mockResolvedValueOnce(OK(summary))
      // opening the unread row marks it read, then refreshes the summary
      .mockResolvedValueOnce(OK({ id: 'm1', status: 'read' }))
      .mockResolvedValueOnce(OK(summary))
      .mockResolvedValueOnce(OK({ id: 'm2' }))
      .mockResolvedValueOnce(PAGE([{ ...inbound, status: 'read' }]))
      .mockResolvedValueOnce(OK({ ...summary, unread: 0 }))

    renderPage()
    await waitFor(() => expect(screen.getByText('Refund request')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Refund request'))
    await waitFor(() => expect(screen.getByLabelText('Reply body')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText('Reply body'), {
      target: { value: 'Refund issued.' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send reply' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/v1/saas/support/messages/m1/reply'),
        expect.objectContaining({ method: 'POST' }),
      )
    })
    const replyCall = fetchMock.mock.calls.find(([url]) =>
      String(url).includes('/m1/reply'),
    )
    expect(replyCall).toBeTruthy()
    const replyInit = replyCall![1] as RequestInit
    expect(JSON.parse(String(replyInit.body))).toEqual({
      body_text: 'Refund issued.',
    })
  })

  it('composes and sends a new message', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([]))
      .mockResolvedValueOnce(OK({ ...summary, total: 0, unread: 0 }))
      .mockResolvedValueOnce(OK({ id: 'm9' }))
      .mockResolvedValueOnce(PAGE([]))
      .mockResolvedValueOnce(OK({ ...summary, total: 1, read: 5 }))

    renderPage()
    await waitFor(() => expect(screen.getByText('No messages')).toBeInTheDocument())

    fireEvent.click(screen.getByRole('button', { name: 'New message' }))
    fireEvent.change(screen.getByLabelText('To'), {
      target: { value: 'merchant@example.com' },
    })
    fireEvent.change(screen.getByLabelText('Subject'), {
      target: { value: 'Welcome' },
    })
    fireEvent.change(screen.getByLabelText('Message'), {
      target: { value: 'Thanks for joining.' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => {
      const sendCall = fetchMock.mock.calls.find(
        ([url, init]) =>
          String(url).endsWith('/saas/support/messages') &&
          (init as RequestInit | undefined)?.method === 'POST',
      )
      expect(sendCall).toBeTruthy()
    })
    const sendCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).endsWith('/saas/support/messages') &&
        (init as RequestInit | undefined)?.method === 'POST',
    )
    expect(sendCall).toBeTruthy()
    const sendInit = sendCall![1] as RequestInit
    expect(JSON.parse(String(sendInit.body))).toEqual({
      to: 'merchant@example.com',
      subject: 'Welcome',
      body_text: 'Thanks for joining.',
    })
  })

  it('sends the status and direction filters to the API', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([inbound]))
      .mockResolvedValueOnce(OK(summary))
      .mockResolvedValueOnce(PAGE([inbound]))
      .mockResolvedValueOnce(PAGE([inbound]))

    renderPage()
    await waitFor(() => expect(screen.getByText('Refund request')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText('Filter by status'), {
      target: { value: 'unread' },
    })
    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([url]) => String(url).includes('status=unread')),
      ).toBe(true)
    })

    fireEvent.change(screen.getByLabelText('Filter by direction'), {
      target: { value: 'inbound' },
    })
    await waitFor(() => {
      const last = String(fetchMock.mock.calls.at(-1)?.[0])
      expect(last).toContain('direction=inbound')
      expect(last).toContain('status=unread')
    })
  })

  it('archives the selected message and unarchives it back to read', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([inbound]))
      .mockResolvedValueOnce(OK(summary))
      // opening the unread row marks it read, then refreshes the summary
      .mockResolvedValueOnce(OK({ id: 'm1', status: 'read' }))
      .mockResolvedValueOnce(OK(summary))
      // archive
      .mockResolvedValueOnce(OK({ id: 'm1', status: 'archived' }))
      .mockResolvedValueOnce(OK({ ...summary, archived: 1, read: 3 }))
      .mockResolvedValueOnce(PAGE([{ ...inbound, status: 'archived' }]))
      // unarchive
      .mockResolvedValueOnce(OK({ id: 'm1', status: 'read' }))
      .mockResolvedValueOnce(OK(summary))
      .mockResolvedValueOnce(PAGE([{ ...inbound, status: 'read' }]))

    renderPage()
    await waitFor(() => expect(screen.getByText('Refund request')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Refund request'))
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Archive' }),
      ).toBeInTheDocument(),
    )

    fireEvent.click(screen.getByRole('button', { name: 'Archive' }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/v1/saas/support/messages/m1/archive'),
        expect.objectContaining({ method: 'POST' }),
      )
    })
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Unarchive' })).toBeInTheDocument(),
    )

    fireEvent.click(screen.getByRole('button', { name: 'Unarchive' }))
    await waitFor(() => {
      const unarchiveCall = fetchMock.mock.calls.find(([url, init]) =>
        String(url).includes('/m1/unarchive') &&
        (init as RequestInit | undefined)?.method === 'POST',
      )
      expect(unarchiveCall).toBeTruthy()
    })
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Archive' })).toBeInTheDocument(),
    )
  })

  it('filters by tenant id and honors a deep-link tenant param', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([inbound]))
      .mockResolvedValueOnce(OK(summary))

    renderPage('/support?tenant=ten-1')
    await waitFor(() => expect(screen.getByText('Refund request')).toBeInTheDocument())

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([url]) => String(url).includes('tenant_id=ten-1')),
      ).toBe(true)
    })
    const input = screen.getByLabelText('Filter by tenant')
    expect(input).toHaveValue('ten-1')

    fireEvent.change(input, { target: { value: 'ten-2' } })
    await waitFor(() => {
      const last = String(fetchMock.mock.calls.at(-1)?.[0])
      expect(last).toContain('tenant_id=ten-2')
    })
  })
})
