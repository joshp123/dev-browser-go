# dev-browser-go

Token-light browser automation via Playwright-Go. **CLI-first** design for LLM agent workflows.

Uses ref-based interaction: get a compact accessibility snapshot, then click/fill by ref ID. Keeps context small. Single Go binary with embedded daemon.

## Acknowledgments

Inspired by [SawyerHood/dev-browser](https://github.com/SawyerHood/dev-browser). ARIA snapshot extraction is vendored from that project. Thanks to Sawyer Hood for the original work and ref-based model.

Thanks to Daniel van Dorp (@djvdorp) for early contributions (pip packaging, console logs) and legacy MCP cleanup work.

## Comparison

| Feature | SawyerHood/dev-browser | dev-browser-go |
|---------|------------------------|----------------|
| Language | TypeScript | Go |
| Runtime | Bun + browser extension | Playwright-Go |
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
./dev-browser-go snapshot
```

## CLI Usage

```bash
dev-browser-go --help              # Full usage
dev-browser-go --version           # Version

dev-browser-go goto https://example.com
dev-browser-go snapshot            # Get refs (e1, e2, ...)
dev-browser-go click-ref e3        # Click ref
dev-browser-go fill-ref e5 "text"  # Fill input
dev-browser-go screenshot          # Capture
dev-browser-go press Enter         # Keyboard
```

The daemon starts automatically on first command and keeps the browser session alive.

### Global Flags

```
--profile <name>    Browser profile (default: "default", env DEV_BROWSER_PROFILE)
--headless          Run headless (default)
--headed            Disable headless
--window-size WxH   Viewport size (default 7680x2160 ultrawide)
--window-scale S    Viewport scale preset (1, 0.75, 0.5)
--device <name>     Device profile name (Playwright)
--output <format>   Output format: summary|json|path (default: summary)
--out <path>        Write output to file (with --output=path)
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DEV_BROWSER_PROFILE` | Browser profile name |
| `HEADLESS` | Override headless default (1/true/yes to enable, 0/false to disable) |
| `DEV_BROWSER_WINDOW_SIZE` | Default viewport size (WxH) |
| `DEV_BROWSER_ALLOW_UNSAFE_PATHS` | Allow artifact writes outside cache dir |

### Viewport + Device Emulation

Viewport only (responsive CSS):
```bash
dev-browser-go --window-size 412x915 goto https://example.com
```

Device profile (UA + DPR + touch + viewport/screen):
```bash
dev-browser-go --device "Galaxy S20 Ultra" goto https://example.com
```
Do not combine `--device` with `--window-size` or `--window-scale`.

List available profiles:
```bash
dev-browser-go devices
```

Note: device profiles use Playwright names; device/viewport flags apply when the daemon starts. Stop the daemon to switch.

## Commands

| Command | Description |
|---------|-------------|
| `goto <url>` | Navigate to URL |
| `snapshot` | Accessibility tree with refs |
| `click-ref <ref>` | Click element by ref |
| `fill-ref <ref> "text"` | Fill input by ref |
| `press <key>` | Keyboard input |
| `screenshot` | Save screenshot (full-page or element crop with padding; crops clamp to 2000x2000) |
| `bounds` | Get element bounding box (selector/ARIA) |
| `console` | Read page console logs (default levels: info,warning,error) |
| `save-html` | Save page HTML |
| `devices` | List device profile names |
| `wait` | Wait for page state |
| `list-pages` | Show open pages |
| `close-page <name>` | Close named page |
| `call <tool>` | Generic tool call with JSON args |
| `actions` | Batch tool calls from JSON |
| `status` | Daemon status |
| `start` | Start daemon |
| `stop` | Stop daemon |

Run `dev-browser-go <command> --help` for command-specific options.

## Integration with AI Agents

Add to your project's agent docs (or use [SKILL.md](SKILL.md) directly):

```markdown
## Browser Automation

Use `dev-browser-go` CLI for browser tasks. Keeps context small via ref-based interaction.

Workflow:
1. `dev-browser-go goto <url>` - navigate
2. `dev-browser-go snapshot` - get interactive elements as refs (e1, e2, etc.)
3. `dev-browser-go click-ref <ref>` or `dev-browser-go fill-ref <ref> "text"` - interact
4. `dev-browser-go screenshot` - capture state if needed
```

Element-level capture:
```bash
dev-browser-go bounds ".vault-panel" --nth 1
dev-browser-go screenshot --selector ".vault-panel" --padding-px 10
```

For detailed workflow examples, see [SKILL.md](SKILL.md).

## Integration with Codex

Codex can use the CLI directly via its shell access. Example prompt:

```
Use dev-browser-go to navigate to example.com and find all links on the page.

Available commands:
- dev-browser-go goto <url>
- dev-browser-go snapshot [--no-interactive-only] [--no-include-headings]
- dev-browser-go click-ref <ref>
- dev-browser-go fill-ref <ref> "text"
- dev-browser-go screenshot
- dev-browser-go press <key>
- dev-browser-go console [--since <id>] [--limit <n>] [--level <lvl> ...]
```

## Tools

- `goto <url>` - navigate
- `snapshot` - accessibility tree with refs
- `click-ref <ref>` - click element
- `fill-ref <ref> "text"` - fill input
- `press <key>` - keyboard input
- `screenshot` - save screenshot
- `bounds` - get element bounds (selector/ARIA)
- `console` - read page console logs (default levels: info,warning,error; repeatable `--level`)
- `save-html` - save page HTML
- `wait` - wait for page state
- `list-pages` - show open pages
- `close-page <name>` - close named page
- `call <tool>` - generic tool call with JSON args
- `actions` - batch tool calls from JSON
- `status` / `start` / `stop` - daemon management

## Versioning & Releases

- Simple SemVer tags (`v0.y.z` for fast moves; bump to `v1.0.0` once stable).
- GitHub Release on each tag with the single Go binary (`dev-browser-go`) and checksums.
- Nix flake outputs follow the tag; no extra artifacts.

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE).

Vendored code from SawyerHood/dev-browser is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
