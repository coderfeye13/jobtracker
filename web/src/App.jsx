import { useState, useEffect, useCallback } from 'react'
import { listApplications, updateApplication, deleteApplication, createApplication, scoreApplication } from './api.js'
import KanbanBoard from './components/KanbanBoard.jsx'
import DetailPanel from './components/DetailPanel.jsx'
import AddModal from './components/AddModal.jsx'
import CVModal from './components/CVModal.jsx'
import InboxModal from './components/InboxModal.jsx'

export default function App() {
  const [apps, setApps] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [selectedId, setSelectedId] = useState(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [showCVModal, setShowCVModal] = useState(false)
  const [showInboxModal, setShowInboxModal] = useState(false)
  const [scoringIds, setScoringIds] = useState(new Set())

  const fetchApps = useCallback(async () => {
    try {
      const data = await listApplications()
      setApps(data ?? [])
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchApps() }, [fetchApps])

  const handleStatusChange = async (id, newStatus) => {
    setApps(prev => prev.map(a => a.id === id ? { ...a, status: newStatus } : a))
    try {
      const updated = await updateApplication(id, { status: newStatus })
      setApps(prev => prev.map(a => a.id === id ? updated : a))
    } catch {
      fetchApps()
    }
  }

  const handleUpdate = async (id, data) => {
    const updated = await updateApplication(id, data)
    setApps(prev => prev.map(a => a.id === id ? updated : a))
    return updated
  }

  const handleDelete = async (id) => {
    await deleteApplication(id)
    setApps(prev => prev.filter(a => a.id !== id))
    setSelectedId(null)
  }

  const handleCreate = async (data) => {
    const created = await createApplication(data)
    setApps(prev => [created, ...prev])
    setShowAddModal(false)
    if (created.job_description?.trim()) {
      scoreInBackground(created.id)
    }
  }

  const handleInboxApplied = (updatedApp) => {
    setApps(prev => prev.map(a => a.id === updatedApp.id ? updatedApp : a))
  }

  const handleAppScored = (id, scoreResult) => {
    setApps(prev => prev.map(a =>
      a.id === id
        ? { ...a, fit_score: scoreResult.score, score_details: JSON.stringify(scoreResult) }
        : a
    ))
  }

  const scoreInBackground = async (id) => {
    setScoringIds(prev => new Set(prev).add(id))
    try {
      const data = await scoreApplication(id)
      handleAppScored(id, data)
    } catch (e) {
      // best-effort: no CV yet (400), AI unavailable (503), etc. — fail silently in the UI,
      // but log so we can tell auto-score apart from "it just never ran"
      console.warn(`auto-score failed for application ${id}:`, e.status, e.message)
    } finally {
      setScoringIds(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  const selectedApp = apps.find(a => a.id === selectedId) ?? null

  if (loading) return <div className="full-page-state">Loading…</div>
  if (error)   return <div className="full-page-state error-state">Error: {error}</div>

  return (
    <div className="app">
      <header className="app-header">
        <h1>JobTracker</h1>
        <div className="header-actions">
          <button className="btn-secondary" onClick={() => setShowInboxModal(true)}>Inbox</button>
          <button className="btn-secondary" onClick={() => setShowCVModal(true)}>My CV</button>
          <button className="btn-primary"   onClick={() => setShowAddModal(true)}>+ Add Application</button>
        </div>
      </header>

      <main className="app-main">
        <KanbanBoard
          apps={apps}
          onStatusChange={handleStatusChange}
          onCardClick={setSelectedId}
          scoringIds={scoringIds}
        />
      </main>

      {selectedApp && (
        <DetailPanel
          key={selectedApp.id}
          app={selectedApp}
          onUpdate={handleUpdate}
          onDelete={handleDelete}
          onClose={() => setSelectedId(null)}
          onOpenCV={() => { setShowCVModal(true) }}
          onAppScored={handleAppScored}
        />
      )}

      {showAddModal && (
        <AddModal onSave={handleCreate} onClose={() => setShowAddModal(false)} />
      )}

      {showCVModal && (
        <CVModal onClose={() => setShowCVModal(false)} />
      )}

      {showInboxModal && (
        <InboxModal onClose={() => setShowInboxModal(false)} onApplied={handleInboxApplied} />
      )}
    </div>
  )
}
