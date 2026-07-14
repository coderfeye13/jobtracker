# Plan B — jobtracker on Kubernetes

> Not a new repo — this work lives inside the existing jobtracker repository
> (new `deploy/` directory + CI workflow + README section). Keep this file as
> `docs/K8S-PLAN.md` or in your notes until Project A is done.

## Goal

Make jobtracker installable on any Kubernetes cluster with one Helm command,
with images built and published automatically by CI. This turns the repo into
a cloud-native showcase aligned with the Werkstudent postings I target
(Kubernetes, Helm, CI/CD, IaC vocabulary — see the Atos MLOps posting).

## Phases

### Phase K1 — Containerize properly

- Multi-stage Dockerfile for the Go backend (build stage → distroless/alpine
  runtime; CGO-free build pays off here: tiny static binary)
- Dockerfile for the frontend (Vite build → nginx serve) OR embed the built
  frontend into the Go binary (`embed.FS`) — decide and document the tradeoff
- `docker compose up` for local one-command start (also improves the README
  quickstart for non-Go users)
- Volume strategy for `jobtracker.db` (persistence outside the container)

### Phase K2 — Kubernetes manifests → Helm chart

- Start with raw manifests on a local kind cluster: Deployment, Service,
  PersistentVolumeClaim (SQLite file), Secret (GEMINI_API_KEY), Ingress
- Convert to a Helm chart: values for image tag, resources, secrets,
  optional gmail credentials mount
- Note the honest limitation in the chart docs: SQLite ⇒ single replica
  (ReadWriteOnce PVC, no horizontal scaling) — and what a Postgres migration
  would change. Knowing and stating this is interview gold.

### Phase K3 — CI/CD

- GitHub Actions: on push to main → go test + build → docker build →
  push image to GHCR (ghcr.io/coderfeye13/jobtracker)
- Optional second workflow: lint (golangci-lint) + vet as PR checks
- Tag-based releases: pushing v0.x tag publishes a versioned image

### Phase K4 — Prove it on a real cluster

- Deploy to GKE free-tier/autopilot (ACC experience reused) or stay on kind —
  either way: screenshots + a "Deploy on Kubernetes" README section with the
  exact commands
- Optional: basic probes (liveness/readiness) and resource limits — small
  additions, big signal

## Deliverables checklist

- [ ] `Dockerfile` (multi-stage) + `docker-compose.yml`
- [ ] `deploy/chart/` Helm chart with sane defaults
- [ ] `.github/workflows/ci.yml` (test + build + publish to GHCR)
- [ ] README: "Run with Docker" + "Deploy on Kubernetes" sections
- [ ] LinkedIn post #2: "jobtracker is now one `helm install` away" —
      short, screenshot of pods running, link to the chart

## Why this matters (the one-paragraph pitch)

jobtracker already proves product thinking and AI integration; this phase
proves operations: packaging, persistence tradeoffs, secrets handling,
automated delivery. Together they cover the full "build it AND run it" story
that backend/cloud Werkstudent postings ask for.
