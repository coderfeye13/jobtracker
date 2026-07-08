import { useState, useEffect } from 'react'
import { getProfile, updateProfile } from '../api.js'

function fmtDate(iso) {
  if (!iso) return null
  return new Date(iso).toLocaleString('de-DE', { dateStyle: 'medium', timeStyle: 'short' })
}

export default function CVModal({ onClose }) {
  const [text, setText] = useState('')
  const [savedText, setSavedText] = useState('')
  const [updatedAt, setUpdatedAt] = useState(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    getProfile()
      .then(p => {
        setText(p.cv_text ?? '')
        setSavedText(p.cv_text ?? '')
        setUpdatedAt(p.updated_at)
      })
      .catch(e => { if (e.status !== 404) setError(e.message) })
      .finally(() => setLoading(false))
  }, [])

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      const p = await updateProfile(text)
      setSavedText(p.cv_text)
      setUpdatedAt(p.updated_at)
    } catch (e) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" style={{ maxWidth: 760 }}>
        <div className="modal-header">
          <h2>My CV</h2>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          {loading ? (
            <div className="muted">Loading…</div>
          ) : (
            <>
              <div className="form-group">
                <label>CV Text</label>
                <textarea
                  className="cv-textarea"
                  value={text}
                  onChange={(e) => setText(e.target.value)}
                  rows={18}
                  placeholder="Paste your CV here — it'll be used for AI scoring and cover letter generation."
                  autoFocus
                />
              </div>
              {updatedAt && (
                <div className="muted" style={{ fontSize: '0.78rem' }}>
                  Last saved: {fmtDate(updatedAt)}
                </div>
              )}
              {!updatedAt && !loading && (
                <div className="hint-box">
                  No CV saved yet. Paste your CV above and click Save.
                </div>
              )}
              {error && <div className="error-banner">⚠ {error}</div>}
            </>
          )}
        </div>

        <div className="modal-footer">
          <button className="btn-ghost" onClick={onClose}>Close</button>
          <button
            className="btn-primary"
            onClick={handleSave}
            disabled={saving || loading || text === savedText || !text.trim()}
          >
            {saving ? 'Saving…' : 'Save CV'}
          </button>
        </div>
      </div>
    </div>
  )
}
