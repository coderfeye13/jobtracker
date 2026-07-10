import { useEffect, useState } from 'react'
import { listInboxEvents, syncInbox, applyInboxEvent, dismissInboxEvent } from '../api.js'

const KIND_LABELS = {
  job_alert: 'Job Alert',
  application_update: 'Update',
  irrelevant: 'Irrelevant',
}

function fmtDate(iso) {
  if (!iso) return null
  return new Date(iso).toLocaleString('de-DE', { dateStyle: 'medium', timeStyle: 'short' })
}

export default function InboxModal({ onClose, onApplied }) {
  const [events, setEvents] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [syncing, setSyncing] = useState(false)
  const [toast, setToast] = useState(null)
  const [confirmApplyId, setConfirmApplyId] = useState(null)
  const [busyId, setBusyId] = useState(null)

  const loadEvents = () => {
    setError(null)
    return listInboxEvents()
      .then(data => setEvents(data ?? []))
      .catch(e => setError(e.message))
  }

  useEffect(() => {
    loadEvents().finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!toast) return
    const t = setTimeout(() => setToast(null), 3500)
    return () => clearTimeout(t)
  }, [toast])

  const handleSync = async () => {
    setSyncing(true)
    setError(null)
    try {
      const res = await syncInbox()
      await loadEvents()
      setToast(`${res.new_events} new event${res.new_events === 1 ? '' : 's'}`)
    } catch (e) {
      setError(e.message)
    } finally {
      setSyncing(false)
    }
  }

  const handleDismiss = async (id) => {
    setBusyId(id)
    try {
      await dismissInboxEvent(id)
      setEvents(prev => prev.filter(e => e.id !== id))
    } catch (e) {
      setError(e.message)
    } finally {
      setBusyId(null)
    }
  }

  const handleApply = async (id) => {
    setBusyId(id)
    try {
      const updatedApp = await applyInboxEvent(id)
      setEvents(prev => prev.filter(e => e.id !== id))
      setConfirmApplyId(null)
      onApplied?.(updatedApp)
    } catch (e) {
      setError(e.message)
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" style={{ maxWidth: 620 }}>
        <div className="modal-header">
          <h2>Inbox</h2>
          <button className="btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          <div className="inbox-toolbar">
            <span className="muted">{events.length} event{events.length === 1 ? '' : 's'}</span>
            <button className="btn-secondary btn-sm" onClick={handleSync} disabled={syncing}>
              {syncing ? <><span className="spinner spinner-sm" /> Syncing…</> : '⟳ Sync now'}
            </button>
          </div>

          {error && <div className="error-banner">⚠ {error}</div>}

          {loading ? (
            <div className="muted">Loading…</div>
          ) : events.length === 0 ? (
            <div className="empty-state">No inbox events yet. Click "Sync now" to check for new mail.</div>
          ) : (
            <div className="inbox-list">
              {events.map(e => (
                <div key={e.id} className="inbox-event">
                  <div className="inbox-event-top">
                    <span className={`kind-badge kind-${e.kind}`}>{KIND_LABELS[e.kind] ?? e.kind}</span>
                    <span className="inbox-event-date">{fmtDate(e.received_at)}</span>
                  </div>
                  <div className="inbox-event-from">{e.from}</div>
                  <div className="inbox-event-subject">{e.subject}</div>
                  {e.summary && <p className="inbox-event-summary">{e.summary}</p>}

                  <div className="inbox-event-actions">
                    {e.kind === 'application_update' && e.application_id && e.suggested_status && (
                      confirmApplyId === e.id ? (
                        <span className="confirm-row">
                          <span className="muted">Apply?</span>
                          <button
                            className="btn-primary btn-sm"
                            onClick={() => handleApply(e.id)}
                            disabled={busyId === e.id}
                          >
                            {busyId === e.id ? 'Applying…' : 'Yes, apply'}
                          </button>
                          <button className="btn-ghost btn-sm" onClick={() => setConfirmApplyId(null)}>Cancel</button>
                        </span>
                      ) : (
                        <button
                          className="btn-secondary btn-sm"
                          onClick={() => setConfirmApplyId(e.id)}
                          disabled={busyId === e.id}
                        >
                          Apply suggested status: {e.suggested_status}
                        </button>
                      )
                    )}
                    <button
                      className="btn-ghost btn-sm"
                      onClick={() => handleDismiss(e.id)}
                      disabled={busyId === e.id}
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {toast && <div className="toast">{toast}</div>}
    </div>
  )
}
