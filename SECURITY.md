# Security

Ratatoskr reads local filesystem metadata and emits local paths.

That makes reports useful. It also makes them private.

## Reporting Issues

Report security issues privately to the maintainer before opening a public issue. Do not paste reports that include private paths, usernames, project names, customer names, or mounted volume names.

## Report Privacy

Ratatoskr reports can contain:

- absolute local paths
- project names
- dependency folders
- local model names
- skipped paths
- scan errors

Treat JSON and text reports as private. Redact paths before sharing them in public issues, chat, screenshots, or agent transcripts.

## Deletion Safety

Ratatoskr v1.1.0 does not delete files.

Future cleaning features must keep deletion explicit, trash-first where possible, and backed by concrete resolved paths. Unknown large files, personal data, databases, Docker volumes, and dangerous candidates must stay out of default cleanup flows.

## Secrets

Do not share `.env` files, credentials, tokens, private keys, or production config in reports or bug examples. Ratatoskr should skip protected files, but the user still owns the final share button. Tiny button. Large consequences.
