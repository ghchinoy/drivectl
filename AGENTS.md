# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Build, Test & Linting Gates

When working with `drivectl`, run the following quality gates before committing:

* **Build:** `go build ./...` (verify both `CGO_ENABLED=1` and `CGO_ENABLED=0`)
* **Test:** `go test -race ./...`
* **Dependencies:** `go mod tidy`
* **Vulnerability Check:** `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
* **Linting & Code Health:** `golangci-lint run -E gocyclo -E cyclop -E gocognit ./...` (configured via `.golangci.yml` v2)
* **Documentation Site (`site/`):** `cd site && ASTRO_TELEMETRY_DISABLED=1 npm run check && ASTRO_TELEMETRY_DISABLED=1 npm run build && npm run check:docs`

## Architecture & Design

**CLI Command Structure:** When adding features to `drivectl`, refer to `docs/DESIGN_PHILOSOPHY.md`.
* **The `call` command** must remain uncomplicated and is strictly for standard REST/JSON interactions.
* **Native subcommands** (like `upload`) should be created for operations involving complex protocols, binary data streaming, or significant user-experience optimizations.

## Discovery & MCP Schema Invariants (`pkg/discovery` & `mcp`)

* **Typed Parameter Defaults:** `mcp.AddTool` (`github.com/modelcontextprotocol/go-sdk/mcp`) resolves input schemas with `ValidateDefaults: true`. In `pkg/discovery/schema.go`, always encode Discovery `MethodParameter.Default` strings as raw JSON booleans or numbers when `Type` is `boolean`, `integer`, or `number`.
* **Discovery `"any"` Types & Recursive `$ref`s:** Normalize Discovery `"type": "any"` to `""` in `jsonschema.Schema`, and maintain a per-branch `visited` map when expanding `doc.Schemas` `$ref`s so recursive schemas (such as `docs.v1` `Tab.childTabs`) neither stack-overflow nor share pointer nodes across the schema tree.
* **Query Parameter Routing on Write Methods:** `pkg/discovery.BuildRequest` extracts `location == "query"` parameters (defined in either `method.Parameters` or `doc.Parameters`) onto the URL query string before serializing the JSON request body, even when `method.Request != nil`.
* **In-Memory Test Doubles:** Use `http.RoundTripper` fakes and `mcp.NewInMemoryTransports()` in `*_test.go` rather than `httptest.NewServer` so `go test -race ./...` runs deterministically inside restricted sandboxes.

## Documentation Site & Release Invariants (`site/` & `.goreleaser.yaml`)

* **GitHub Pages Base Path (`base: '/drivectl'`):** The documentation site is served from GitHub Project Pages at `https://ghchinoy.github.io/drivectl/`. Keep `base: '/drivectl'` in `site/astro.config.mjs`, prefix internal Markdown links in `docs/site/` with `/drivectl/...`, and prefix custom Astro component asset paths with `import.meta.env.BASE_URL`. Run `npm run check:docs` in `site/` to verify no unprefixed `/_astro/` or root-relative links slip in.
* **Starlight Content Directory (`docs/site/`):** Public documentation pages live in `docs/site/` (loaded via `docsLoader({ base: '../docs/site' })`), keeping repository-internal design docs in `docs/` out of the published site tree.
* **GoReleaser v2 Config (`.goreleaser.yaml`):** `.github/workflows/release.yaml` runs `goreleaser/goreleaser-action@v6` (`version: latest`) on `v*` tags; use plural `formats: [tar.gz]` / `format_overrides.formats: [zip]` in `.goreleaser.yaml`.

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**
- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var

<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Version-controlled: Built on Dolt with cell-level merge
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update <id> --claim --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task atomically**: `bd update <id> --claim`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Auto-Sync

bd automatically syncs with git:

- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- END BEADS INTEGRATION -->
