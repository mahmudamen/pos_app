import { useEffect, useState } from 'react'
import { ApiError, api, INVOICE_STATUSES } from '../lib/api'
import type { Invoice, InvoiceStatus } from '../types'
import { INVOICE_STATUS_LABELS } from '../lib/billing'
import { ErrorBanner, InvoiceBadge, Money, Spinner } from '../components/ui'

export function InvoicesPage() {
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<InvoiceStatus | ''>('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

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
      <header className="page-head">
        <h2>Invoices</h2>
        <div className="head-actions">
          <select
            className="status-filter"
            value={status}
            onChange={(e) => { setPage(1); setStatus(e.target.value as InvoiceStatus | '') }}
            aria-label="Filter by status"
          >
            <option value="">All</option>
            {INVOICE_STATUSES.map((s) => (
              <option key={s} value={s}>
                {INVOICE_STATUS_LABELS[s]}
              </option>
            ))}
          </select>
          <span className="muted">{total} total</span>
        </div>
      </header>
      <ErrorBanner error={error} />
      {!invoices.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Date</th>
              <th>Tenant</th>
              <th>Description</th>
              <th>Status</th>
              <th>Amount</th>
              <th>Provider</th>
              <th>Actions</th>
            </tr>
          </thead>
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
                      <button className="btn small danger" disabled={busy !== null} onClick={() => void voidInvoice(inv)}>
                        Void
                      </button>
                    </>
                  ) : inv.status === 'paid' ? (
                    <button className="btn small danger" disabled={busy !== null} onClick={() => void refund(inv)}>
                      Refund
                    </button>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}