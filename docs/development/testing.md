# Testing

## Overview

The repository now uses a layered test matrix:

- Unit tests: default `go test ./...`
- Core race tests: `go test -race ./server/internal/core -count=1 -timeout 300s`
- Integration tests: explicit `integration` build tag
- Mock implant E2E tests: explicit `mockimplant` build tag
- Real implant E2E tests: explicit `realimplant` build tag, kept in a manual workflow
- Stress tests: reserved for future `stress`-tagged suites

PR CI runs unit tests, the targeted core race suite, the client/server integration suite, the mock implant suite, and the core testing inventory command. The real implant suite stays out of the default blocking pipeline because it requires a Windows runner plus external implant binaries. Stress tests are intentionally out of scope for the current pipeline.

The long-lived coverage plan lives in `docs/development/core-testing-roadmap.md`.
The machine-readable inventory source of truth lives in `docs/development/core-testing-manifest.json`.

## Local Commands

Run the default CI-equivalent checks:

```bash
go mod tidy
go vet ./...
go run ./scripts/testinventory -output dist/testing
go test ./... -count=1 -timeout 300s
CGO_ENABLED=0 go build ./...
```

Run the client/server integration suite:

```bash
packages=$(go run ./scripts/testmatrix -layer integration)
go test -tags=integration $packages -count=1 -timeout 300s
```

Run the core race guard for concurrent state/session regressions:

```bash
go test -race ./server/internal/core -count=1 -timeout 300s
```

Run the mock implant task E2E guard:

```bash
packages=$(go run ./scripts/testmatrix -layer mockimplant)
go test -tags=mockimplant $packages -count=1 -timeout 300s
```

Run the real implant suite locally:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
$packages = go run ./scripts/testmatrix -layer realimplant -format lines
go test -tags=realimplant $packages -count=1 -timeout 600s
```

Run the workflow locally with `act`:

```bash
act pull_request -W .github/workflows/ci.yaml
```

## Tagged Package Discovery

The CI workflow does not hardcode tagged package lists anymore. It discovers test
packages directly from build tags:

```bash
go run ./scripts/testmatrix -layer integration -format lines
go run ./scripts/testmatrix -layer mockimplant -format lines
go run ./scripts/testmatrix -layer realimplant -format lines
```

This avoids workflow drift when new tagged tests are added under existing
layers.

## Inventory Command

The inventory command scans repository packages, classifies test files by layer, and compares them against the core manifest.

Run it with:

```bash
go run ./scripts/testinventory -output dist/testing
```

The generated report includes:

- package-level test presence and layer classification
- core component status against expected layers
- chain-level missing-layer summaries
- a top gap list for broad package blind spots

Use the report as a recommendation engine. The intended review order is:

1. Tier-1 components with no direct coverage
2. Tier-1 components missing expected layers
3. chains with unresolved missing layers
4. broad package gaps that are not yet in the manifest

## Test Layout

- `client/core`: client-side state handling
- `client/command`: command-first conformance coverage for implant-facing CLI commands
- `server/rpc`: control-plane routing, authorization matching, and listener/pipeline resolution
- `helper/intl`: Lua bundle validation and embedded resource loading
- `server`: client/server integration entrypoint
- `server/testsupport`: reusable mTLS/gRPC harness for integration tests and mock implant E2E coverage

## Notes

- Integration tests use a real gRPC server, real mTLS certificates, and a lightweight fake listener control loop. This keeps authentication and state-sync behavior realistic without requiring implants or external processes.
- `server/internal/core` now has dedicated guards around task recovery, cache trimming, listener/job runtime state, secure rotation counters, and db-only session recovery through the real listener `Checkin` path.
- The mock implant harness adds a deeper task-path layer at `ListenerRPC/SpiteStream`. It is documented in `docs/tests/mock-implant-e2e.md`.
- The `realimplant` workflow is manual by design. It expects a self-hosted Windows runner and repository variables `MALICE_REAL_IMPLANT_BIN` plus `MALICE_REAL_IMPLANT_MUTANT`. `MALICE_REAL_IMPLANT_WORKSPACE` remains optional.
- Command conformance tests are documented in `docs/development/command-testing.md`.
- Detailed test records live under `docs/tests/`.
- Control-plane regression findings are tracked in `docs/tests/control-plane-regression-record.md`.
- `helper/intl` tests depend on the community Lua/resource bundle. When that bundle is not present in the checkout, the suite skips explicitly instead of failing nondeterministically.
- Local coverage collection on some Windows environments can be blocked by antivirus when Go writes instrumented temporary files. Coverage is useful for analysis, but it is not the sole CI gate.

## Manual And Conditional Suites

The repository also contains test mechanisms that are intentionally not part of
the default PR gate:

- `realimplant`: real listener plus real `malefic.exe` process; runs through `.github/workflows/realimplant.yaml`
- `client/command/armory` and `client/command/mal` real GitHub smoke tests; require `MALICE_REAL_GITHUB_TESTS=1` and rely on live upstream availability
- `server/internal/llm` live provider tests; require `MAL_AGENT_E2E_API_KEY` and a reachable model endpoint
- future `stress` suites and benchmark-style probes; useful for analysis, but not stable blocking gates
