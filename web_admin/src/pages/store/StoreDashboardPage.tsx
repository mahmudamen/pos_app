import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, api } from '../../lib/api'
import type { DashboardSummary } from '../../types'
import { formatDate } from '../../lib/billing'
import { ErrorBanner, Money, Spinner, Stat } from '../../components/ui'

export function StoreDashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function load() {
    setError(null)
    try {
      setSummary(await api.dashboardSummary())
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load dashboard')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [])

  if (error) {
    return <ErrorBanner error={error} />
  }
  if (!summary) {
    return <Spinner />
  }

  const currency = summary.recent_sales?.[0]?.currency ?? 'EGP'
  const today = summary.today

  return (
    <div className="page">
      <header className="page-head">
        <h2>Store dashboard</h2>
        <div className="head-actions">
          <span className="muted">{formatDate(summary.date)}</span>
          <Link className="btn" to="/sales">
            View sales
          </Link>
        </div>
      </header>

      <div className="stat-grid">
        <Stat
          label="Today's revenue"
          value={<Money minor={today.revenue_minor} currency={currency} />}
        />
        <Stat label="Sales today" value={String(today.sales_count)} />
        <Stat
          label="Average sale"
          value={<Money minor={today.avg_sale_minor} currency={currency} />}
        />
        <Stat label="Items sold" value={String(today.items_sold)} />
        <Stat
          label="Tax collected"
          value={<Money minor={today.tax_minor} currency={currency} />}
        />
        <Stat
          label="Gross profit"
          value={<Money minor={today.profit_minor} currency={currency} />}
        />
      </div>

      <div className="grid-2">
        <section className="card">
          <h3>Recent sales</h3>
          {summary.recent_sales?.length ? (
            <table className="table">
              <tbody>
                {summary.recent_sales.map((sale) => (
                  <tr key={sale.id}>
                    <td className="small muted">{sale.id.slice(0, 8)}</td>
                    <td className="small">{sale.payment_method}</td>
                    <td className="num">
                      <Money minor={sale.total_minor} currency={sale.currency} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">No sales today.</p>
          )}
        </section>

        <section className="card">
          <h3>Top products</h3>
          {summary.top_products?.length ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Product</th>
                  <th className="num">Qty</th>
                  <th className="num">Revenue</th>
                </tr>
              </thead>
              <tbody>
                {summary.top_products.map((p, i) => (
                  <tr key={`${p.product_name}-${i}`}>
                    <td>
                      {p.product_name}
                      <div className="muted small">{p.sku}</div>
                    </td>
                    <td className="num">{p.quantity}</td>
                    <td className="num">
                      <Money minor={p.revenue_minor} currency={currency} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">No product activity today.</p>
          )}
        </section>
      </div>
    </div>
  )
}