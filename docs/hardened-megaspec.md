# Ratatoskr - Hardened Megaspec

## Purpose

Ratatoskr is a conservative local disk cleaner CLI with a Norse soul and a practical job:

Scan the filesystem tree, identify generated waste, explain every finding, and only clean when the user explicitly asks.

It should feel fast, sharp, slightly feral, and deeply trustworthy.

Ratatoskr is not:

- a "delete everything" tool
- a bloated Mac cleaner app
- a fake system optimizer
- an AI cleanup oracle
- a SaaS dashboard wearing a fake mustache
- rm -rf with a squirrel logo

Ratatoskr is a small CLI creature with good judgment.

## Mythological Soul

Ratatoskr is the squirrel that runs up and down Yggdrasil, carrying messages between the eagle in the crown and Nidhogg at the roots.

For this tool:

- the filesystem is Yggdrasil
- generated waste is rot in the roots and branches
- Ratatoskr runs the tree
- Ratatoskr reports what is gnawing at the disk
- cleaning is pruning, not burning

Core feeling:

A small, fast filesystem scout that finds rot, reports honestly, and only gnaws when told.

## Product Shape

Start as a CLI tool.

Primary MVP commands:

- `ratatoskr scan`
- `ratatoskr report`
- `ratatoskr rules`

Cleaning comes after scan is trustworthy:

- `ratatoskr clean --safe`

Later commands:

- `ratatoskr clean --include-risk=cautious`
- `ratatoskr explain`
- `ratatoskr doctor`
- `ratatoskr restore`
- `ratatoskr watch`

Avoid for MVP:

- `ratatoskr clean --all`

There should be no broad "clean everything" command in the first version.

## Core Principles

## 1. Safe by default

Ratatoskr must never delete anything unless the user explicitly asks it to clean.

`ratatoskr scan` must be completely read-only.

`ratatoskr report` must be completely read-only.

`ratatoskr rules` must be completely read-only.

## 2. Explain before action

Every candidate must show:

- path
- size
- category
- risk level
- rule name
- reason it is considered removable
- whether it can be cleaned by default

No candidate should appear as magic.

## 3. Rule-level safety, not category vibes

A category is not automatically safe.

"Cache" does not mean safe.
"Log" does not mean safe.
"Build output" does not mean safe.

A candidate is safe only when a narrow rule knows:

- what owns the path
- why it is generated
- what happens if removed
- what must not be included
- what conditions must be true

Safety belongs to the rule, not the folder name.

## 4. Never infer-delete personal files

Ratatoskr must never delete user-created files by guessing.

Never clean by default:

- documents
- photos
- videos
- downloads
- desktop files
- database dumps
- archives
- unknown large files
- project roots
- home directory
- repository roots
- anything outside known generated-artifact rules

Unknown files may be reported, but never cleaned by default.

## 5. Conservative deletion

Prefer known generated targets:

- package caches
- build artifacts
- framework caches
- old logs
- temp files
- known generated folders

Avoid cleverness.

No "this looks old, delete it."
No "this is big, delete it."
No "AI thinks this is probably junk."

## 6. Concrete deletion list

Before cleaning, Ratatoskr must resolve candidates to concrete paths.

It must print the exact deletion list.

It must never delete from unresolved broad globs.

It must never delete scan roots, home directory, repository roots, project roots, or protected system paths.

## 7. Trash-first

Default cleaning should move files to trash where possible.

If trash fails, Ratatoskr should abort.

Permanent deletion should require an explicit flag:

- `ratatoskr clean --safe --delete`

## 8. Re-check before deletion

Between scan and clean, files may change.

Before deleting each candidate, Ratatoskr should re-stat the path.

If size, type, ownership, symlink target, or modified time changed unexpectedly, skip it and report it.

## 9. Inspectable rules

Rules must be visible.

The user must be able to see:

