# Deploying dev-browser-go

Deployment = GitHub release. Pre-1.0: breakage OK; no compatibility promises.

## Prereqs
- `go` available
- Optional: `nix develop` (Playwright browsers)

## Local checks
```bash
go test ./...
go build ./cmd/dev-browser-go
```

## Deploy steps
1. Bump version in `cmd/dev-browser-go/main.go`.
2. Tag release: `git tag -a v0.y.z -m "v0.y.z"`.
3. Push: `git push origin main --tags`.
4. Create GitHub Release with the single `dev-browser-go` binary and checksums.
5. Update `README.md` or `CHANGELOG.md` if needed.

## Notes
- Single binary only.
- Flake outputs Go binary only.
