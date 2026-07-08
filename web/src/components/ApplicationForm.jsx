import { useState } from 'react'

const SOURCES  = ['linkedin', 'indeed', 'stepstone', 'referral', 'company_site', 'other']
const EMP_TYPES = ['werkstudent', 'fulltime', 'parttime', 'internship']
const PERIODS  = ['hourly', 'monthly', 'yearly']
const STATUSES = ['saved', 'applied', 'interview', 'offer', 'rejected', 'ghosted']

function initForm(initial = {}) {
  const defaults = {
    company: '', position: '', city: '', source: '', url: '',
    employment_type: '', salary_min: '', salary_max: '', salary_period: '',
    status: 'saved', applied_at: '', notes: '', job_description: '',
  }
  const out = { ...defaults }
  for (const [k, v] of Object.entries(initial)) {
    if (v != null) out[k] = String(v)
  }
  return out
}

export default function ApplicationForm({ initial, onSave, onBack }) {
  const [form, setForm] = useState(() => initForm(initial))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)

  const set = (key) => (e) => setForm(f => ({ ...f, [key]: e.target.value }))

  const handleSubmit = async (e) => {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      const data = {}
      for (const [k, v] of Object.entries(form)) {
        if (v === '') continue
        if (k === 'salary_min' || k === 'salary_max') {
          const n = parseFloat(v)
          if (!isNaN(n)) data[k] = n
        } else {
          data[k] = v
        }
      }
      await onSave(data)
    } catch (e) {
      setError(e.message)
      setSaving(false)
    }
  }

  return (
    <form className="app-form" onSubmit={handleSubmit}>
      <div className="modal-body">
        <div className="form-grid">
          <div className="form-group">
            <label>Company *</label>
            <input className="form-input" value={form.company} onChange={set('company')} required />
          </div>
          <div className="form-group">
            <label>Position *</label>
            <input className="form-input" value={form.position} onChange={set('position')} required />
          </div>
          <div className="form-group">
            <label>City</label>
            <input className="form-input" value={form.city} onChange={set('city')} />
          </div>
          <div className="form-group">
            <label>Employment Type</label>
            <select className="form-select" value={form.employment_type} onChange={set('employment_type')}>
              <option value="">—</option>
              {EMP_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Source</label>
            <select className="form-select" value={form.source} onChange={set('source')}>
              <option value="">—</option>
              {SOURCES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Status</label>
            <select className="form-select" value={form.status} onChange={set('status')}>
              {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Salary Min (€)</label>
            <input className="form-input" type="number" min="0" value={form.salary_min} onChange={set('salary_min')} />
          </div>
          <div className="form-group">
            <label>Salary Max (€)</label>
            <input className="form-input" type="number" min="0" value={form.salary_max} onChange={set('salary_max')} />
          </div>
          <div className="form-group">
            <label>Salary Period</label>
            <select className="form-select" value={form.salary_period} onChange={set('salary_period')}>
              <option value="">—</option>
              {PERIODS.map(p => <option key={p} value={p}>{p}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Applied At</label>
            <input className="form-input" type="date" value={form.applied_at} onChange={set('applied_at')} />
          </div>
          <div className="form-group form-full">
            <label>URL</label>
            <input className="form-input" type="url" value={form.url} onChange={set('url')} placeholder="https://…" />
          </div>
        </div>

        <div className="form-group">
          <label>Notes</label>
          <textarea className="form-textarea" rows={3} value={form.notes} onChange={set('notes')} />
        </div>
        <div className="form-group">
          <label>Job Description</label>
          <textarea className="form-textarea" rows={6} value={form.job_description} onChange={set('job_description')} />
        </div>

        {error && <div className="error-banner">⚠ {error}</div>}
      </div>

      <div className="modal-footer">
        {onBack && <button type="button" className="btn-ghost" onClick={onBack}>← Back</button>}
        <button type="submit" className="btn-primary" disabled={saving}>
          {saving ? 'Saving…' : 'Save Application'}
        </button>
      </div>
    </form>
  )
}
