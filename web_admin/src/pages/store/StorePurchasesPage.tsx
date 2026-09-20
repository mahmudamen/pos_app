import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { canManage } from '../../auth/roles'
import { ApiError, api } from '../../lib/api'
import type { Purchase, PurchaseItemInput } from '../../types'
import { decimalToMinor, formatDate } from '../../lib/billing'
import { ErrorBanner, Money, Spinner } from '../../components/ui'

type ItemRow = {
  product_name: string
  quantity: string
  unit: string
  unit_price: string
}

const EMPTY_ITEM: ItemRow = {
  product_name: '',
  quantity: '1',
  unit: 'pcs',
  unit_price: '',
}

export function StorePurchasesPage() {
  const { role } = useAuth()
  const manageable = canManage(role)

  const [purchases, setPurchases] = useState<Purchase[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [supplier, setSupplier] = useState('')
  const [invoiceNo, setInvoiceNo] = useState('')
  const [tax, setTax] = useState('')
  const [items, setItems] = useState<ItemRow[]>([{ ...EMPTY_ITEM }])

  async function load() {
    setError(null)
    try {
      setPurchases(await api.listPurchases())
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load purchases')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [])

  function startCreate() {
    setSupplier('')
    setInvoiceNo('')
    setTax('')
    setItems([{ ...EMPTY_ITEM }])
    setFormOpen(true)
  }

  function updateItem(i: number, patch: Partial<ItemRow>) {
    setItems(items.map((row, idx) => (idx === i ? { ...row, ...patch } : row)))
  }

  async function save() {
    setError(null)
    if (!supplier.trim()) {
      setError('Supplier is required')
      return
    }
    const rows: PurchaseItemInput[] = []
    for (const row of items) {
      if (!row.product_name.trim() || !row.quantity.trim() || !row.unit_price.trim()) {
        continue
      }
      const unitPrice = decimalToMinor(row.unit_price)
      if (unitPrice === null) {
        setError(`Enter a valid unit price for ${row.product_name || 'item'}`)
        return
      }
      rows.push({
        product_name: row.product_name.trim(),
        quantity: parseInt(row.quantity, 10) || 0,
        unit: row.unit.trim(),
        unit_price_minor: unitPrice,
      })
    }
    if (rows.length === 0) {
      setError('Add at least one item')
      return
    }
    const taxMinor = tax.trim() === '' ? undefined : decimalToMinor(tax)
    if (taxMinor === null) {
      setError('Enter a valid tax amount')
      return
    }
    setBusy(true)
    try {
      await api.createPurchase({
        supplier: supplier.trim(),
        invoice_no: invoiceNo.trim() || undefined,
        currency: 'EGP',
        items: rows,
        tax_minor: taxMinor,
      })
      setFormOpen(false)
      await load()
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
        <h2>Purchases</h2>
        {manageable && !formOpen ? (
          <button className="btn primary" onClick={startCreate}>
            New purchase
          </button>
        ) : null}
      </header>
      <ErrorBanner error={error} />

      {manageable && formOpen ? (
        <section className="card plan-form">
          <h3>New purchase</h3>
          <div className="form-grid">
            <label>
              Supplier
              <input
                type="text"
                value={supplier}
                onChange={(e) => setSupplier(e.target.value)}
              />
            </label>
            <label>
              Invoice number
              <input
                type="text"
                value={invoiceNo}
                onChange={(e) => setInvoiceNo(e.target.value)}
              />
            </label>
            <label>
              Tax (EGP)
              <input
                type="text"
                inputMode="decimal"
                value={tax}
                placeholder="0.00"
                onChange={(e) => setTax(e.target.value)}
              />
            </label>
          </div>
          <h4>Items</h4>
          <div className="form-grid">
            {items.map((row, i) => (
              <div key={i} className="span-2">
                <div className="form-row">
                  <input
                    type="text"
                    placeholder="Product name"
                    aria-label="Product name"
                    value={row.product_name}
                    onChange={(e) => updateItem(i, { product_name: e.target.value })}
                  />
                  <input
                    type="number"
                    min={1}
                    aria-label="Quantity"
                    value={row.quantity}
                    onChange={(e) => updateItem(i, { quantity: e.target.value })}
                  />
                  <input
                    type="text"
                    aria-label="Unit"
                    value={row.unit}
                    onChange={(e) => updateItem(i, { unit: e.target.value })}
                  />
                  <input
                    type="text"
                    inputMode="decimal"
                    aria-label="Unit price"
                    placeholder="12.50"
                    value={row.unit_price}
                    onChange={(e) => updateItem(i, { unit_price: e.target.value })}
                  />
                  <button
                    className="btn small danger"
                    disabled={items.length <= 1}
                    onClick={() => setItems(items.filter((_, idx) => idx !== i))}
                  >
                    ✕
                  </button>
                </div>
              </div>
            ))}
          </div>
          <div className="form-actions">
            <button
              className="btn small"
              onClick={() => setItems([...items, { ...EMPTY_ITEM }])}
            >
              + Add item
            </button>
            <button
              className="btn primary"
              disabled={busy}
              onClick={() => void save()}
            >
              {busy ? 'Saving…' : 'Save purchase'}
            </button>
            <button className="btn" onClick={() => setFormOpen(false)}>
              Cancel
            </button>
          </div>
        </section>
      ) : null}

      {!purchases.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Date</th>
              <th>Supplier</th>
              <th>Invoice</th>
              <th className="num">Items</th>
              <th className="num">Subtotal</th>
              <th className="num">Tax</th>
              <th className="num">Total</th>
              <th>Entered by</th>
            </tr>
          </thead>
          <tbody>
            {purchases.map((p) => (
              <tr key={p.id}>
                <td className="muted">{formatDate(p.created_at)}</td>
                <td>{p.supplier}</td>
                <td className="muted">{p.invoice_no || '—'}</td>
                <td className="num">{p.item_count}</td>
                <td className="num">
                  <Money minor={p.subtotal_minor} currency={p.currency} muted />
                </td>
                <td className="num">
                  <Money minor={p.tax_minor} currency={p.currency} muted />
                </td>
                <td className="num">
                  <Money minor={p.total_minor} currency={p.currency} />
                </td>
                <td className="muted">{p.created_by}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}