# Release Checklist

Current public release marker: v0.2.0.

Use this checklist before publishing a Goncho release or asking users to upgrade pinned hosts.

## Local validation

1. Run formatting and unit tests:

   ```bash
   go test ./...
   git diff --check
   ```

2. Run release smoke:

   ```bash
   make release-smoke
   ```

3. Run stable end-to-end benchmark smoke when retrieval behavior changed:

   ```bash
   make stable-e2e-bench-smoke
   ```

4. Verify public module import behavior:

   ```bash
   make public-module-smoke
   ```

5. Verify local operator surfaces:

   ```bash
   go run ./cmd/goncho version --json
   go run ./cmd/goncho schema-fingerprint --json
   go run ./cmd/goncho doctor --json --db ./goncho.db
   ```

## Publishing checks

1. Draft the GitHub release with changelog highlights, migration notes, and any security notes.
2. Confirm GitHub release assets/tags match the module version.
3. Verify pkg.go.dev renders the new version and package docs.
4. Run `go list -m -json github.com/TrebuchetDynamics/goncho@latest` from a clean environment.
5. Update public release metadata in `Makefile` only after the new release is visible.

## Post-release

- Run `go run ./cmd/goncho upgrade-check --json --current <old> --latest <new>` and verify it reports the expected update.
- Keep benchmark claims tied to exact artifacts and commands.
- Do not publish connector apply instructions unless the connector has golden tests and host smoke coverage.
