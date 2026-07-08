import { useState } from 'react'
import { parseJobPosting } from '../api.js'
import ApplicationForm from './ApplicationForm.jsx'

export default function AddModal({ onSave, onClose }) {
  const [step, setStep] = useState('input') // 'input' | 'form'
  const [rawText, setRawText] = useState('')
  const [url, setUrl] = useState('')
  const [parsing, setParsing] = useState(false)
  const [parseError, setParseError] = useState(null)
  const [draft, setDraft] = useState(null)

  const handleParse = async () => {
    setParsing(true)
    setParseError(null)
    try {
      const result = await parseJobPosting(rawText, url || undefined)
      setDraft(result)
      setStep('form')
    } catch (e) {
      setParseError(e.message)
    } finally {
      setParsing(false)
    }
  }

  const handleBackToInput = () => {
    setStep('input')
    setDraft(null)
  }

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div className="modal-header">
          <h2>{step === 'input' ? 'Add from Job Posting' : 'Review & Save'}</h2>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        {step === 'input' ? (
          <div className="modal-body">
            <div className="form-group">
              <label>Job Posting Text</label>
              <textarea
                className="posting-textarea"
                value={rawText}
                onChange={(e) => setRawText(e.target.value)}
                placeholder="Paste the full job posting here…"
                rows={12}
                autoFocus
              />
            </div>
            <div className="form-group">
              <label>Source URL <span className="muted">(optional)</span></label>
              <input
                className="form-input"
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://linkedin.com/jobs/…"
              />
            </div>

            {parseError && (
              <div className="error-banner">⚠ {parseError}</div>
            )}

            <div className="input-actions">
              <button
                className="btn-primary"
                onClick={handleParse}
                disabled={!rawText.trim() || parsing}
              >
                {parsing
                  ? <><span className="spinner" /> Parsing…</>
                  : '✨ Parse with AI'}
              </button>
              <button
                className="btn-secondary"
                onClick={() => { setDraft({}); setStep('form') }}
                disabled={parsing}
              >
                Add Manually
              </button>
            </div>
          </div>
        ) : (
          <ApplicationForm
            initial={draft}
            onSave={onSave}
            onBack={handleBackToInput}
          />
        )}
      </div>
    </div>
  )
}
