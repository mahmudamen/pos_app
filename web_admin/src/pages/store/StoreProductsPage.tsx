import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { canManage } from '../../auth/roles'
import { localizedName, localizedUnit } from '../../i18n/ar'
import { ApiError, api } from '../../lib/api'
import type { Category, Product } from '../../types'
import { decimalToMinor } from '../../lib/billing'
import { ErrorBanner, Money, Spinner } from '../../components/ui'

type ProductForm = {
  name: string
  name_ar: string
  description: string
  description_ar: string
  sku: string
  barcode: string
  price: string
  category_id: string
}

const EMPTY: ProductForm = {
  name: '',
  name_ar: '',
  description: '',
  description_ar: '',
  sku: '',
  barcode: '',
  price: '',
  category_id: '',
}

export function StoreProductsPage() {
  const { role } = useAuth()
  const manageable = canManage(role)

  const [products, setProducts] = useState<Product[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const limit = 50
  const [query, setQuery] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [edit, setEdit] = useState<Product | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<ProductForm>(EMPTY)

  async function load(p: number, q = query) {
    setError(null)
    try {
      const res = await api.listProducts(p, limit, q)
      setProducts(res.data)
      setTotal(res.meta.total)
      setPage(p)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load products')
      }
    }
  }

  useEffect(() => {
    void load(1)
  }, [])

  useEffect(() => {
    void (async () => {
      try {
        setCategories(await api.listCategories())
      } catch {
        setCategories([])
      }
    })()
  }, [])

  function startCreate() {
    setEdit(null)
    setForm(EMPTY)
    setShowForm(true)
  }

  function startEdit(p: Product) {
    setEdit(p)
    setForm({
      name: p.name,
      name_ar: p.name_ar ?? '',
      description: p.description ?? '',
      description_ar: p.description_ar ?? '',
      sku: p.sku ?? '',
      barcode: p.barcode ?? '',
      price: (p.price_minor / 100).toFixed(2),
      category_id: p.category_id ?? '',
    })
    setShowForm(true)
  }

  async function save() {
    setError(null)
    if (!form.name.trim()) {
      setError('Name is required')
      return
    }
    const priceMinor = decimalToMinor(form.price)
    if (priceMinor === null) {
      setError('Enter a valid price (e.g. 12.50)')
      return
    }
    const payload = {
      name: form.name.trim(),
      name_ar: form.name_ar.trim(),
      description: form.description.trim(),
      description_ar: form.description_ar.trim(),
      sku: form.sku.trim(),
      barcode: form.barcode.trim(),
      price_minor: priceMinor,
      currency: 'EGP',
      category_id: form.category_id || undefined,
      is_active: true,
    }
    setBusy(true)
    try {
      if (edit) {
        await api.updateProduct(edit.id, payload)
      } else {
        await api.createProduct(payload)
      }
      setEdit(null)
      setForm(EMPTY)
      setShowForm(false)
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
        <h2>Products</h2>
        <div className="head-actions">
          <input
            className="search"
            type="search"
            placeholder="Search products"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              void load(1, e.target.value)
            }}
          />
          {manageable ? (
            <button className="btn primary" onClick={startCreate}>
              New product
            </button>
          ) : null}
        </div>
      </header>
      <ErrorBanner error={error} />

      {manageable && showForm ? (
        <section className="card plan-form">
          <h3>{edit ? `Edit ${edit.name}` : 'New product'}</h3>
          <div className="form-grid">
            <label>
              Name
              <input
                type="text"
                value={form.name}
                placeholder="Product name"
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </label>
            <label>
              Arabic name
              <input
                type="text"
                value={form.name_ar}
                placeholder="الاسم بالعربية"
                onChange={(e) => setForm({ ...form, name_ar: e.target.value })}
              />
            </label>
            <label>
              Price (EGP)
              <input
                type="text"
                inputMode="decimal"
                value={form.price}
                placeholder="12.50"
                onChange={(e) => setForm({ ...form, price: e.target.value })}
              />
            </label>
            <label>
              Category
              <select
                value={form.category_id}
                onChange={(e) => setForm({ ...form, category_id: e.target.value })}
              >
                <option value="">— None —</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              SKU
              <input
                type="text"
                value={form.sku}
                onChange={(e) => setForm({ ...form, sku: e.target.value })}
              />
            </label>
            <label>
              Barcode
              <input
                type="text"
                value={form.barcode}
                onChange={(e) => setForm({ ...form, barcode: e.target.value })}
              />
            </label>
            <label className="span-2">
              Description
              <input
                type="text"
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </label>
            <label className="span-2">
              Arabic description
              <input
                type="text"
                value={form.description_ar}
                onChange={(e) =>
                  setForm({ ...form, description_ar: e.target.value })
                }
              />
            </label>
          </div>
          <div className="form-actions">
            <button
              className="btn primary"
              disabled={busy}
              onClick={() => void save()}
            >
              {busy ? 'Saving…' : edit ? 'Save changes' : 'Create product'}
            </button>
            <button
              className="btn"
              onClick={() => {
                setEdit(null)
                setForm(EMPTY)
                setShowForm(false)
              }}
            >
              Cancel
            </button>
          </div>
        </section>
      ) : null}

      {!products.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>المنتج / Product</th>
              <th>English name</th>
              <th>السعر / Price</th>
              <th>المخزون / Stock</th>
              <th>الحالة / Status</th>
              {manageable ? <th>إجراءات / Actions</th> : null}
            </tr>
          </thead>
          <tbody>
            {products.map((p) => (
              <tr key={p.id} className={p.is_active ? '' : 'muted'}>
                <td>
                  <strong>{localizedName(p.name_ar, p.name)}</strong>
                  <div className="muted small">
                    {p.sku}
                    {p.barcode ? ` · ${p.barcode}` : ''}
                    {p.unit ? ` · ${localizedUnit(p.unit)}` : ''}
                  </div>
                </td>
                <td className="muted">{p.name_ar ? p.name : '—'}</td>
                <td className="num">
                  <Money minor={p.price_minor} currency={p.currency} />
                </td>
                <td className="num">{p.stock_quantity}</td>
                <td>
                  {p.is_active ? (
                    <span className="badge ok">active</span>
                  ) : (
                    <span className="badge muted">inactive</span>
                  )}
                </td>
                {manageable ? (
                  <td className="actions">
                    <button
                      className="btn small"
                      disabled={busy}
                      onClick={() => startEdit(p)}
                    >
                      Edit
                    </button>
                  </td>
                ) : null}
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