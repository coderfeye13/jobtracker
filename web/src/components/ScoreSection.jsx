import { useState } from 'react'
import { scoreApplication } from '../api.js'
import { scoreCls } from './ApplicationCard.jsx'

function parseDetails(raw) {
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export default function ScoreSection({ app, onOpenCV, onAppScored }) {
  const [result, setResult] = useState(() => parseDetails(app.score_details))
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [noCV, setNoCV] = useState(false)

  const runScore = async () => {
    setLoading(true)
    setError(null)
    setNoCV(false)
    try {
      const data = await scoreApplication(app.id)
      setResult(data)
      onAppScored(app.id, data)
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

  const cls = result ? scoreCls(result.score) : ''

  return (
    <div className="detail-section">
      <div className="section-header-row">
        <span className="section-label">CV Fit Score</span>
        <button
          className="btn-ghost btn-xs"
          onClick={runScore}
          disabled={loading}
        >
          {loading
            ? <><span className="spinner spinner-sm" /> Scoring…</>
            : result ? 'Re-score' : '▶ Score against my CV'}
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
        <div className="score-result">
          <div className={`score-number ${cls}`}>{result.score}<span className="score-unit">/100</span></div>

          {result.matched_keywords?.length > 0 && (
            <div className="score-block">
              <span className="section-label">Matched</span>
              <div className="chips">
                {result.matched_keywords.map(k => <span key={k} className="chip-match">{k}</span>)}
              </div>
            </div>
          )}

          {result.missing_keywords?.length > 0 && (
            <div className="score-block">
              <span className="section-label">Missing</span>
              <div className="chips">
                {result.missing_keywords.map(k => <span key={k} className="chip-miss">{k}</span>)}
              </div>
            </div>
          )}

          {result.suggestions?.length > 0 && (
            <div className="score-block">
              <span className="section-label">Suggestions</span>
              <ul className="suggestions-list">
                {result.suggestions.map((s, i) => <li key={i}>{s}</li>)}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
