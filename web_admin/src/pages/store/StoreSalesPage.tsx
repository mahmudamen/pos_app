import { useEffect, useState } from 'react'
import { ApiError, api } from '../../lib/api'
import type { SaleSummary } from '../../types'
import { formatDate } from '../../lib/billing'
import { ErrorBanner, Money, Spinner } from '../../components/ui'

export function StoreSalesPage() {
  const [sales, setSales] = useState<SaleSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const limit = 50
  const [error, setError] = useState<string | null>(null)

  async function load(p: number) {
    setError(null)
    try {
      const res = await api.listSales(p, limit)
      setSales(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load sales')
      }
    }
  }

  useEffect(() => {
    void load(1)
  }, [])

  return (
    <div className="page">
      <header className="page-head">
        <h2>Sales</h2>
        <span className="muted">{total} total</span>
      </header>
      <ErrorBanner error={error} />
      {!sales.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Date</th>
              <th>Reference</th>
              <th>Status</th>
              <th>Payment</th>
              <th className="num">Subtotal</th>
              <th className="num">Discount</th>
              <th className="num">Tax</th>
              <th className="num">Total</th>
            </tr>
          </thead>
          <tbody>
            {sales.map((s) => (
              <tr key={s.id}>
                <td className="muted">{formatDate(s.created_at)}</td>
                <td className="small muted">{s.id.slice(0, 8)}</td>
                <td>{s.status}</td>
                <td>{s.payment_method}</td>
                <td className="num">
                  <Money minor={s.subtotal_minor} currency={s.currency} muted />
                </td>
                <td className="num">
                  <Money minor={s.discount_minor} currency={s.currency} muted />
                </td>
                <td className="num">
                  <Money minor={s.tax_minor} currency={s.currency} muted />
                </td>
                <td className="num">
                  <Money minor={s.total_minor} currency={s.currency} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {Math.ceil(total / limit) > 1 ? (
        <div className="pager">
          <button
            className="btn"
            disabled={page <= 1}
            onClick={() => void load(page - 1)}
          >
            Prev
          </button>
          <span>
            Page {page} of {Math.ceil(total / limit)}
          </span>
          <button
            className="btn"
            disabled={page >= Math.ceil(total / limit)}
            onClick={() => void load(page + 1)}
          >
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}