- which rules are active
- where they came from
- what they match
- what they exclude
- what risk they assign
- what reason they give

## 10. Competent terminal UX

Output should be clean, grouped, readable, and slightly atmospheric.

The soul is mythic.
The UX is competent.

No fake epic mode.
No horned helmet nonsense.
No "By Odin's beard".

## Risk Levels

## safe

Generated files that are normally safe to remove when matched by a narrow, verified rule.

Examples:

- known cache folders
- known build output
- known framework-generated logs
- known temporary files

Safe means:

- owned by a known tool/framework/package manager
- can be regenerated
- not likely to contain user-created data
- matched by a narrow rule
- not protected by exclusions

## cautious

Usually removable, but deletion may cost time, require reinstall/rebuild, or surprise the user.

Examples:

- `node_modules`
- `vendor`
- package manager stores
- Docker images
- Docker build cache
- large generated project artifacts

Cautious items are not cleaned by default.

They require explicit selection:

- `ratatoskr clean --include-risk=cautious`

## dangerous

Potential data loss or unclear ownership.

Examples:

- Docker volumes
- database files
- database dumps
- downloads
- photos
- documents
- archives
- unknown large files
- anything outside known generated rules
- anything matched only by size or age

Dangerous items are report-only by default.

No dangerous cleaning workflow in MVP.

## Categories

Plain categories:

- caches
- logs
- builds
- dependencies
- containers
- temp
- framework-runtime
- package-manager
- unknown-large-files
- report-only

Optional display groups:

- Roots: system-level waste
- Branches: project-level waste
- Nests: dependency and cache folders
- Bark: logs and surface debris
- Rot: stale generated artifacts

Important:

Mythic display groups must never replace category or risk.

Every finding must still show plain category and risk.

## Commands

## `ratatoskr scan`

Read-only.

Scans configured roots and reports cleanup candidates.

Default behavior:

- scan current directory when run inside a project
- scan configured roots if config exists
- scan known user cache locations only when explicitly enabled or configured
- do not scan entire home directory by default
- do not scan system root

Useful flags:

- `ratatoskr scan --path ~/Code`
- `ratatoskr scan --format text`
- `ratatoskr scan --format json`
- `ratatoskr scan --include-risk=cautious`
- `ratatoskr scan --docker`
- `ratatoskr scan --no-myth`

Output must include:

- total potential reclaimable size
- safe reclaimable size
- cautious report-only size
- dangerous/report-only size
- skipped paths
- unreadable paths
- applied rules
- candidates grouped by display group/category

Example output:

```text
Ratatoskr ran the tree.

Found 18.4 GB of potential disk waste.

Safe to clean:
  6.1 GB

Requires explicit selection:
  9.8 GB

Report-only:
  2.5 GB

Roots
  ~/Library/Caches/Homebrew
  Size: 4.2 GB
  Risk: safe
  Category: caches
  Rule: homebrew-cache
  Reason: Homebrew cache files can be regenerated or re-downloaded.

Branches
  ~/Code/site/.next
  Size: 1.4 GB
  Risk: safe
  Category: builds
  Rule: node-next-build
  Reason: Next.js build output is generated from project source.

Nests
  ~/Code/foo/node_modules
  Size: 2.8 GB
  Risk: cautious
  Category: dependencies
  Rule: node-modules
  Reason: Dependencies can usually be reinstalled, but deletion may cost time.

Rot
  ~/Downloads/old-backup.sql
  Size: 2.5 GB
  Risk: dangerous
  Category: unknown-large-files
  Rule: report-large-file
  Reason: Large user-owned file. Report-only. Never cleaned by default.

Skipped:
  ~/Code/private-app/storage/app
  Reason: possible user/application data
```

## `ratatoskr report`

Read-only.

Produces a report from a scan.

Formats:

- `ratatoskr report --format text`
- `ratatoskr report --format json`
- `ratatoskr report --format markdown`

Report should include:

