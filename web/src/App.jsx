import { useState, useEffect, useCallback } from 'react'
import { listApplications, updateApplication, deleteApplication, createApplication } from './api.js'
import KanbanBoard from './components/KanbanBoard.jsx'
import DetailPanel from './components/DetailPanel.jsx'
import AddModal from './components/AddModal.jsx'
import CVModal from './components/CVModal.jsx'

export default function App() {
  const [apps, setApps] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [selectedId, setSelectedId] = useState(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [showCVModal, setShowCVModal] = useState(false)

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
  }

  const handleAppScored = (id, scoreResult) => {
    setApps(prev => prev.map(a =>
      a.id === id
        ? { ...a, fit_score: scoreResult.score, score_details: JSON.stringify(scoreResult) }
        : a
    ))
  }

  const selectedApp = apps.find(a => a.id === selectedId) ?? null

  if (loading) return <div className="full-page-state">Loading…</div>
  if (error)   return <div className="full-page-state error-state">Error: {error}</div>

  return (
    <div className="app">
      <header className="app-header">
        <h1>JobTracker</h1>
        <div className="header-actions">
          <button className="btn-secondary" onClick={() => setShowCVModal(true)}>My CV</button>
          <button className="btn-primary"   onClick={() => setShowAddModal(true)}>+ Add Application</button>
        </div>
      </header>

      <main className="app-main">
        <KanbanBoard
          apps={apps}
          onStatusChange={handleStatusChange}
          onCardClick={setSelectedId}
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
    </div>
  )
}
