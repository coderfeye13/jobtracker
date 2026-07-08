import { useState } from 'react'
import { generateCoverLetter } from '../api.js'

const LANGUAGES = [
  { value: 'de', label: 'Deutsch' },
  { value: 'en', label: 'English' },
  { value: 'tr', label: 'Türkçe' },
]

const TONES = [
  { value: 'formal',   label: 'Formal' },
  { value: 'warm',     label: 'Warm' },
  { value: 'concise',  label: 'Concise' },
]

export default function CoverLetterSection({ app, onOpenCV }) {
  const [language, setLanguage] = useState('de')
  const [tone, setTone] = useState('formal')
  const [letter, setLetter] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [noCV, setNoCV] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleGenerate = async () => {
    setLoading(true)
    setError(null)
    setNoCV(false)
    try {
      const data = await generateCoverLetter(app.id, language, tone)
      setLetter(data.cover_letter)
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
    await navigator.clipboard.writeText(letter)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="detail-section">
      <span className="section-label">Cover Letter</span>

      <div className="cl-controls">
        <select className="form-select form-select-sm" value={language} onChange={e => setLanguage(e.target.value)}>
          {LANGUAGES.map(l => <option key={l.value} value={l.value}>{l.label}</option>)}
        </select>
        <select className="form-select form-select-sm" value={tone} onChange={e => setTone(e.target.value)}>
          {TONES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select>
        <button className="btn-secondary btn-sm" onClick={handleGenerate} disabled={loading}>
          {loading
            ? <><span className="spinner spinner-sm" /> Generating…</>
            : letter ? 'Regenerate' : '✨ Generate'}
        </button>
      </div>

      {noCV && (
        <div className="hint-box">
          <span>No CV uploaded yet.</span>
          <button className="btn-secondary btn-sm" onClick={onOpenCV}>Upload CV →</button>
        </div>
      )}

      {error && <div className="error-banner">⚠ {error}</div>}

      {letter && (
        <div className="cl-result">
          <textarea
            className="cover-letter-textarea"
            value={letter}
            onChange={e => setLetter(e.target.value)}
          />
          <button className="btn-secondary btn-sm" onClick={handleCopy} style={{ alignSelf: 'flex-start' }}>
            {copied ? '✓ Copied!' : 'Copy'}
          </button>
        </div>
      )}
    </div>
  )
}