- summary
- candidates
- rules
- skipped paths
- errors
- warnings
- privacy note

Privacy note:

Reports contain local file paths. Treat them as private.

## `ratatoskr rules`

Read-only.

Shows active rules and where they came from.

Output per rule:

- rule name
- source
- category
- risk
- paths/globs
- exclusions
- `applies_when` conditions
- reason
- cleaning behavior
- whether enabled

Example rule display:

```text
Rule: laravel-logs
Source: built-in
Risk: safe
Category: logs
Matches: storage/logs/*.log
Applies when: artisan exists and composer.json contains laravel/framework
Excludes: storage/app, storage/framework/sessions
Reason: Laravel log files are generated runtime artifacts.
Cleaned by default: yes
```

## `ratatoskr clean --safe`

First cleaning command.

Runs scan, selects only safe candidates, prints exact deletion list, asks for confirmation, and moves selected items to trash.

Default behavior:

- safe only
- trash-first
- confirmation required
- re-stat before deletion
- skip changed paths
- report failures
- never clean cautious or dangerous items

Useful flags:

- `ratatoskr clean --safe`
- `ratatoskr clean --safe --dry-run`
- `ratatoskr clean --safe --yes`
- `ratatoskr clean --safe --delete`
- `ratatoskr clean --safe --category logs`
- `ratatoskr clean --safe --path ~/Code`

Confirmation should show:

- number of paths
- total size
- trash/delete mode
- risk level
- exact candidates
- skipped candidates

Example prompt:

```text
Ratatoskr is ready to prune 6.1 GB from 14 safe candidates.

Mode: trash
Risk: safe only

Proceed? [y/N]
```

## `ratatoskr clean --include-risk=cautious`

Later, not MVP.

Allows cautious cleanup with explicit opt-in.

Must still exclude dangerous items.

Should require stronger confirmation than safe cleanup.

Example:

```sh
ratatoskr clean --include-risk=cautious
```

Prompt should make cost clear:

```text
This includes cautious items such as node_modules or vendor.
They are usually rebuildable, but deletion may cost time and bandwidth.

Proceed? [y/N]
```

## No `ratatoskr clean --all` in MVP

Do not implement `--all` for MVP.

Reason:

"All" is ambiguous and dangerous.
It encourages user trust in a broad action before the rule system has earned it.

Prefer explicit risk inclusion:

- `--safe`
- `--include-risk=cautious`
- later maybe `--target exact-path`

Dangerous cleaning should require a separate future design.

## Configuration

Ratatoskr should support config.

Possible file names:

- `ratatoskr.yml`
- `.ratatoskr.yml`

Initial config shape:

```yaml
scan:
  roots:
    - ~/Code
    - ~/Sites

clean:
  default_mode: trash
  default_risk: safe

rules:
  include:
    - laravel
    - node
    - composer
    - homebrew

exclude:
  paths:
    - ~/Code/important-archive
    - **/.git
    - **/storage/app
    - **/storage/framework/sessions
```

Important config rules:

- config cannot override protected paths
- custom rules must provide risk
- custom rules must provide reason
- custom rules cannot target home directory directly
- custom rules cannot target `/`
- custom rules cannot target repository root
- custom rules cannot use broad parent deletion without concrete resolution

## Built-In Rule Requirements

Every built-in rule must define:

- name
- category
- risk
- path patterns
- `applies_when` conditions
- exclusions
- reason
- expected consequence of deletion
- `cleanable_by_default` true/false

Example:

```yaml
name: laravel-logs
category: logs
risk: safe
paths:
  - storage/logs/*.log
applies_when:
  - artisan exists
  - composer.json contains laravel/framework
exclusions:
  - storage/app
  - storage/framework/sessions
reason: Laravel log files are generated runtime artifacts.
consequence: Historical local logs are removed.
cleanable_by_default: true
```

## MVP Detection Rules

## Laravel

