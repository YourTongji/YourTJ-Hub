# Session Startup Checklist

> Quick reference for starting a new session on YourTJ-Hub
> Read this at the start of every session

## Essential Files to Read

1. ✅ **AGENTS.md** - Repository hard constraints and operating guide
2. ✅ **docs/README.md** - Documentation index and fact sources
3. ✅ **docs/development/README.md** - Development workflow
4. ✅ **.agents/local/REQUIREMENTS.md** - Local AI agent requirements

## Before Making Any Changes

- [ ] Read AGENTS.md and relevant docs
- [ ] Check current branch (should be `feat/*`, `fix/*`, or `docs/*`)
- [ ] Verify branch is created from `origin/dev`
- [ ] Identify change impact scope
- [ ] Understand layer boundaries (service/models/http)

## Key Constraints to Remember

- 🔒 Never commit `config.toml`
- 🔒 User IDs must be uint64 (numeric)
- 🔒 PostgreSQL only (no MySQL types)
- 🔒 Single binary deployment
- 🔒 Changes require doc updates

## Branch Workflow

```bash
# Check current branch
git branch

# Create new branch from dev
git checkout dev
git pull origin dev
git checkout -b feat/your-feature

# After changes
git add <relevant-files>
git commit -m "feat: your message"
git push origin feat/your-feature
```

## Testing Checklist

- [ ] Backend: `go vet ./...` and `go test ./...`
- [ ] Web: `pnpm typecheck` and `pnpm test`
- [ ] Build: `make build` succeeds
- [ ] Relevant documentation updated

## Status Words

- `Current` = Done and verified
- `Partial` = In progress (state what's missing)
- `Planned` = Not started
- `Decision needed` = Blocked on decision

## When in Doubt

1. Read the docs first
2. Check existing code patterns
3. Ask for clarification
4. Use the `$yourtj-development` skill