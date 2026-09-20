import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { canManage } from '../../auth/roles'
import { ApiError, api, REASONS } from '../../lib/api'
import type { AdjustmentReason, InventoryAdjustment, Product } from '../../types'
import { formatDate } from '../../lib/billing'
import { ErrorBanner, Spinner } from '../../components/ui'

const REASON_LABELS: Record<AdjustmentReason, string> = {
  damaged: 'Damaged',
  restock: 'Restock',
  count: 'Count',
}

const REASON_COLORS: Record<AdjustmentReason, string> = {
  damaged: 'bad',
  restock: 'ok',
  count: 'info',
}

export function StoreInventoryPage() {
  const { role } = useAuth()
  const manageable = canManage(role)

  const [adjustments, setAdjustments] = useState<InventoryAdjustment[]>([])
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const limit = 50
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [productId, setProductId] = useState('')
  const [reason, setReason] = useState<AdjustmentReason>('restock')
  const [delta, setDelta] = useState('')
  const [note, setNote] = useState('')

  async function load(p: number) {
    setError(null)
    try {
      const res = await api.listInventoryAdjustments(p, limit)
      setAdjustments(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load inventory adjustments')
      }
    }
  }

  useEffect(() => {
    void load(1)
  }, [])

  useEffect(() => {
    void (async () => {
      try {
        const res = await api.listProducts(1, 200)
        setProducts(res.data.filter((p) => p.is_active))
      } catch {
        setProducts([])
      }
    })()
  }, [])

  function startCreate() {
    setProductId('')
    setReason('restock')
    setDelta('')
    setNote('')
    setFormOpen(true)
  }

  async function save() {
    setError(null)
    if (!productId) {
      setError('Choose a product')
      return
    }
    const quantityDelta = parseInt(delta, 10)
    if (!Number.isFinite(quantityDelta) || quantityDelta === 0) {
      setError('Quantity delta must be a non-zero integer')
      return
    }
    setBusy(true)
    try {
      await api.createInventoryAdjustment({
        product_id: productId,
        reason,
        quantity_delta: quantityDelta,
        note: note.trim() || undefined,
      })
      setFormOpen(false)
      await load(page)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Save failed')
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Inventory</h2>
        {manageable && !formOpen ? (
          <button className="btn primary" onClick={startCreate}>
            New adjustment
          </button>
        ) : null}
      </header>
      <ErrorBanner error={error} />

      {manageable && formOpen ? (
        <section className="card plan-form">
          <h3>Inventory adjustment</h3>
          <div className="form-grid">
            <label>
              Product
              <select
                value={productId}
                onChange={(e) => setProductId(e.target.value)}
              >
                <option value="">— Choose —</option>
                {products.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Reason
              <select
                value={reason}
                onChange={(e) =>
                  setReason(e.target.value as AdjustmentReason)
                }
              >
                {REASONS.map((r) => (
                  <option key={r} value={r}>
                    {REASON_LABELS[r]}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Quantity delta (negative = remove)
              <input
                type="number"
                value={delta}
                placeholder="-2 or +5"
                onChange={(e) => setDelta(e.target.value)}
              />
            </label>
            <label>
              Note
              <input
                type="text"
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
            </label>
          </div>
          <div className="form-actions">
            <button
              className="btn primary"
              disabled={busy}
              onClick={() => void save()}
            >
              {busy ? 'Saving…' : 'Save adjustment'}
            </button>
            <button className="btn" onClick={() => setFormOpen(false)}>
              Cancel
            </button>
          </div>
        </section>
      ) : null}

      {!adjustments.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Date</th>
              <th>Product</th>
              <th>Reason</th>
              <th className="num">Delta</th>
              <th>Note</th>
              <th>By</th>
            </tr>
          </thead>
          <tbody>
            {adjustments.map((a) => (
              <tr key={a.id}>
                <td className="muted">{formatDate(a.created_at)}</td>
                <td>
                  {a.product_name}
                  <div className="muted small">{a.sku}</div>
                </td>
                <td>
                  <span className={`badge ${REASON_COLORS[a.reason] ?? 'muted'}`}>
                    {REASON_LABELS[a.reason] ?? a.reason}
                  </span>
                </td>
                <td
                  className={`num ${a.quantity_delta < 0 ? 'muted' : ''}`}
                >
                  {a.quantity_delta > 0 ? `+${a.quantity_delta}` : a.quantity_delta}
                </td>
                <td className="muted">{a.note || '—'}</td>
                <td className="muted">{a.created_by}</td>
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