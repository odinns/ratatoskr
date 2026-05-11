# Ratatoskr

Ratatoskr is a conservative local disk scanner for developer machines.

It finds generated waste, explains why it was flagged, shows the rebuild cost, and helps you plan enough reclaimed space without deleting anything. v1.0.0 is read-only. No clean command. No background agent. No mystery button with teeth.

## What It Does

- Scans project trees and explicit paths.
- Flags known generated waste from Laravel, Node, Composer, Rust, package caches, app caches, and local AI model stores.
- Labels each candidate as `safe`, `cautious`, or `dangerous`.
- Explains the rule, reason, consequence, rebuild cost, durability, and default cleanability.
- Produces JSON reports for humans, scripts, and local agents.
- Projects a target-space plan with `summary --target 17GB`.
- Ships an agent instruction file at `skills/ratatoskr-report-analysis/SKILL.md`.

## What It Refuses

- It does not delete files in v1.0.0.
- It does not scan your whole home directory by default.
- It does not treat large files as trash.
- It does not infer-delete personal files.
- It does not clean Docker volumes, databases, photos, downloads, mail stores, or unknown large files.

Unknown stays report-only. That is the point.

## Install

From source:

```sh
go install github.com/odinns/ratatoskr/cmd/ratatoskr@v1.0.0
```

For local development:

```sh
go build ./cmd/ratatoskr
```

## Quick Start

Scan the current directory:

```sh
ratatoskr scan
```

Scan a developer tree:

```sh
ratatoskr scan --path ~/Code
```

Write a JSON report:

```sh
ratatoskr report --path ~/Code --format json > ratatoskr-report.json
```

Plan for a target amount of space:

```sh
ratatoskr summary --file ratatoskr-report.json --target 17GB
```

Get the same projection as JSON:

```sh
ratatoskr summary --file ratatoskr-report.json --target 17GB --format json
```

Inspect the built-in rules:

```sh
ratatoskr rules
```

## The 17 GB Case

Ratatoskr is built for the annoying moment where a dataset, build, or export needs more disk than you have.

The useful flow is:

```sh
ratatoskr report --path ~/Code --format json > ratatoskr-report.json
ratatoskr summary --file ratatoskr-report.json --target 17GB
```

The projection picks safe generated waste first. If that is not enough, it adds cautious rebuildable candidates. Report-only paths are excluded from the action path and shown as warnings.

That output is a plan. It is not permission to delete blindly.

## Agent Skill

The repo includes:

```text
skills/ratatoskr-report-analysis/SKILL.md
```

Use it with Codex, Claude, Gemini, or another local agent that can read files. Give the agent the skill and the JSON report. It should rank likely safe targets, cautious rebuildable targets, manual-review targets, and do-not-touch paths.

Example prompt:

```text
Use skills/ratatoskr-report-analysis/SKILL.md.
Analyze ratatoskr-report.json.
I need 17 GB.
Tell me what to inspect first.
Do not suggest deletion commands yet.
```

Reports contain local file paths. Treat them as private.

## Release Checks

Before tagging:

```sh
go test ./...
git diff --check
make dist
```

Release smoke check:

```sh
ratatoskr scan --path <fixture>
ratatoskr report --path <fixture> --format json > report.json
ratatoskr summary --file report.json --target 17GB
ratatoskr summary --file report.json --target 17GB --format json
ratatoskr rules
```

None of those commands should mutate files.
