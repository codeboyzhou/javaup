# Contributing

## Before opening a change

Use the checked-in Rust toolchain and run:

```text
cargo fmt --all -- --check
cargo clippy --workspace --all-targets --all-features --locked -- -D warnings
cargo test --workspace --all-targets --all-features --locked
RUSTDOCFLAGS="-D warnings" cargo doc --workspace --all-features --no-deps --locked
```

On PowerShell, set the documentation flags with
`$env:RUSTDOCFLAGS='-D warnings'` before the Cargo command.

Changes to platform-native paths, executable discovery, persistence, or process
creation must be checked on Windows, Linux, and macOS through CI. Persistence
format changes require a new schema version, a migration test, and a guarantee
that malformed existing data is not overwritten.

## Design expectations

- Keep UI and process-execution adapters outside `javaup-core`.
- Preserve `Path` and `OsStr` values at OS boundaries.
- Return typed errors with their original source instead of flattening them to
  strings or generic I/O errors.
- Add focused unit tests for edge cases and integration tests for user-visible
  behavior. Prefer property tests for codecs and parsers with broad input spaces.
- Never log raw passwords, tokens, credentials, private keys, or access keys.

## Commits

Use Conventional Commits in English:

```text
type(scope): lowercase imperative subject
```

Allowed types are `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
`build`, `ci`, and `chore`. Keep the subject at 72 characters or fewer and do
not end it with a period.
