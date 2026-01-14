---
name: dev-browser-go
description: Browser automation with persistent sessions via CLI. Use when users ask to navigate websites, fill forms, take screenshots, test web apps, or automate browser workflows. Trigger phrases include "go to [url]", "click on", "fill out the form", "take a screenshot", "test the website", or any browser interaction request.
---

# Dev Browser Skill (CLI)

Browser automation that maintains page state via a persistent daemon. Uses ref-based interaction for token efficiency.

## When to Use

- Testing web apps during development
- Filling forms, clicking buttons
- Taking screenshots
- Scraping structured data
- Any browser automation task

## Quick Start

The daemon starts automatically on first command. Just run:

```bash
dev-browser-go goto https://example.com
dev-browser-go snapshot
dev-browser-go click-ref e3
```

Run `dev-browser-go --help` for full CLI reference.

## Core Workflow

1. **Navigate** to a URL
2. **Snapshot** to get interactive elements as refs (e1, e2, etc.)
3. **Interact** using refs (click, fill, press)
4. **Screenshot** if visual verification needed

### Example: Login Flow

```bash
# Navigate to login page
dev-browser-go goto https://github.com/login

# Get interactive elements
dev-browser-go snapshot
# Output:
# e1: textbox "Username or email address"
# e2: textbox "Password"
# e3: button "Sign in"

# Fill and submit
dev-browser-go fill-ref e1 "myusername"
dev-browser-go fill-ref e2 "mypassword"
dev-browser-go click-ref e3

# Verify result
dev-browser-go snapshot
```

## Commands Reference

### Navigation & Pages
```bash
dev-browser-go goto <url>                    # Navigate to URL
dev-browser-go goto <url> --page checkout    # Use named page
dev-browser-go list-pages                    # List open pages
dev-browser-go close-page <name>             # Close named page
```

### Inspection
```bash
dev-browser-go snapshot                      # Get refs for interactive elements
dev-browser-go snapshot --no-interactive-only  # Include all elements
dev-browser-go snapshot --engine aria        # Use ARIA engine (better for complex UIs)
dev-browser-go screenshot                    # Full-page screenshot
dev-browser-go screenshot --annotate-refs    # Overlay ref labels on screenshot
dev-browser-go screenshot --selector ".panel" --padding-px 10  # Element crop + padding
dev-browser-go screenshot --crop 0,0,800,600 # Crop region (max 2000x2000)
dev-browser-go bounds ".panel" --nth 1      # Element bounds (CSS or ARIA)
dev-browser-go save-html --path page.html    # Save page HTML
```

### Interaction
```bash
dev-browser-go click-ref <ref>               # Click element by ref
dev-browser-go fill-ref <ref> "text"         # Fill input by ref
dev-browser-go press Enter                   # Press key
dev-browser-go press Tab                     # Navigate with Tab
dev-browser-go press Escape                  # Close modals
```

### Waiting
```bash
dev-browser-go wait                          # Wait for page load
dev-browser-go wait --state networkidle      # Wait for network idle
dev-browser-go wait --timeout-ms 5000        # Custom timeout
```

### Batch Actions
```bash
# Execute multiple actions in one call
echo '[{"tool":"click_ref","args":{"ref":"e1"}},{"tool":"press","args":{"key":"Enter"}}]' | dev-browser-go actions
```

### Daemon Management
```bash
dev-browser-go status                        # Check daemon status
dev-browser-go stop                          # Stop daemon (closes browser)
dev-browser-go start --headless              # Start in headless mode
```

## Interpreting Snapshots

Snapshot output looks like:
```
e1: textbox "Search" [placeholder: "Type to search..."]
e2: button "Submit" [disabled]
e3: link "Home" [/url: /home]
e4: checkbox "Remember me" [checked]
e5: combobox "Country" [expanded]
```

- `eN` - Element reference for interaction
- `[disabled]`, `[checked]`, `[expanded]` - Element states
- `[placeholder: ...]`, `[/url: ...]` - Element properties

## Tips

### Small Steps
Run one action at a time, check output, then proceed. Don't chain multiple actions blindly.

### Use Named Pages
For multi-page workflows, use `--page` to keep contexts separate:
```bash
dev-browser-go goto https://app.com/settings --page settings
dev-browser-go goto https://app.com/profile --page profile
dev-browser-go snapshot --page settings  # Back to settings
```

### Headless Mode
Default is headless. To disable:
```bash
dev-browser-go start --headed
# or
HEADLESS=0 dev-browser-go goto https://example.com
```

### Viewport Size
Default is ultrawide 7680x2160. Adjust with flags:
```bash
dev-browser-go goto https://example.com --window-scale 0.75  # 5760x1620 (0.75x default)
# or
dev-browser-go goto https://example.com --window-size 3840x1080
```
Env default:
```bash
DEV_BROWSER_WINDOW_SIZE=412x915 dev-browser-go goto https://example.com
```

### Device Emulation
Use Playwright device profiles for UA + DPR + touch + viewport/screen:
```bash
dev-browser-go devices
dev-browser-go --device "Galaxy S20 Ultra" goto https://example.com
```
Do not combine `--device` with `--window-size` or `--window-scale`.
Note: device/viewport flags apply when the daemon starts. Stop the daemon to switch.

### Element Screenshots
For component-level captures, use CSS selectors from your codebase:
```bash
dev-browser-go bounds ".vault-panel"                        # Verify selector + check dimensions
dev-browser-go screenshot --selector ".vault-panel" --padding-px 8
```

Tips:
- Use class selectors from your source code — you know your component names
- Check `bounds` first to verify dimensions match expected area
- Small padding (8-16px) for tight crops

### Debugging
If something isn't working:
```bash
dev-browser-go screenshot                    # See current state
dev-browser-go snapshot --no-interactive-only  # See all elements
```

## See Also

- `dev-browser-go --help` for full CLI reference
- [README.md](README.md) for installation and architecture
