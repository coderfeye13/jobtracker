export const ET_LABELS = {
  werkstudent: 'Werkstudent',
  fulltime: 'Full-time',
  parttime: 'Part-time',
  internship: 'Internship',
}

export function formatSalary(app) {
  if (!app.salary_min && !app.salary_max) return null
  const fmt = (n) => n.toLocaleString('de-DE', { maximumFractionDigits: 0 })
  const suffix = { hourly: '€/h', monthly: '€/mo', yearly: '€/yr' }[app.salary_period] ?? '€'
  if (app.salary_min && app.salary_max)
    return `${fmt(app.salary_min)} – ${fmt(app.salary_max)} ${suffix}`
  if (app.salary_min) return `${fmt(app.salary_min)}+ ${suffix}`
  return `bis ${fmt(app.salary_max)} ${suffix}`
}

export function scoreCls(score) {
  if (score >= 75) return 'score-green'
  if (score >= 50) return 'score-amber'
  return 'score-red'
}

export default function ApplicationCard({ app, onClick, scoring }) {
  const salary = formatSalary(app)

  return (
    <div
      className="app-card"
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData('appId', app.id)
        e.dataTransfer.effectAllowed = 'move'
      }}
      onClick={onClick}
    >
      <div className="card-top">
        <div className="card-company">{app.company}</div>
        {scoring ? (
          <span className="spinner spinner-sm card-score-spinner" title="Scoring…" />
        ) : app.fit_score != null && (
          <span className={`score-badge ${scoreCls(app.fit_score)}`}>{app.fit_score}</span>
        )}
      </div>
      <div className="card-position">{app.position}</div>
      <div className="card-meta">
        {app.city && <span className="badge badge-city">{app.city}</span>}
        {app.employment_type && (
          <span className="badge badge-type">{ET_LABELS[app.employment_type] ?? app.employment_type}</span>
        )}
      </div>
      {salary && <div className="card-salary">{salary}</div>}
    </div>
  )
}
