---
id: opencode
title: OpenCode
sidebar_position: 2
---

# OpenCode Integration

How to use beads with [OpenCode](https://opencode.ai).

## What you get

- **Automatic context injection** - `bd prime` runs on session start (~1-2k tokens)
- **Pre-compaction sync** - `bd prime` runs before context compaction to preserve workflow instructions
- **Direct CLI access** - Use `bd` commands naturally during sessions

## Requirements

- [beads CLI](/getting-started/installation) installed
- [OpenCode](https://opencode.ai) installed

## Quick start

```bash
bd setup opencode
```

Restart OpenCode for changes to take effect.

## Installation options

### Global vs project

```bash
# Global installation (default)
# Config: ~/.config/opencode/opencode.json
bd setup opencode

# Project-level installation
# Config: .opencode/opencode.json
bd setup opencode --project
```

Use project-level when you want beads hooks only for specific projects.

### Stealth mode

```bash
bd setup opencode --stealth
```

Stealth mode runs `bd prime --stealth` which flushes output only, skipping git operations. Use when you want minimal side effects.

### Check status

```bash
bd setup opencode --check
```

Shows whether hooks are installed and their location.

### Remove hooks

```bash
# Remove global hooks
bd setup opencode --remove

# Remove project hooks
bd setup opencode --remove --project
```

## How it works

The setup command adds hooks to `opencode.json`:

```json
{
  "experimental": {
    "hook": {
      "session_start": [
        { "command": ["bd", "prime"] }
      ],
      "pre_compact": [
        { "command": ["bd", "prime"] }
      ]
    }
  }
}
```

| Hook | Trigger | Action |
| :--- | :------ | :----- |
| `session_start` | Session begins | Injects workflow context |
| `pre_compact` | Before context compaction | Preserves workflow instructions |

**Configuration locations:**

| Type | Path |
| :--- | :--- |
| Global | `~/.config/opencode/opencode.json` |
| Project | `.opencode/opencode.json` |

## Essential commands for agents

### Creating issues

```bash
# Always include description for context
bd create "Fix authentication bug" \
  --description="Login fails with special characters in password" \
  -t bug -p 1 --json

# Link discovered issues
bd create "Found SQL injection" \
  --description="User input not sanitized in query builder" \
  --deps discovered-from:bd-42 --json
```

### Working on issues

```bash
# Find ready work
bd ready --json

# Start work
bd update bd-42 --status in_progress --json

# Complete work
bd close bd-42 --reason "Fixed in commit abc123" --json
```

### Syncing

```bash
# ALWAYS run at session end
bd sync
```

## Best practices

### Always use `--json`

```bash
bd list --json          # Parse programmatically
bd create "Task" --json # Get issue ID from output
bd show bd-42 --json    # Structured data
```

### Always include descriptions

```bash
# Good
bd create "Fix auth bug" \
  --description="Login fails when password contains quotes" \
  -t bug -p 1 --json

# Bad - no context for future work
bd create "Fix auth bug" -t bug -p 1 --json
```

### Sync before session end

```bash
# ALWAYS run before ending
bd sync
```

## Troubleshooting

### Context not injected

```bash
# Check hook setup
bd setup opencode --check

# Manually prime
bd prime
```

### Changes not syncing

```bash
# Force sync
bd sync

# Check daemon
bd info
bd daemons health
```

### Database not found

```bash
# Initialize beads
bd init --quiet
```

### Run diagnostics

```bash
bd doctor
```

The doctor command checks OpenCode integration status along with other system health.

## See Also

- [Claude Code](/integrations/claude-code) - Claude Code integration
- [MCP Server](/integrations/mcp-server) - For MCP-only environments
- [IDE Setup](/getting-started/ide-setup) - Other editors
