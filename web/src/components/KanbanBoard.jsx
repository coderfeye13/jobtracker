import KanbanColumn from './KanbanColumn.jsx'

export const STATUSES = [
  { key: 'saved',      label: 'Saved',      color: '#6366f1' },
  { key: 'applied',    label: 'Applied',    color: '#3b82f6' },
  { key: 'interview',  label: 'Interview',  color: '#f59e0b' },
  { key: 'offer',      label: 'Offer',      color: '#10b981' },
  { key: 'rejected',   label: 'Rejected',   color: '#ef4444' },
  { key: 'ghosted',    label: 'Ghosted',    color: '#6b7280' },
]

export default function KanbanBoard({ apps, onStatusChange, onCardClick, scoringIds }) {
  return (
    <div className="kanban-board">
      {STATUSES.map(({ key, label, color }) => (
        <KanbanColumn
          key={key}
          status={key}
          label={label}
          color={color}
          apps={apps.filter(a => (a.status ?? 'saved') === key)}
          onStatusChange={onStatusChange}
          onCardClick={onCardClick}
          scoringIds={scoringIds}
        />
      ))}
    </div>
  )
}
