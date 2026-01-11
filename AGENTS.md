# AGENTS

Owner: Josh. Style: telegraph; short clauses ok.

Repo: ref-based browser automation. Go primary path only (single binary `dev-browser-go` with internal daemon flag). Snapshot JS vendored under `internal/devbrowser/snapshot_assets*`.

Usage:
- Go: `go build ./cmd/dev-browser-go` and `./dev-browser-go goto https://example.com` (spawns daemon via `--daemon` internally). One binary only. Artifacts/state under platform cache/state `dev-browser-go/<profile>/`.

Behavior: ref snapshots via injected JS (simple/aria engines). Crop clamp 2000x2000; anything larger downscales poorly in common models. `DEV_BROWSER_WINDOW_SIZE` sets viewport/screen. `HEADLESS` default via env; `DEV_BROWSER_PROFILE` selects profile. Paths locked to caches/state dirs; `DEV_BROWSER_ALLOW_UNSAFE_PATHS=1` to write elsewhere.

Packaging: Go-only. Single binary `dev-browser-go` (daemon via `--daemon`). Flake outputs Go binary only.

Testing: use nix develop (Playwright browsers present) then `HEADLESS=1 ./dev-browser-go goto https://example.com` and `./dev-browser-go snapshot`. Run `go test ./...`. Keep files <500 LOC.

CLI: clig.dev compliant. `--help`, `--version`, subcommand help all work. See SKILL.md for usage workflows, README.md for install/reference.
