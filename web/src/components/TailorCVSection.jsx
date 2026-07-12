import { useState } from 'react'
import { tailorCV } from '../api.js'

const LANGUAGES = [
  { value: 'de', label: 'Deutsch' },
  { value: 'en', label: 'English' },
  { value: 'tr', label: 'Türkçe' },
]

export default function TailorCVSection({ app, onOpenCV }) {
  const [open, setOpen] = useState(false)
  const [language, setLanguage] = useState('de')
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [noCV, setNoCV] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleTailor = async () => {
    setLoading(true)
    setError(null)
    setNoCV(false)
    try {
      const data = await tailorCV(app.id, language)
      setResult(data)
    } catch (e) {
      if (e.status === 400 && e.message.toLowerCase().includes('cv')) {
        setNoCV(true)
      } else {
        setError(e.message)
      }
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = async () => {
    await navigator.clipboard.writeText(result.tailored_cv)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (!open) {
    return (
      <div className="detail-section">
        <button className="btn-secondary btn-sm" onClick={() => setOpen(true)}>
          🎯 Tailor CV
        </button>
      </div>
    )
  }

  return (
    <div className="detail-section">
      <span className="section-label">Tailor CV</span>

      <div className="cl-controls">
        <select className="form-select form-select-sm" value={language} onChange={e => setLanguage(e.target.value)}>
          {LANGUAGES.map(l => <option key={l.value} value={l.value}>{l.label}</option>)}
        </select>
        <button className="btn-secondary btn-sm" onClick={handleTailor} disabled={loading}>
          {loading
            ? <><span className="spinner spinner-sm" /> Tailoring your CV…</>
            : result ? 'Retailor' : '🎯 Tailor'}
        </button>
      </div>

      {noCV && (
        <div className="hint-box">
          <span>No CV uploaded yet.</span>
          <button className="btn-secondary btn-sm" onClick={onOpenCV}>Upload CV →</button>
        </div>
      )}

      {error && <div className="error-banner">⚠ {error}</div>}

      {result && (
        <div className="tailor-result">
          <div className="hint-box tailor-hint">
            <span>Draft only — review every change before using. Nothing is saved.</span>
          </div>

          <div className="tailor-columns">
            <div className="tailor-cv-col">
              <textarea
                className="cover-letter-textarea tailor-cv-textarea"
                value={result.tailored_cv}
                readOnly
              />
              <button className="btn-secondary btn-sm" onClick={handleCopy} style={{ alignSelf: 'flex-start' }}>
                {copied ? '✓ Copied!' : 'Copy'}
              </button>
            </div>

            <div className="tailor-changes-col">
              <span className="section-label">Changes</span>
              <ul className="changes-list">
                {result.changes.map((c, i) => <li key={i}>{c}</li>)}
              </ul>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