Detect Laravel project by:

- `artisan` file exists
- `composer.json` contains `laravel/framework`

Safe candidates:

- `storage/logs/*.log`
- `bootstrap/cache/*.php`, with explanation that framework cache may need rebuild
- `storage/framework/views/*`, if clearly compiled views
- `storage/framework/cache/*`, only with narrow exclusions

Never touch:

- `storage/app`
- `storage/framework/sessions`
- `database/*.sqlite`
- `.env`
- `public/uploads`
- `public/storage`
- user-defined storage paths

Notes:

Laravel storage is dangerous unless narrowly understood.

Do not treat all of `storage` as cleanable.

## Composer

Safe candidates:

- `~/.composer/cache`, when explicitly scanning user cache locations
- Composer cache directories detected via Composer config where possible

Cautious candidates:

- project `vendor`

Never clean `vendor` by default.

Reason:

`vendor` is generated but deletion can interrupt work and requires reinstall.

## Node

Safe candidates:

- `.next`
- `dist`
- `build`
- `.turbo`
- `.vite`
- `coverage`

Cautious candidates:

- `node_modules`
- pnpm store
- yarn cache depending on location/version
- bun cache depending on location/version

Never clean `node_modules` by default.

Node rules should detect project context using:

- `package.json`
- lockfiles
- known build tool folders

Possible lockfiles:

- `package-lock.json`
- `pnpm-lock.yaml`
- `yarn.lock`
- `bun.lockb`
- `bun.lock`

## Package Managers

Potential safe or cautious caches:

- npm cache
- yarn cache
- pnpm store
- bun cache
- Homebrew cache

Rules must be versioned and individually disableable.

Do not assume one path works for all versions.

Prefer command-discovered cache paths later, but MVP can start with common paths as long as rules are explicit.

## Docker

Docker is report-first for MVP.

Safe-ish report candidates:

- dangling images
- stopped containers
- build cache

Dangerous report candidates:

- volumes
- named volumes
- bind mounts
- database containers
- anything with unclear ownership

Rules:

- Docker cleanup requires `--docker`
- Docker volumes are dangerous by default
- no volume deletion in MVP
- Docker command output should be parsed carefully
- errors should be reported, not hidden

Future commands may include:

- `ratatoskr scan --docker`
- `ratatoskr clean --docker --safe`

But MVP should not clean Docker.

## Generic Logs and Temp Files

Safe only when inside known generated contexts.

Examples:

- project log folders matched by known framework/tool rules
- OS temp directories where ownership and path are clear
- tool-specific temp directories

Avoid:

- deleting any file ending in `.log` anywhere
- deleting arbitrary tmp folders in projects
- deleting unknown files by extension only

## Unknown Large Files

Report-only.

Purpose:

Help user understand disk usage without implying cleanup safety.

Rules:

- never clean by default
- risk dangerous
- category unknown-large-files
- reason must say report-only
- no clean command should include these in MVP

Example:

```text
~/Downloads/archive-2021.zip
Size: 7.4 GB
Risk: dangerous
Reason: Large user-owned file. Report-only. Ratatoskr will not clean this.
```

## Path Safety

Before deletion, Ratatoskr must:

- expand paths
- resolve real paths
- check path exists
- check path type
- check path is not symlink escape
- check path is inside allowed root
- check path is not protected
- check path is not a scan root
- check path is not home directory
- check path is not `/`
- check path is not a repository root
- check path is not project root
- re-stat immediately before deletion

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
- project root
- repository root
- database files
- `.env` files

Deletion must target concrete resolved paths only.

No deletion from unresolved globs.

## Symlink Policy

Default:

- do not follow symlinks for deletion
- report symlinks separately if matched
- never delete symlink targets outside allowed roots
- never allow symlink traversal to escape scan root

Later optional behavior:

- explicit rule may allow deleting the symlink itself, not its target
- following symlink targets should remain non-MVP

