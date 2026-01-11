# dev-browser-mcp

Token-light browser automation via Playwright. **CLI-first** design for LLM agent workflows.

Uses ref-based interaction: get a compact accessibility snapshot, then click/fill by ref ID. Keeps context small.

## Acknowledgments

This project is a Python/CLI rewrite inspired by [SawyerHood/dev-browser](https://github.com/SawyerHood/dev-browser). The ARIA snapshot extraction logic is vendored from that project. Thanks to Sawyer Hood for the original work and the ref-based interaction model.

**Consider using Sawyer's original** if you want:
- Native Claude Skill integration (install via `.claude-plugin`)
- TypeScript/Bun ecosystem
- Tighter Claude Desktop integration

This repo is for CLI-first workflows, Nix packaging, or if you prefer Python/Playwright.

## Comparison

| Feature | SawyerHood/dev-browser | dev-browser-mcp |
|---------|------------------------|-----------------|
| Language | TypeScript | Python |
| Runtime | Bun + browser extension | Playwright (Python) |
| Interface | Claude Skill plugin | CLI + daemon (+ MCP) |
| Install | `.claude-plugin` | pip/Nix |
| Best for | Claude Desktop users | CLI agents, Codex, Nix users |
| Snapshot engine | ARIA (JS) | Same (vendored) |

Both use the same ref-based interaction model. Pick based on your environment.

## Why CLI over MCP?

MCP adds overhead: extra process, stdio piping, JSON-RPC framing, connection management. For browser automation, that's a lot of indirection when you can just call a CLI.

The CLI approach:
- **Lower latency** - direct subprocess, no protocol overhead
- **Easier debugging** - run commands yourself, see exactly what happens
- **Simpler integration** - any agent that can shell out works
- **Persistent sessions** - daemon keeps browser alive between calls

The MCP server exists if you need it, but the CLI + daemon is the recommended path.

## Install

Requires Python 3.11+ and Playwright browsers.

### pip (local checkout)

```bash
python -m venv .venv
source .venv/bin/activate
pip install .
python -m playwright install chromium

dev-browser goto https://example.com
dev-browser snapshot
dev-browser click-ref e3
```

### Nix (flake)

No overlays required. The flake exposes the CLI, daemon, and MCP server:

```bash
nix run github:joshp123/dev-browser-mcp#dev-browser -- goto https://example.com
nix run github:joshp123/dev-browser-mcp#dev-browser -- snapshot
```

Install to your profile:

```bash
nix profile install github:joshp123/dev-browser-mcp#dev-browser
```

## CLI Usage

```bash
dev-browser goto https://example.com
dev-browser snapshot                    # get refs
dev-browser click-ref e3                # click ref e3
dev-browser fill-ref e5 "search query"  # fill input
dev-browser screenshot
dev-browser press Enter
```

The daemon starts automatically on first command and keeps the browser session alive.

## Integration with Claude Code

Add to your project's `CLAUDE.md`:

```markdown
## Browser Automation

Use `dev-browser` CLI for browser tasks. Keeps context small via ref-based interaction.

Workflow:
1. `dev-browser goto <url>` - navigate
2. `dev-browser snapshot` - get interactive elements as refs (e1, e2, etc.)
3. `dev-browser click-ref <ref>` or `dev-browser fill-ref <ref> "text"` - interact
4. `dev-browser screenshot` - capture state if needed

Example:
\`\`\`bash
dev-browser goto https://github.com/login
dev-browser snapshot
# Output: e1: textbox "Username" | e2: textbox "Password" | e3: button "Sign in"
dev-browser fill-ref e1 "myuser"
dev-browser fill-ref e2 "mypass"
dev-browser click-ref e3
\`\`\`
```

## Integration with Codex

Codex can use the CLI directly via its shell access. Example prompt:

```
Use dev-browser to navigate to example.com and find all links on the page.

Available commands:
- dev-browser goto <url>
- dev-browser snapshot [--interactive-only / --no-interactive-only]
- dev-browser click-ref <ref>
- dev-browser fill-ref <ref> "text"
- dev-browser screenshot
- dev-browser press <key>
```

## Tools

CLI commands (recommended):
- `goto <url>` - navigate
- `snapshot` - accessibility tree with refs
- `click-ref <ref>` - click element
- `fill-ref <ref> "text"` - fill input
- `press <key>` - keyboard input
- `screenshot` - save screenshot
- `save-html` - save page HTML
- `list-pages` - show open pages
- `status` / `start` / `stop` - daemon management

MCP tools (if you must):
- `page` / `list_pages` / `close_page`
- `goto` / `snapshot` / `click_ref` / `fill_ref` / `press`
- `screenshot` / `save_html`
- `actions` - batch calls

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE).

Vendored code from SawyerHood/dev-browser is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
