import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ApiError, api } from '../lib/api'
import { formatDate } from '../lib/billing'
import type { SupportMessage, SupportStatus, SupportSummary } from '../types'
import { EmptyState, ErrorBanner, PageHead, Spinner, Stat } from '../components/ui'

const STATUSES: SupportStatus[] = ['unread', 'read', 'archived']
const DIRECTIONS = ['inbound', 'outbound'] as const

const STATUS_TONE: Record<SupportStatus, string> = {
  unread: 'warn',
  read: 'ok',
  archived: 'muted',
}

function StatusBadge({ status }: { status: SupportStatus }) {
  return <span className={`badge ${STATUS_TONE[status] ?? 'muted'}`}>{status}</span>
}

/**
 * The platform's shared inbox: the cross-tenant queue the IMAP poller fills,
 * plus compose/reply. Reads keep the operator on one screen — pick a thread
 * on the left, answer it on the right, without losing the list scroll.
 */
export function SupportInboxPage() {
  const [params] = useSearchParams()
  const tenantParam = params.get('tenant') ?? ''
  const [messages, setMessages] = useState<SupportMessage[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [summary, setSummary] = useState<SupportSummary | null>(null)
  const [status, setStatus] = useState('')
  const [direction, setDirection] = useState('')
  const [tenant, setTenant] = useState(tenantParam)
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<SupportMessage | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)

  const [reply, setReply] = useState('')
  const [composing, setComposing] = useState(false)
  const [to, setTo] = useState('')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')

  async function load(p = page, st = status, dir = direction, q = query, ten = tenant) {
    setError(null)
    try {
      const res = await api.listSupportMessages({
        page: p,
        limit: 25,
        status: st,
        direction: dir,
        q,
        tenant_id: ten || undefined,
      })
      setMessages(res.data)
      setTotal(res.meta.total)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to load the inbox')
    } finally {
      setLoading(false)
    }
  }

  async function loadSummary() {
    try {
      setSummary(await api.supportSummary())
    } catch {
      // A summary failure should not blank the queue; it is informational.
    }
  }

  useEffect(() => {
    void load()
    void loadSummary()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function open(message: SupportMessage) {
    setSelected(message)
    setReply('')
    if (message.status === 'unread') {
      try {
        await api.markSupportRead(message.id)
        setMessages((rows) =>
          rows.map((r) => (r.id === message.id ? { ...r, status: 'read' } : r)),
        )
        setSelected({ ...message, status: 'read' })
        void loadSummary()
      } catch {
        // Non-fatal: the reply still works.
      }
    } else {
      try {
        const full = await api.getSupportMessage(message.id)
        setSelected(full)
      } catch {
        // Keep the list row we already have.
      }
    }
  }

  function onSearch(e: FormEvent) {
    e.preventDefault()
    setPage(1)
    void load(1)
  }

  async function setStatusFor(m: SupportMessage, target: SupportStatus) {
    if (m.status === target) {
      return
    }
    setBusy(true)
    setError(null)
    try {
      if (target === 'archived') {
        await api.archiveSupportMessage(m.id)
      } else {
        await api.unarchiveSupportMessage(m.id)
      }
      setMessages((rows) => rows.map((r) => (r.id === m.id ? { ...r, status: target } : r)))
      setSelected((cur) => (cur?.id === m.id ? { ...cur, status: target } : cur))
      await loadSummary()
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to update the message')
    } finally {
      setBusy(false)
    }
  }

  function onFilter(next: { status?: string; direction?: string }) {
    if (next.status !== undefined) {
      setStatus(next.status)
      setPage(1)
      void load(1, next.status, direction, query)
    }
    if (next.direction !== undefined) {
      setDirection(next.direction)
      setPage(1)
      void load(1, status, next.direction, query)
    }
  }

  async function onReply(e: FormEvent) {
    e.preventDefault()
    if (!selected || !reply.trim()) {
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.replySupportMessage(selected.id, { body_text: reply })
      setReply('')
      await load()
      await loadSummary()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to send the reply')
    } finally {
      setBusy(false)
    }
  }

  async function onSend(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.sendSupportMessage({ to, subject, body_text: body })
      setComposing(false)
      setTo('')
      setSubject('')
      setBody('')
      await load(1)
      await loadSummary()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to send the message')
    } finally {
      setBusy(false)
    }
  }

  const pages = Math.max(1, Math.ceil(total / 25))

  return (
    <div className="page">
      <PageHead
        title="Support inbox"
        sub="Messages from merchants, ingested from the support mailbox."
      >
        <button
          className="btn primary"
          onClick={() => setComposing((v) => !v)}
        >
          {composing ? 'Cancel' : 'New message'}
        </button>
      </PageHead>

      <ErrorBanner error={error} />

      {summary ? (
        <div className="stat-grid inner" style={{ marginBottom: 18 }}>
          <Stat label="Unread" value={summary.unread} tone="warn" />
          <Stat label="Read" value={summary.read} tone="accent" />
          <Stat label="Archived" value={summary.archived} />
          <Stat label="Total" value={summary.total} tone="info" />
        </div>
      ) : null}

      {composing ? (
        <form className="card form-card" onSubmit={onSend} style={{ marginBottom: 18 }}>
          <h3>New message</h3>
          <div className="form-row">
            <label>
              To
              <input
                type="email"
                required
                value={to}
                onChange={(e) => setTo(e.target.value)}
                placeholder="merchant@example.com"
              />
            </label>
            <label>
              Subject
              <input
                required
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
              />
            </label>
          </div>
          <label>
            Message
            <textarea
              required
              rows={5}
              value={body}
              onChange={(e) => setBody(e.target.value)}
            />
          </label>
          <div className="head-actions">
            <button className="btn primary" type="submit" disabled={busy}>
              {busy ? 'Sending…' : 'Send'}
            </button>
          </div>
        </form>
      ) : null}

      <div className="head-actions" style={{ marginBottom: 12 }}>
        <form onSubmit={onSearch}>
          <input
            className="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search subject, sender or body"
            aria-label="Search messages"
          />
        </form>
        <input
          className="search"
          style={{ maxWidth: 220 }}
          value={tenant}
          onChange={(e) => {
            setTenant(e.target.value)
            setPage(1)
            void load(1, status, direction, query, e.target.value)
          }}
          placeholder={tenantParam ? `Tenant ${tenantParam}` : 'Filter by tenant id'}
          aria-label="Filter by tenant"
        />
        <select
          className="status-filter"
          value={status}
          onChange={(e) => onFilter({ status: e.target.value })}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <select
          className="status-filter"
          value={direction}
          onChange={(e) => onFilter({ direction: e.target.value })}
          aria-label="Filter by direction"
        >
          <option value="">Both directions</option>
          {DIRECTIONS.map((d) => (
            <option key={d} value={d}>
              {d}
            </option>
          ))}
        </select>
      </div>

      {loading ? (
        <Spinner />
      ) : messages.length === 0 ? (
        <EmptyState
          title="No messages"
          hint="Messages that arrive at the support mailbox land here."
        />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Direction</th>
              <th>From / To</th>
              <th>Subject</th>
              <th>Tenant</th>
              <th>Received</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {messages.map((m) => (
              <tr
                key={m.id}
                onClick={() => void open(m)}
                style={{ cursor: 'pointer' }}
                className={selected?.id === m.id ? 'selected' : undefined}
              >
                <td>
                  <span className={`badge ${m.direction === 'inbound' ? 'info' : 'muted'}`}>
                    {m.direction}
                  </span>
                </td>
                <td>
                  {m.direction === 'inbound' ? m.from_email : m.to_email}
                  {m.from_name ? <div className="small muted">{m.from_name}</div> : null}
                </td>
                <td>
                  {m.subject || '(no subject)'}
                  {m.error ? <div className="small muted">send error: {m.error}</div> : null}
                </td>
                <td>{m.tenant_name ?? <span className="muted">—</span>}</td>
                <td className="muted">{formatDate(m.received_at ?? m.created_at)}</td>
                <td>
                  <StatusBadge status={m.status} />
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
            onClick={() => {
              setPage(page - 1)
              void load(page - 1)
            }}
          >
            Prev
          </button>
          <span>
            Page {page} of {pages}
          </span>
          <button
            className="btn"
            disabled={page >= pages}
            onClick={() => {
              setPage(page + 1)
              void load(page + 1)
            }}
          >
            Next
          </button>
        </div>
      ) : null}

      {selected ? (
        <section className="card form-card" style={{ marginTop: 18 }} aria-label="Message detail">
          <div className="comm-card-head">
            <div>
              <h3>{selected.subject || '(no subject)'}</h3>
              <p className="muted small">
                From {selected.from_email}
                {selected.to_email ? ` → ${selected.to_email}` : ''} ·{' '}
                {formatDate(selected.received_at ?? selected.sent_at ?? selected.created_at)}
                {selected.tenant_name ? ` · ${selected.tenant_name}` : ''}
              </p>
            </div>
            <StatusBadge status={selected.status} />
          </div>
          <div className="head-actions">
            {selected.status === 'archived' ? (
              <button
                className="btn"
                onClick={() => void setStatusFor(selected, 'read')}
                disabled={busy}
              >
                Unarchive
              </button>
            ) : (
              <button
                className="btn"
                onClick={() => void setStatusFor(selected, 'archived')}
                disabled={busy}
              >
                Archive
              </button>
            )}
          </div>
          <pre
            style={{
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              fontFamily: 'inherit',
              margin: '12px 0',
            }}
          >
            {selected.body_text || '(empty message)'}
          </pre>
          <form onSubmit={onReply} className="reply-form">
            <textarea
              rows={4}
              value={reply}
              onChange={(e) => setReply(e.target.value)}
              placeholder={`Reply to ${selected.from_email}`}
              aria-label="Reply body"
            />
            <div className="head-actions">
              <button className="btn primary" type="submit" disabled={busy || !reply.trim()}>
                {busy ? 'Sending…' : 'Send reply'}
              </button>
            </div>
          </form>
        </section>
      ) : null}
    </div>
  )
}