## Race and Timing Safety

Files can change after scan.

Before deletion:

- check candidate still exists
- check candidate type is unchanged
- check candidate resolved path is unchanged
- check candidate modified time is not newer than scan time, where applicable
- check size has not changed unexpectedly

If changed:

- skip
- report as skipped due to change

Do not try to be clever.

Skip beats regret.

## Error Handling

Report separately:

- unreadable paths
- permission denied
- disappeared paths
- changed paths
- trash failures
- deletion failures
- rule parse failures
- config validation failures

Errors should not be hidden inside verbose mode only.

A cleanup summary must include:

- cleaned count
- cleaned size
- skipped count
- failed count
- failure reasons

## Performance

MVP should handle large project folders sensibly.

Required:

- avoid scanning protected directories unnecessarily
- prune ignored folders where possible
- avoid descending into `.git`
- avoid calculating sizes twice when avoidable
- show progress for long scans
- allow path-specific scan

Useful later:

- max depth
- parallel walking
- scan cache
- incremental scan
- ignore file support

Performance must not weaken safety.

## Output Formats

## Text

Default human output.

Should be readable, grouped, and compact.

## JSON

Machine-readable output.

Must include:

- `schema_version`
- `generated_at`
- `scan_roots`
- `totals`
- `candidates`
- `skipped`
- `errors`
- `active_rules`

JSON reports contain private local paths.

## Markdown

Useful for saving/sharing.

Must include privacy note.

Should not include any hidden destructive command.

## UX Voice

Tone:

- calm
- useful
- direct
- lightly mythic
- not goofy
- not corporate
- not fantasy cosplay

Good:

```text
Ratatoskr found 12.6 GB of reclaimable disk waste.
```

Good:

```text
3.4 GB is safe to clean now.
7.8 GB requires explicit selection.
1.4 GB is report-only.
```

Good:

```text
The roots are noisy.
```

Bad:

```text
By Odin's beard, your disk is cursed!
```

Bad:

```text
Summoning sacred cleanup ritual...
```

Bad:

```text
AI has determined these files are useless.
```

Voice rule:

Use mythology as seasoning, not soup.

## Platform

Initial target:

- macOS first
- Linux later
- Windows not required for MVP

macOS assumptions:

- trash behavior exists but can fail
- user cache paths are common but not universal
- permissions may block scan/delete
- external disks may behave differently

Linux later:

- trash behavior varies
- XDG paths matter
- headless environments may not have trash

## Language Choice

Language can be decided separately.

Good candidates:

## Go

Pros:

- single binary
- fast filesystem walking
- easy distribution
- good CLI ecosystem

Cons:

- less expressive type modeling than Rust
- safety must be disciplined manually

## Rust

Pros:

- very fitting for safe filesystem tooling
- fast
- strong type modeling
- good for path/rule safety

Cons:

- heavier build ergonomics
- slower to prototype

## PHP / Laravel Zero

Pros:

- fast for Odinn-land
- familiar ecosystem
- fun dogfooding
- easy rich CLI

Cons:

- less ideal as a universal disk cleaner
- runtime dependency story is weaker
- ironic if tool meant to clean dependency bloat

## Node

Pros:

- quick CLI
- rich ecosystem

Cons:

- dependency-heavy
- ironic for a disk cleaner
- distribution less elegant

Recommendation:

Use Go or Rust if Ratatoskr is meant to become a real standalone tool.

Use Laravel Zero only if the first goal is fast personal prototype and ecosystem dogfooding.

Avoid Node unless speed of prototype matters more than product shape.

## Milestones

## Milestone 1: Scan-only MVP

Build:

- `ratatoskr scan`
- `ratatoskr rules`
- `ratatoskr report --format json`

Must include:

- filesystem walking
- size calculation
- narrow built-in rules
- plain category/risk/reason output
- skipped/unreadable paths
- JSON output
- no deletion

