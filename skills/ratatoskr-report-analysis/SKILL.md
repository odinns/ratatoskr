---
name: ratatoskr-report-analysis
description: Analyze Ratatoskr scan JSON reports, triage cleanup candidates, explain risk, and recommend safe next actions without deleting anything.
---

# Ratatoskr Report Analysis

Use this skill when the user gives you a Ratatoskr JSON report or asks what to delete after running Ratatoskr.

Ratatoskr is the scanner. This skill is the judgment layer.

Do not delete files. Do not suggest broad deletion commands unless the user explicitly asks for a concrete cleanup command and the target is narrow, understood, and recoverable.

## Main Rescue Case

The core use case is simple:

```text
I am running out of disk space and need 17 GB for this dataset to be processed.
```

In that situation, do not produce a giant cleanup essay. Find the shortest safe path to enough space.

1. Confirm how much space the user needs.
2. Read the Ratatoskr report.
3. Rank candidates by safe reclaimed space first.
4. Add cautious rebuildable targets only if safe waste is not enough.
5. Keep do-not-touch items visible, but out of the action path.
6. End with the next concrete command or manual review step.

The user needs disk space, not a museum tour of every suspicious file on the machine.

## Inputs

Accept any of these:

- a path to a Ratatoskr JSON report
- pasted Ratatoskr JSON
- summarized Ratatoskr output
- a partial report from an interrupted streamed scan

If the report path exists locally, read it directly. Prefer `jq` or a small script for summarizing large reports.

## First Pass

1. Validate that the report is complete JSON.
2. If incomplete, say so and analyze only what can be safely read if possible.
3. Record:
   - `schema_version`
   - scan roots
   - candidate count
   - totals
   - skipped count
   - error count
4. Look for report quality problems before cleanup advice:
   - duplicate physical paths
   - suspicious sparse-file sizes
   - system paths mixed with user paths
   - mounted volumes included unexpectedly
   - personal data stores treated as generic large files

## Cleanup Ranking

Group findings into four buckets.

### Likely Safe Generated Waste

Examples:

- Laravel logs
- framework caches
- build output
- test coverage databases
- temporary test browser installs

Still explain deletion cost. Safe does not mean silent.

### Cautious Rebuildable Waste

Examples:

- `node_modules`
- `vendor`
- package-manager caches
- Playwright, Puppeteer, Cypress, Homebrew caches

These are usually recoverable but may cost time, downloads, or a broken working session.

### Manual Review Targets

Examples:

- application support snapshots
- local model files
- duplicate backup archives
- old project clones
- DB recovery folders

These can be excellent cleanup targets, but the user must recognize the data.

### Do Not Touch Blindly

Examples:

- database files
- Photos libraries
- Mail stores
- Messages attachments
- active app support state
- system files
- mounted backup drives
- unknown large personal media

Say plainly why deletion is risky.

## Output Shape

Lead with the decision, not the machinery.

Use this structure:

```text
Short verdict.

Best cleanup targets:
...

Needs manual review:
...

Do not delete blindly:
...

Tool/report issues:
...

Next command:
...
```

Keep it tight. The report can be huge; the user needs a shortlist, not a landfill tour.

For the 17 GB rescue case, include a running total:

```text
Need: 17 GB
Safe cleanup: 7.4 GB
Still needed: 9.6 GB
Best cautious target: node_modules across old projects, 10.8 GB
Decision: safe cleanup alone is not enough; review the cautious dependency folders next.
```

## Command Advice

Prefer app-native cleanup commands when they exist:

- `npm cache clean --force` only when the user accepts the tradeoff.
- `brew cleanup` for Homebrew caches.
- package-manager cache commands over deleting internals by hand.
- stop DBngin/MySQL before any database-folder cleanup.

For manual trashing, prefer moving whole recognized folders to Trash, not deleting random files inside live app or database trees.

Never recommend deleting:

- `.ibd` files from a live MySQL data directory
- arbitrary files inside `Photos Library.photoslibrary`
- arbitrary files inside `Mail`
- arbitrary files inside `.git/objects`
- system Preboot, Cryptex, dyld cache, or Xcode internals unless the user is uninstalling the owning tool

## Feeding Back Into Ratatoskr

When the report exposes scanner weaknesses, capture them as implementation work:

- bad size accounting
- duplicate path counting
- missing rule categories
- noisy permissions
- poor summary output
- traversal that wastes time

If the user uses Runes, write those as atomic Runes tasks.

## Tone

Use Odinn's direct, practical voice.

The point is not to sound clever. The point is to prevent expensive stupidity.
