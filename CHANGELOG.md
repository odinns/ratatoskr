# Changelog

## v1.1.0 - 2026-05-11

Scan/report closeout release.

Ratatoskr v1.1.0 stays read-only. The useful changes are better project artifact metadata, narrower cache and state rules, and a summary that points at the right work instead of dumping a giant list on the floor.

### Added

- Project metadata for Node artifacts and Composer `vendor`.
- Node artifact detection for `.nuxt`.
- Project artifact rules for Swift `.build`, Terraform `.terraform`, Serverless `.serverless`, Gradle `build`, and Maven `target`.
- Named cautious cache rules for Homebrew, Cypress, Playwright, Puppeteer, npm, and Composer caches.
- Known-state rules for DBngin database state, DBngin recovery folders, Chrome on-device model weights, and Cursor snapshots.
- `summary` project artifact groups ranked by project root size.
- `summary` report-quality hints for duplicate skips, scan errors, high skipped counts, and broad fallback cache rules dominating a report.
- Target projection running totals in text and JSON.

### Changed

- `summary --format json` now includes `project_artifact_groups` and `report_quality_hints`.
- The bundled report-analysis skill now understands project groups, report-quality hints, and the new artifact/cache/state categories.
- Site and README now describe v1.1 behavior instead of the v1.0 surface.

### Not Included

- No cleaning command.
- No config file.
- No staleness scoring.
- No private dependency access checks.

## v1.0.0 - 2026-05-11

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
