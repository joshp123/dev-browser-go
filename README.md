# dev-browser-go

Token-light browser automation via Playwright-Go. **CLI-first** design for LLM agent workflows.

Uses ref-based interaction: get a compact accessibility snapshot, then click/fill by ref ID. Keeps context small. Single Go binary with embedded daemon.

## Acknowledgments

Inspired by [SawyerHood/dev-browser](https://github.com/SawyerHood/dev-browser). ARIA snapshot extraction is vendored from that project. Thanks to Sawyer Hood for the original work and ref-based model.

## Comparison

| Feature | SawyerHood/dev-browser | dev-browser-go |
|---------|------------------------|----------------|
| Language | TypeScript | Go |
| Runtime | Bun + browser extension | Playwright (Go) |
| Interface | Browser extension skill | CLI + daemon |
| Install | `.plugin` | Go binary / Nix |
| Best for | Desktop skill users | CLI/LLM agents, Nix users |
| Snapshot engine | ARIA (JS) | Same (vendored) |

## Why CLI (no MCP)

- Lower latency: direct subprocess, no JSON-RPC framing
- Easier debugging: run commands yourself, see stdout/stderr
- Simpler integration: any agent that can shell out works
- Persistent sessions: daemon keeps browser alive between calls

## Install

Playwright browsers are required. The Nix package wraps `PLAYWRIGHT_BROWSERS_PATH` to the packaged Chromium; dev shell includes the driver/browsers. Outside Nix, Playwright-Go will download on first run.

### Nix (flake)

```bash
nix run github:joshp123/dev-browser-go#dev-browser-go -- goto https://example.com
nix profile install github:joshp123/dev-browser-go#dev-browser-go
```

### Go build

```bash
go build ./cmd/dev-browser-go
./dev-browser-go goto https://example.com
./dev-browser-go snapshot --engine aria --format list
```

Env knobs (kept minimal): `DEV_BROWSER_PROFILE`, `HEADLESS`, `DEV_BROWSER_WINDOW_SIZE`, `DEV_BROWSER_ALLOW_UNSAFE_PATHS` (for artifacts). Daemon is the same binary invoked with `--daemon` internally; `HEADLESS=1` recommended for CI/agents.

## CLI Usage

```bash
dev-browser-go goto https://example.com
dev-browser-go snapshot                    # get refs
dev-browser-go click-ref e3                # click ref e3
dev-browser-go fill-ref e5 "search query"  # fill input
dev-browser-go screenshot
dev-browser-go press Enter
```

The daemon starts automatically on first command and keeps the browser session alive. Screenshot crops clamp to 2000x2000; models downscale larger captures and quality will be poor.

## Integration with AI Agents

Add to your project's agent docs:

```markdown
## Browser Automation

Use `dev-browser-go` CLI for browser tasks. Keeps context small via ref-based interaction.

Workflow:
1. `dev-browser-go goto <url>` - navigate
2. `dev-browser-go snapshot` - get interactive elements as refs (e1, e2, etc.)
3. `dev-browser-go click-ref <ref>` or `dev-browser-go fill-ref <ref> "text"` - interact
4. `dev-browser-go screenshot` - capture state if needed

Example:
```bash
dev-browser-go goto https://github.com/login
dev-browser-go snapshot
# Output: e1: textbox "Username" | e2: textbox "Password" | e3: button "Sign in"
dev-browser-go fill-ref e1 "myuser"
dev-browser-go fill-ref e2 "mypass"
dev-browser-go click-ref e3
```
```

## Integration with shell-driven agents

Any agent with shell access can call the CLI. Example prompt:

```
Use dev-browser-go to navigate to example.com and find all links on the page.

Available commands:
- dev-browser-go goto <url>
- dev-browser-go snapshot [--interactive-only / --no-interactive-only]
- dev-browser-go click-ref <ref>
- dev-browser-go fill-ref <ref> "text"
- dev-browser-go screenshot
- dev-browser-go press <key>
```

## Tools

- `goto <url>` - navigate
- `snapshot` - accessibility tree with refs
- `click-ref <ref>` - click element
- `fill-ref <ref> "text"` - fill input
- `press <key>` - keyboard input
- `screenshot` - save screenshot
- `save-html` - save page HTML
- `list-pages` - show open pages
- `status` / `start` / `stop` - daemon management

## Versioning & Releases

- Simple SemVer tags (`v0.y.z` for fast moves; bump to `v1.0.0` once stable).
- GitHub Release on each tag with the single Go binary (`dev-browser-go`) and checksums.
- Nix flake outputs follow the tag; no extra artifacts.

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE).

Vendored code from SawyerHood/dev-browser is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
