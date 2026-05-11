# Changelog

## v1.0.0 - Unreleased

First public release.

Ratatoskr v1.0.0 is a read-only developer disk scanner and planning tool.

### Added

- `ratatoskr scan` for read-only project and path scans.
- `ratatoskr report --format json` for agent-readable reports.
- `ratatoskr summary --file report.json` for report triage.
- `ratatoskr summary --target 17GB` for read-only target-space projection.
- `ratatoskr rules` for inspecting built-in rules.
- Artifact metadata: project root, marker file, project type, and artifact path.
- Rebuild cost, reclaim durability, preferred cleanup, and consequence fields.
- Bundled agent skill at `skills/ratatoskr-report-analysis/SKILL.md`.

### Not Included

- No cleaning command.
- No `clean --all`.
- No background monitor.
- No automatic deletion.
- No dangerous cleanup workflow.

Cleaning can wait until scanning is boringly reliable.
