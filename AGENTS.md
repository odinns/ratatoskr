# AGENTS.md

## Project

Ratatoskr is a conservative local disk cleaner CLI.

It scans filesystem trees, identifies generated waste, explains every finding, and only cleans when the user explicitly asks.

The source-of-truth product spec lives in [docs/hardened-megaspec.md](docs/hardened-megaspec.md). Read it before designing commands, rules, deletion logic, or output.

## Core Direction

Build a small CLI creature with good judgment.

Ratatoskr should feel fast, sharp, slightly feral, and trustworthy. The mythology is seasoning. The safety model is the meal.

The MVP starts scan-only:

- `ratatoskr scan`
- `ratatoskr rules`
- `ratatoskr report --format json`

Cleaning comes only after scanning is boringly reliable:

- `ratatoskr clean --safe`

Do not build `clean --all`. Not now. Probably not later without a separate design.

## Non-Negotiables

- `scan`, `report`, and `rules` are read-only.
- Deletion must be explicit.
- Default cleaning is safe-only and trash-first.
- Safety belongs to rules, not categories.
- Every candidate needs path, size, category, risk, rule, reason, and default-cleanability.
- Unknown large files are report-only.
- Dangerous candidates are not cleaned in the MVP.
- Personal files are never deleted by inference.
- Docker volumes are dangerous by default.
- Mythic display groups never replace plain category and risk.
- Re-stat immediately before deletion.
- Skip changed paths. Skip beats regret.

## Implementation Bias

Prefer Go or Rust for a real standalone tool.

Use Laravel Zero only for a fast personal prototype. Avoid Node unless prototype speed matters more than the product shape. A disk cleaner that hauls a dependency asteroid behind it is funny, but not helpful.

Keep the implementation plain:

- narrow rules
- explicit conditions
- concrete resolved paths
- simple output models
- visible errors
- tests around safety boundaries

Do not add abstraction until duplication hurts or the rule model needs it.

## Rule Model

Every built-in rule must define:

- name
- source
- category
- risk
- path patterns
- `applies_when` conditions
- exclusions
- reason
- consequence
- `cleanable_by_default`

Rules must be inspectable through `ratatoskr rules`.

Never mark something safe because the folder name sounds disposable. A cache is not safe because it is called cache. A log is not safe because it ends in `.log`.

Safe means a narrow rule knows who owns the path, why it is generated, what deletion costs, and what must be excluded.

## Path Safety

Deletion must operate on concrete resolved paths only.

Before deleting anything:

- expand the path
- resolve the real path
- confirm it exists
- confirm type, size, modified time, and resolved path still match the candidate
- confirm it is inside an allowed scan root
- confirm it is not protected
- confirm it is not a scan root, project root, repository root, home directory, or `/`
- confirm it is not a symlink escape

Protected paths include:

- `/`
- `~`
- `~/Desktop`
- `~/Documents`
- `~/Downloads`
- `~/Pictures`
- `~/Movies`
- `~/.ssh`
- `~/.gnupg`
- `.git`
- project roots
- repository roots
- database files
- `.env` files

No deletion from unresolved globs.

No following symlinks for deletion in the MVP.

## MVP Rule Targets

Start with narrow, explainable rules for:

- Laravel logs and generated framework cache, with strict storage exclusions
- Node build outputs such as `.next`, `dist`, `build`, `.turbo`, `.vite`, and `coverage`
- Composer cache when explicit user-cache scanning is enabled
- cautious report-only `vendor`
- cautious report-only `node_modules`
- common package caches, only when paths are explicit
- unknown large files as dangerous/report-only

Never clean by default:

- `storage/app`
- `storage/framework/sessions`
- `database/*.sqlite`
- `.env`
- `public/uploads`
- `public/storage`
- `vendor`
- `node_modules`
- Docker volumes
- downloads, documents, photos, videos, archives, database dumps, or unknown large files

## Command Behavior

`ratatoskr scan` should default to the current project directory. It must not scan the whole home directory or system root by default.

Known user cache locations require explicit enablement or config.

`ratatoskr report` should render existing scan data as text, JSON, or markdown. Reports must include the privacy note:

```text
Reports contain local file paths. Treat them as private.
```

`ratatoskr clean --safe` should run a fresh scan, select only safe candidates, print the exact deletion list, ask for confirmation unless `--yes` is passed, then move candidates to trash. If trash fails, abort unless permanent deletion was explicitly requested with `--delete`.

## Output Voice

Calm. Useful. Direct. Lightly mythic.

Good:

```text
Ratatoskr found 12.6 GB of reclaimable disk waste.
3.4 GB is safe to clean now.
7.8 GB requires explicit selection.
1.4 GB is report-only.
```

Bad:

```text
By Odin's beard, your disk is cursed!
Summoning sacred cleanup ritual...
AI has determined these files are useless.
```

Use mythology as seasoning, not soup.

## Testing Expectations

Safety tests matter more than happy-path demos.

Cover:

- read-only commands do not mutate
- safe/cautious/dangerous filtering
- protected path rejection
- repo root and project root rejection
- symlink escape handling
- changed-path skip behavior
- unresolved glob rejection
- exact deletion list generation
- trash failure behavior
- JSON schema shape
- rule explanation output

Use fixtures for fake projects. Keep tests small enough that failures point at the actual broken contract.

## Documentation Voice

All written docs, comments, and explanatory text in this repo should use the `odinns-voice` skill.

Keep it plain. Kill pitch-deck fog. Explain the consequence, not the vibe.

## Current Success Criteria

Milestone 1 is done when Ratatoskr can:

- run in a project folder
- scan an explicit path such as `~/Code`
- find obvious generated waste
- avoid personal files as cleanable candidates
- explain every candidate
- output readable text and valid JSON
- list active rules
- mutate nothing

Until that is true, cleaning can wait.