Initial rule sets:

- Laravel
- Node
- Composer
- common package caches
- known build folders
- known logs
- unknown large files as report-only

Success criteria:

- can run in a project folder
- can run against `~/Code` when explicitly passed
- finds obvious generated waste
- does not flag personal documents as cleanable
- explains every candidate
- output is readable enough to trust
- no mutation occurs

## Milestone 2: Safe clean

Build:

- `ratatoskr clean --safe`
- `ratatoskr clean --safe --dry-run`
- `ratatoskr clean --safe --yes`
- `ratatoskr clean --safe --delete`

Must include:

- exact deletion list
- confirmation prompt
- trash-first behavior
- delete fallback only with explicit `--delete`
- re-stat before deletion
- symlink protection
- protected path checks
- skipped/error summary

Success criteria:

- only safe rule-level candidates are cleaned
- cautious and dangerous items are excluded
- trash failure aborts by default
- changed paths are skipped
- user can understand exactly what happened

## Milestone 3: Config and reports

Build:

- `.ratatoskr.yml`
- `ratatoskr.yml`
- `ratatoskr report --format markdown`
- rule include/exclude config
- path exclusions
- config validation

Must include:

- invalid config errors
- rule listing with source
- custom rule guardrails
- no protected path override

Success criteria:

- user can configure scan roots
- user can disable rules
- user can exclude paths
- user can inspect effective rules
- config cannot turn Ratatoskr into arbitrary `rm -rf`

## Milestone 4: Cautious opt-in

Build later:

- `ratatoskr clean --include-risk=cautious`

Must include:

- stronger confirmation
- clear rebuild/time cost warning
- no dangerous inclusion
- no `--all`

Candidates:

- `node_modules`
- `vendor`
- selected package stores
- selected Docker images/build cache, if Docker support is ready

Success criteria:

- user knows this may cost reinstall/rebuild time
- no data-like artifacts included
- no Docker volumes included

## Non-Goals For Now

Do not build:

- GUI
- daemon/watch mode
- cloud sync
- system optimizer
- duplicate photo detection
- AI cleanup
- auto-clean background agent
- aggressive home-folder cleanup
- arbitrary Downloads cleanup
- Docker volume deletion
- dangerous cleanup workflow
- "clean everything"
- personal file inference
- file age based deletion
- extension-only deletion
- magic recommendations

## Open Questions

These should be decided before implementation or during early prototype:

1. Language choice: Go, Rust, or Laravel Zero?
2. Should macOS trash be implemented via native API, trash CLI dependency, or library?
3. Should scan default to current directory only, or include known cache paths?
4. Should package manager cache paths be hardcoded first or discovered by commands?
5. Should Ratatoskr maintain a scan cache, or always scan fresh?
6. Should rules live in code first, YAML later, or YAML from day one?
7. Should unknown large file reporting be included in MVP, or saved for later?

## Hard Decisions Already Made

- No `clean --all` in MVP.
- No dangerous cleanup in MVP.
- Docker volumes are dangerous by default.
- Unknown large files are report-only.
- Mythic labels are display only.
- Safety is rule-level, not category-level.
- `scan` and `report` are read-only.
- `clean` is explicit and trash-first.
- Personal files are never deleted by inference.
- Re-stat before delete.
- Symlink deletion is conservative by default.

## One-Line Identity

Ratatoskr is a small, fast filesystem cleaner that runs the tree, reports the rot, and safely prunes generated waste.

## Short Pitch

Ratatoskr scans your filesystem like Yggdrasil, finds generated waste in the roots and branches, explains what is safe to prune, and refuses to touch personal data by guesswork.

## Confidence

Conditionally confident.

The strategy is sound if safety remains rule-level, deletion stays explicit, Docker volumes remain dangerous/report-only, and implementation treats path handling, symlinks, trash behavior, and race conditions as first-class safety work rather than boring plumbing.
