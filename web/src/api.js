export const API_BASE = 'http://localhost:8080/api/v1'

async function handleResponse(r) {
  if (r.status === 204) return null
  const body = await r.json().catch(() => ({ message: 'Request failed' }))
  if (!r.ok) throw Object.assign(new Error(body.message || 'Request failed'), { status: r.status })
  return body
}

export const listApplications = () =>
  fetch(`${API_BASE}/applications`).then(handleResponse)

export const createApplication = (data) =>
  fetch(`${API_BASE}/applications`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  }).then(handleResponse)

export const updateApplication = (id, data) =>
  fetch(`${API_BASE}/applications/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  }).then(handleResponse)

export const deleteApplication = (id) =>
  fetch(`${API_BASE}/applications/${id}`, { method: 'DELETE' }).then(handleResponse)

export const parseJobPosting = (rawText, url) =>
  fetch(`${API_BASE}/ai/parse-job`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ raw_text: rawText, ...(url ? { url } : {}) }),
  }).then(handleResponse)

export const parseJobURL = (url) =>
  fetch(`${API_BASE}/ai/parse-url`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  }).then(handleResponse)

export const getProfile = () =>
  fetch(`${API_BASE}/profile`).then(handleResponse)

export const updateProfile = (cvText) =>
  fetch(`${API_BASE}/profile`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ cv_text: cvText }),
  }).then(handleResponse)

export const scoreApplication = (appId) =>
  fetch(`${API_BASE}/ai/score`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ application_id: appId }),
  }).then(handleResponse)

export const generateCoverLetter = (appId, language, tone) =>
  fetch(`${API_BASE}/ai/cover-letter`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ application_id: appId, language, tone }),
  }).then(handleResponse)

export const tailorCV = (appId, language) =>
  fetch(`${API_BASE}/ai/tailor-cv`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ application_id: appId, language }),
  }).then(handleResponse)

export const syncInbox = () =>
  fetch(`${API_BASE}/inbox/sync`, { method: 'POST' }).then(handleResponse)

export const listInboxEvents = ({ kind, includeDismissed } = {}) => {
  const params = new URLSearchParams()
  if (kind) params.set('kind', kind)
  if (includeDismissed) params.set('include_dismissed', 'true')
  const qs = params.toString()
  return fetch(`${API_BASE}/inbox/events${qs ? `?${qs}` : ''}`).then(handleResponse)
}

export const applyInboxEvent = (id) =>
  fetch(`${API_BASE}/inbox/events/${id}/apply`, { method: 'POST' }).then(handleResponse)

export const dismissInboxEvent = (id) =>
  fetch(`${API_BASE}/inbox/events/${id}/dismiss`, { method: 'POST' }).then(handleResponse)
