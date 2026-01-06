# OpenCode integration design

This document explains design decisions for OpenCode integration in beads.

## Integration approach

**Recommended: CLI + Hooks** - Beads uses a simple, universal approach to OpenCode integration:
- `bd prime` command for context injection (~1-2k tokens)
- Hooks (session_start/pre_compact) for automatic context refresh
- Direct CLI commands with `--json` flags

## Why CLI + hooks?

**Context efficiency matters**, even with large context windows:

1. **Compute cost scales with tokens** - Every token in your context consumes compute on every inference, regardless of whether it's used
2. **Latency increases with context** - Larger prompts take longer to process
3. **Energy consumption** - Each token has environmental impact; lean prompts are more sustainable
4. **Attention quality** - Models attend better to smaller, focused contexts

**The math:**
- MCP tool schemas can add 10-50k tokens to context (depending on number of tools)
- `bd prime` adds ~1-2k tokens of workflow context
- That's 10-50x less context overhead

## Implementation

OpenCode uses `experimental.hook` in `opencode.json` configuration:

| Hook | Trigger | Action |
| :--- | :------ | :----- |
| `session_start` | Session begins | Runs `bd prime` |
| `pre_compact` | Before context compaction | Runs `bd prime` |

**Configuration locations:**
- Global: `~/.config/opencode/opencode.json`
- Project: `.opencode/opencode.json`

## Installation

```bash
# Install OpenCode hooks globally
bd setup opencode

# Install for this project only
bd setup opencode --project

# Use stealth mode (flush only, no git operations)
bd setup opencode --stealth

# Check installation status
bd setup opencode --check

# Remove hooks
bd setup opencode --remove
```

**What it installs:**
- session_start hook: Runs `bd prime` when OpenCode starts a session
- pre_compact hook: Runs `bd prime` before context compaction to preserve workflow instructions

## Related files

- [cmd/bd/prime.go](../cmd/bd/prime.go) - Context generation
- [cmd/bd/setup/opencode.go](../cmd/bd/setup/opencode.go) - Hook installation
- [cmd/bd/doctor/opencode.go](../cmd/bd/doctor/opencode.go) - Integration verification
