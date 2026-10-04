import { useEffect, useState } from 'react'
import { ApiError, api, INVOICE_STATUSES } from '../lib/api'
import type { Invoice, InvoiceStatus } from '../types'
import { INVOICE_STATUS_LABELS } from '../lib/billing'
import { EmptyState, ErrorBanner, InvoiceBadge, Money, PageHead, TableSkeleton } from '../components/ui'

export function InvoicesPage() {
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<InvoiceStatus | ''>('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  // Void and refund both move real money, so they ask before they fire.
  const [confirming, setConfirming] = useState<string | null>(null)

  async function load() {
    setError(null)
    try {
      const res = await api.listInvoices(page, 100, status)
      setInvoices(res.data)
      setTotal(res.meta.total)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load invoices')
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [page, status])

  async function pay(inv: Invoice) {
    setBusy(inv.id)
    setError(null)
    try {
      await api.payInvoice(inv.id, inv.provider)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(null)
    }
  }

  async function voidInvoice(inv: Invoice) {
    setBusy(inv.id)
    setError(null)
    setConfirming(null)
    try {
      await api.voidInvoice(inv.id)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(null)
    }
  }

  async function refund(inv: Invoice) {
    setBusy(inv.id)
    setError(null)
    setConfirming(null)
    try {
      await api.refundInvoice(inv.id)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="page">
      <PageHead title="Invoices" sub={`${total} invoices raised across all merchants`}>
        <select
          className="status-filter"
          value={status}
          onChange={(e) => { setPage(1); setStatus(e.target.value as InvoiceStatus | '') }}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          {INVOICE_STATUSES.map((s) => (
            <option key={s} value={s}>
              {INVOICE_STATUS_LABELS[s]}
            </option>
          ))}
        </select>
      </PageHead>
      <ErrorBanner error={error} />
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Date</th>
              <th>Tenant</th>
              <th>Description</th>
              <th>Status</th>
              <th className="num">Amount</th>
              <th>Provider</th>
              <th>Actions</th>
            </tr>
          </thead>
          {loading ? (
            <TableSkeleton rows={5} cols={7} />
          ) : (
          <tbody>
            {invoices.map((inv) => (
              <tr key={inv.id}>
                <td className="muted">{inv.created_at.slice(0, 10)}</td>
                <td>{inv.tenant_name}</td>
                <td>{inv.description || <span className="muted">—</span>}</td>
                <td><InvoiceBadge status={inv.status} /></td>
                <td className="num"><Money minor={inv.amount_minor} currency={inv.currency} muted /></td>
                <td className="muted">{inv.provider}{inv.provider_ref ? ` · ${inv.provider_ref.slice(0, 8)}` : ''}</td>
                <td className="actions">
                  {inv.status === 'open' ? (
                    <>
                      <button className="btn small primary" disabled={busy !== null} onClick={() => void pay(inv)}>
                        Pay
                      </button>
                      {confirming === `void-${inv.id}` ? (
                        <>
                          <button
                            className="btn small danger"
                            disabled={busy !== null}
                            onClick={() => void voidInvoice(inv)}
                          >
                            Confirm void
                          </button>
                          <button className="btn small" onClick={() => setConfirming(null)}>
                            Cancel
                          </button>
                        </>
                      ) : (
                        <button
                          className="btn small danger"
                          disabled={busy !== null}
                          onClick={() => setConfirming(`void-${inv.id}`)}
                        >
                          Void
                        </button>
                      )}
                    </>
                  ) : inv.status === 'paid' ? (
                    confirming === `refund-${inv.id}` ? (
                      <>
                        <button
                          className="btn small danger"
                          disabled={busy !== null}
                          onClick={() => void refund(inv)}
                        >
                          Confirm refund
                        </button>
                        <button className="btn small" onClick={() => setConfirming(null)}>
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button
                        className="btn small danger"
                        disabled={busy !== null}
                        onClick={() => setConfirming(`refund-${inv.id}`)}
                      >
                        Refund
                      </button>
                    )
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
          )}
        </table>
        {!loading && invoices.length === 0 && !error ? (
          <EmptyState
            title={status ? `No ${INVOICE_STATUS_LABELS[status].toLowerCase()} invoices` : 'No invoices yet'}
            hint={
              status
                ? 'This filter is applied by the API across every merchant. Clear it to see all statuses.'
                : 'Invoices appear here as soon as one is raised for a merchant.'
            }
          />
        ) : null}
      </div>
      {total > invoices.length ? (
        <div className="pager">
          <span>
            Showing {invoices.length} of {total}
          </span>
          <div className="pager-controls">
            <button className="btn" disabled={page <= 1 || loading} onClick={() => setPage(page - 1)}>
              Prev
            </button>
            <span className="num">
              {page} / {Math.max(1, Math.ceil(total / 100))}
            </span>
            <button className="btn" disabled={loading} onClick={() => setPage(page + 1)}>
              Next
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}