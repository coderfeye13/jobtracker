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
