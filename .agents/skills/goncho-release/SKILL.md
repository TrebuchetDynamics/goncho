---
name: goncho-release
description: Release Goncho next minor version safely. Use when asked to make release CI green, publish Goncho, create a GitHub release, tag a Goncho version, or update pkg.go.dev/go.dev package metadata.
---

# Goncho Release

## Quick start

Run preflight and report the target version before mutating release files:

```sh
git status --short --branch
git fetch --tags origin
python3 .agents/skills/goncho-release/scripts/next_minor.py
```

If unrelated local edits exist, isolate them before editing release metadata.

## Workflow

1. **Establish the target**
   - Confirm `main` tracks `origin/main` and tags are fetched.
   - Derive next minor from latest semver tag, for example `v0.2.0 -> v0.3.0`.
   - Confirm GitHub auth if publishing: `gh auth status`.
2. **Update release metadata**
   - Move `CHANGELOG.md` `Unreleased` content under `## vX.Y.Z - YYYY-MM-DD`.
   - Update `Makefile` `PUBLIC_LATEST_VERSION` and `PUBLIC_LATEST_PUBLISHED_DATE`.
   - Update README/docs/pkg.go.dev references, badges, version-qualified `go get` snippets, package docs, and guarded release metadata tests.
   - Keep module path `github.com/TrebuchetDynamics/goncho` unchanged.
3. **Make CI green locally**
   - Run `make release-metadata-smoke` after metadata edits.
   - Run `make release-smoke` as the release gate.
   - Inspect `go doc ./service` if package docs changed.
4. **Commit and local tag**
   - Stage only release-scope files.
   - Commit: `Release Goncho vX.Y.Z`.
   - Create an annotated local tag: `git tag -a vX.Y.Z -m "Goncho vX.Y.Z"`.
   - Re-run `make release-smoke` with the local tag present.
5. **Publish only after confirmation**
   - Push branch, verify GitHub Actions green, then push tag.
   - Create GitHub release from changelog notes.
   - Verify Go proxy/go.dev visibility with `go list -m -json github.com/TrebuchetDynamics/goncho@vX.Y.Z`.

## Skill contract

### Entry protocol
- Trivial: answer release-process questions without mutating the repo.
- Medium ambiguity: propose next minor from latest tag and ask whether to publish externally now or prepare a local release commit.
- High ambiguity/risk: stop before pushing tags, creating GitHub releases, overwriting dirty files, deleting/moving tags, or publishing with red CI.

### Topology check
- State/ownership: branch, dirty files, latest semver tag, changelog state, GitHub auth, release permissions.
- Feedback/validation: `make release-smoke`, GitHub Actions green, GitHub release visible, Go module proxy/pkg.go.dev metadata reachable.
- Blast radius: public module tags are immutable for Go consumers; docs/tests pin public latest version/date.
- Timing/ordering: commit -> local tag -> local gate -> push branch -> remote CI -> push tag -> GitHub release -> go.dev verification.

### Verification gate
Done requires:
- `make release-metadata-smoke` passes,
- `make release-smoke` passes or blocker output is reported,
- branch push and GitHub Actions are green before tag push,
- GitHub release URL is reported after creation,
- Go proxy sees `github.com/TrebuchetDynamics/goncho@vX.Y.Z` or proxy lag is reported with retry evidence.

### Red lines
- Do not push tags, create GitHub releases, or publish externally without explicit user confirmation in that turn.
- Do not force-push, delete/recreate remote tags, edit history, or overwrite unrelated local work.
- Do not release from a dirty/divergent branch unless dirty files are explicitly release-scoped.
- Do not change module path, license, or public package boundaries just to satisfy pkg.go.dev.
- Do not claim CI green from local build success alone; verify GitHub Actions after push.

### Output contract
End with: target and previous version, files changed, commit/tag IDs, local validation, GitHub Actions status, GitHub release URL, go.dev/proxy verification, and blockers.

## References
- [Goncho release checklist](references/release-checklist.md)
