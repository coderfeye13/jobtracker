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
- Phase 2: CV fit scoring + cover letter generation
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
