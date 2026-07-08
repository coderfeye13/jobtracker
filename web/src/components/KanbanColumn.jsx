import { useState } from 'react'
import ApplicationCard from './ApplicationCard.jsx'

export default function KanbanColumn({ status, label, color, apps, onStatusChange, onCardClick }) {
  const [dragOver, setDragOver] = useState(false)

  const handleDragOver = (e) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleDrop = (e) => {
    e.preventDefault()
    setDragOver(false)
    const id = parseInt(e.dataTransfer.getData('appId'), 10)
    if (id) onStatusChange(id, status)
  }

  return (
    <div
      className={`kanban-column${dragOver ? ' drag-over' : ''}`}
      onDragOver={handleDragOver}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
    >
      <div className="column-header" style={{ borderLeftColor: color }}>
        <span className="column-title" style={{ color }}>{label}</span>
        <span className="column-count">{apps.length}</span>
      </div>
      <div className="column-cards">
        {apps.map(app => (
          <ApplicationCard
            key={app.id}
            app={app}
            onClick={() => onCardClick(app.id)}
          />
        ))}
      </div>
    </div>
  )
}
