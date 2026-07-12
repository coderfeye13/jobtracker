import { useState } from 'react'
import { ET_LABELS, formatSalary } from './ApplicationCard.jsx'
import ScoreSection from './ScoreSection.jsx'
import CoverLetterSection from './CoverLetterSection.jsx'
import TailorCVSection from './TailorCVSection.jsx'

const SOURCE_LABELS = {
  linkedin: 'LinkedIn', indeed: 'Indeed', stepstone: 'StepStone',
  referral: 'Referral', company_site: 'Company Site', other: 'Other',
}

export default function DetailPanel({ app, onUpdate, onDelete, onClose, onOpenCV, onAppScored }) {
  const [notes, setNotes] = useState(app.notes ?? '')
  const [saving, setSaving] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const handleSaveNotes = async () => {
    setSaving(true)
    try { await onUpdate(app.id, { notes }) }
    finally { setSaving(false) }
  }

  const salary = formatSalary(app)

  return (
    <div className="detail-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="detail-panel">

        <div className="detail-header">
          <div className="detail-title">
            <h2 className="detail-company">{app.company}</h2>
            <p className="detail-position">{app.position}</p>
          </div>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="detail-body">
          <div className="detail-grid">
            {app.city            && <Field label="City"    value={app.city} />}
            {app.employment_type && <Field label="Type"    value={ET_LABELS[app.employment_type] ?? app.employment_type} />}
            {salary              && <Field label="Salary"  value={salary} />}
            {app.source          && <Field label="Source"  value={SOURCE_LABELS[app.source] ?? app.source} />}
            {app.status          && <Field label="Status"  value={app.status} />}
            {app.applied_at      && <Field label="Applied" value={app.applied_at} />}
            {app.url && (
              <div className="field full-width">
                <span className="field-label">URL</span>
                <a href={app.url} target="_blank" rel="noopener noreferrer" className="field-link">
                  {app.url}
                </a>
              </div>
            )}
          </div>

          {app.job_description && (
            <div className="detail-section">
              <span className="section-label">Job Description</span>
              <pre className="job-desc">{app.job_description}</pre>
            </div>
          )}

          <div className="detail-section">
            <span className="section-label">Notes</span>
            <textarea
              className="notes-textarea"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={4}
              placeholder="Add notes…"
            />
            <button
              className="btn-secondary btn-sm"
              onClick={handleSaveNotes}
              disabled={saving || notes === (app.notes ?? '')}
              style={{ alignSelf: 'flex-start', marginTop: 6 }}
            >
              {saving ? 'Saving…' : 'Save Notes'}
            </button>
          </div>

          <ScoreSection app={app} onOpenCV={onOpenCV} onAppScored={onAppScored} />

          <CoverLetterSection app={app} onOpenCV={onOpenCV} />

          <TailorCVSection app={app} onOpenCV={onOpenCV} />
        </div>

        <div className="detail-footer">
          {confirmDelete ? (
            <div className="confirm-row">
              <span className="muted">Delete this application?</span>
              <button className="btn-danger" onClick={() => onDelete(app.id)}>Yes, delete</button>
              <button className="btn-ghost" onClick={() => setConfirmDelete(false)}>Cancel</button>
            </div>
          ) : (
            <button className="btn-danger-outline" onClick={() => setConfirmDelete(true)}>Delete</button>
          )}
        </div>

      </div>
    </div>
  )
}

function Field({ label, value }) {
  return (
    <div className="field">
      <span className="field-label">{label}</span>
      <span className="field-value">{value}</span>
    </div>
  )
}
