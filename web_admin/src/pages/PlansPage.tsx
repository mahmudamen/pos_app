import { useEffect, useState } from 'react'
import { ApiError, api } from '../lib/api'
import type { BillingPeriod, Plan } from '../types'
import {
  decimalToMinor,
  formatMoney,
  monthlyEquivalentMinor,
  PERIOD_LABELS,
} from '../lib/billing'
import { ErrorBanner, Spinner } from '../components/ui'

type PlanForm = {
  code: string
  name: string
  description: string
  price: string
  currency: string
  billing_period: BillingPeriod
  features: string
  max_users: string
  max_products: string
}

const EMPTY: PlanForm = {
  code: '',
  name: '',
  description: '',
  price: '',
  currency: 'EGP',
  billing_period: 'monthly',
  features: '',
  max_users: '0',
  max_products: '0',
}

export function PlansPage() {
  const [plans, setPlans] = useState<Plan[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [edit, setEdit] = useState<Plan | null>(null)
  const [form, setForm] = useState<PlanForm>(EMPTY)

  async function load() {
    setError(null)
    try {
      setPlans(await api.listPlans())
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load plans')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [])

  function startCreate() {
    setEdit(null)
    setForm(EMPTY)
  }

  function startEdit(p: Plan) {
    setEdit(p)
    setForm({
      code: p.code,
      name: p.name,
      description: p.description,
      price: (p.price_minor / 100).toFixed(2),
      currency: p.currency,
      billing_period: p.billing_period,
      features: p.features.join(', '),
      max_users: String(p.max_users),
      max_products: String(p.max_products),
    })
  }

  async function save() {
    setError(null)
    if (!form.code.trim() || !form.name.trim()) {
      setError('Code and name are required')
      return
    }
    const priceMinor = decimalToMinor(form.price)
    if (priceMinor === null) {
      setError('Enter a valid price (e.g. 99.00)')
      return
    }
    const payload = {
      code: form.code.trim(),
      name: form.name.trim(),
      description: form.description.trim(),
      price_minor: priceMinor,
      currency: form.currency.trim().toUpperCase() || 'EGP',
      billing_period: form.billing_period,
      features: form.features.split(',').map((f) => f.trim()).filter(Boolean),
      max_users: parseInt(form.max_users, 10) || 0,
      max_products: parseInt(form.max_products, 10) || 0,
    }
    setBusy(true)
    try {
      if (edit) {
        await api.updatePlan(edit.id, { ...payload, code: undefined })
      } else {
        await api.createPlan(payload)
      }
      setEdit(null)
      setForm(EMPTY)
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

  async function deactivate(p: Plan) {
    setBusy(true)
    setError(null)
    try {
      await api.updatePlan(p.id, { is_active: false })
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(false)
    }
  }

  const formEditing = edit !== null || form !== EMPTY || form.code !== ''

  return (
    <div className="page">
      <header className="page-head">
        <h2>Plans</h2>
        <button className="btn primary" onClick={startCreate}>
          New plan
        </button>
      </header>
      <ErrorBanner error={error} />

      {formEditing ? (
        <section className="card plan-form">
          <h3>{edit ? `Edit ${edit.name}` : 'New plan'}</h3>
          <div className="form-grid">
            <label>
              Code
              <input
                type="text"
                value={form.code}
                disabled={!!edit}
                placeholder="cafe"
                onChange={(e) => setForm({ ...form, code: e.target.value })}
              />
            </label>
            <label>
              Name
              <input
                type="text"
                value={form.name}
                placeholder="Café"
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </label>
            <label>
              Price
              <input
                type="text"
                inputMode="decimal"
                value={form.price}
                placeholder="99.00"
                onChange={(e) => setForm({ ...form, price: e.target.value })}
              />
            </label>
            <label>
              Currency
              <input
                type="text"
                maxLength={3}
                value={form.currency}
                placeholder="EGP"
                onChange={(e) => setForm({ ...form, currency: e.target.value })}
              />
            </label>
            <label>
              Billing period
              <select
                value={form.billing_period}
                onChange={(e) =>
                  setForm({ ...form, billing_period: e.target.value as BillingPeriod })
                }
              >
                <option value="monthly">Monthly</option>
                <option value="yearly">Yearly</option>
              </select>
            </label>
            <label>
              Max users (0 = unlimited)
              <input
                type="number"
                min={0}
                value={form.max_users}
                onChange={(e) => setForm({ ...form, max_users: e.target.value })}
              />
            </label>
            <label>
              Max products (0 = unlimited)
              <input
                type="number"
                min={0}
                value={form.max_products}
                onChange={(e) => setForm({ ...form, max_products: e.target.value })}
              />
            </label>
            <label className="span-2">
              Description
              <input
                type="text"
                value={form.description}
                placeholder="Short description"
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </label>
            <label className="span-2">
              Features (comma separated)
              <input
                type="text"
                value={form.features}
                placeholder="pos.basic, inventory, multi-device"
                onChange={(e) => setForm({ ...form, features: e.target.value })}
              />
            </label>
          </div>
          <div className="form-actions">
            <button className="btn primary" disabled={busy} onClick={() => void save()}>
              {busy ? 'Saving…' : edit ? 'Save changes' : 'Create plan'}
            </button>
            <button className="btn" onClick={() => { setEdit(null); setForm(EMPTY) }}>
              Cancel
            </button>
          </div>
        </section>
      ) : null}

      {!plans.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Plan</th>
              <th>Price</th>
              <th>Monthly equiv.</th>
              <th>Period</th>
              <th>Limits</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {plans.map((p) => (
              <tr key={p.id} className={p.is_active ? '' : 'muted'}>
                <td>
                  <strong>{p.code}</strong>
                  <div className="muted small">{p.name}</div>
                  {!p.is_active ? <span className="badge muted">inactive</span> : null}
                </td>
                <td>{formatMoney(p.price_minor, p.currency)}</td>
                <td>
                  {formatMoney(monthlyEquivalentMinor(p.price_minor, p.billing_period), p.currency)}
                </td>
                <td>{PERIOD_LABELS[p.billing_period]}</td>
                <td className="muted">{limits(p)}</td>
                <td>
                  <button className="btn small" disabled={busy} onClick={() => startEdit(p)}>
                    Edit
                  </button>{' '}
                  {p.is_active ? (
                    <button className="btn small danger" disabled={busy} onClick={() => void deactivate(p)}>
                      Deactivate
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

function limits(p: Plan): string {
  const users = p.max_users === 0 ? '∞' : String(p.max_users)
  const products = p.max_products === 0 ? '∞' : String(p.max_products)
  return `${users} users · ${products} products`
}