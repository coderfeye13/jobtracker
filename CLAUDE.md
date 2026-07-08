# JobTracker

Self-hosted job application tracker. Go backend, contract-first design.

## Architecture

- `api/openapi.yaml` is the source of truth. Never edit `internal/gen/` by hand;
  regenerate with: `oapi-codegen -config oapi-codegen.yaml api/openapi.yaml`
- Stack: Echo v4, GORM + glebarez/sqlite (pure Go), oapi-codegen v2
- Layers: handler (HTTP) -> store (DB). API types (`gen.*`) and DB model
  (`store.Application`) are deliberately separate; mapping lives in
  `internal/handler/mapping.go`
- DB file `jobtracker.db` is gitignored (personal data)

## Roadmap

- Phase 1 (current): CRUD + `POST /ai/parse-job` via Gemini (structured JSON output)
- Phase 1.5 (bonus): `POST /ai/parse-url` — fetch a public job posting URL
  server-side (company career pages, not LinkedIn), extract main content
  (e.g. go-shiori/go-readability), feed into the existing ParseJob pipeline.
  Same AI function, second input channel. Skip pages behind login walls;
  return a clear error suggesting copy-paste instead.
- Phase 2 (DONE): CV profile (PUT/GET /profile), AI fit scoring
  (POST /ai/score — persists fit_score + score_details on the application),
  cover letter generation (POST /ai/cover-letter — draft only, not persisted).
  Backend complete; frontend integration pending.
- Phase 3: Gmail integration (job alert filtering, rejection/progress classification)
- Phase 4+ (idea, not scheduled): Browser extension ("clip to jobtracker") —
  a small Chrome extension that grabs the visible text of the currently open
  job posting (user is already logged in, so no scraping/ToS issues) and
  POSTs it to the local API's parse endpoint. One-click capture from
  LinkedIn/Indeed/StepStone. Reference: how Teal/Simplify extensions work.
  Big "wow" feature for the public GitHub repo; keep out of scope until
  core phases are done.

## Working style — IMPORTANT

I am learning Go with this project. Do not write complete implementations
unprompted. Prefer: explain the approach, give one example, let me write the
rest, then review my code. Point me to compiler errors instead of fixing
everything silently. Exception: boilerplate (mapping, config) can be written fully.

## Commands

- Build: `go build ./...`
- Run: `go run ./cmd/server` (port 8080)
- Deps: `go mod tidy` after import changes

## Frontend (`web/`)

Stack: Vite + React, plain fetch, no state library. CSS variables, dark theme.

```bash
cd web && npm install   # first time only
npm run dev             # http://localhost:5173
```

Dev workflow: run both the Go server and `npm run dev` simultaneously.
CORS is configured for `http://localhost:5173` (Vite default).

Component structure:

- `App.jsx` — state, fetch/update/delete/create handlers
- `components/KanbanBoard.jsx` — 6 status columns
- `components/KanbanColumn.jsx` — HTML5 drag-and-drop drop target
- `components/ApplicationCard.jsx` — draggable card, salary formatter
- `components/DetailPanel.jsx` — right slide-in, notes edit, delete confirm
- `components/AddModal.jsx` — AI parse flow or manual add
- `components/ApplicationForm.jsx` — all ApplicationInput fields
- `api.js` — `API_BASE` constant + fetch helpers (applications, profile, ai/score, ai/cover-letter)
- `components/CVModal.jsx` — GET/PUT /profile; textarea + last-updated display
- `components/ScoreSection.jsx` — POST /ai/score; score number, keyword chips, suggestions; pre-populates from score_details
- `components/CoverLetterSection.jsx` — POST /ai/cover-letter; language/tone selects, editable monospace result, copy button
