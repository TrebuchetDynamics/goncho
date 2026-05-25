# Goncho Release Checklist

Use this reference only after loading `goncho-release`.

## Repo-specific facts

- Module path: `github.com/TrebuchetDynamics/goncho`.
- Public service import path: `github.com/TrebuchetDynamics/goncho/service`.
- Current release metadata is guarded by `service/release_metadata_test.go` and `make release-metadata-smoke`.
- Release gate: `make release-smoke` = release metadata, ecosystem smoke, `go test ./...`, `go vet ./...`, `go test -race ./...`, docs-site build.
- Public go.dev publication is tag-based; pushing a valid semver tag makes the Go module discoverable once the proxy/pkgsite catch up.
- GitHub Pages/docs CI is in `.github/workflows/docs-site.yml`; still check all branch workflows with `gh run list` because repo CI can change.

## Preflight commands

```sh
git status --short --branch
git remote -v
git fetch --tags origin
git log --oneline -5
git tag --list 'v*' --sort=-v:refname | head -10
python3 .agents/skills/goncho-release/scripts/next_minor.py
gh auth status
```

Block if:

- branch is not `main` or is behind/diverged from `origin/main`;
- worktree has unrelated edits that release metadata updates would overwrite;
- latest semver tag is not on the expected public release line;
- `gh auth status` lacks release permission and the task requires GitHub release creation.

## Version metadata update map

For `vX.Y.Z` on `YYYY-MM-DD`, inspect and update all public-latest references:

```sh
rg -n 'v[0-9]+\.[0-9]+\.[0-9]+|published [A-Z][a-z]+ [0-9]{1,2}, [0-9]{4}|PUBLIC_LATEST' \
  CHANGELOG.md Makefile README.md service/release_metadata_test.go service/doc.go \
  docs-site/src/content/docs docs
```

Common required updates:

- `CHANGELOG.md`: add `## vX.Y.Z - YYYY-MM-DD` below `Unreleased`; keep old release sections below it.
- `Makefile`: set `PUBLIC_LATEST_VERSION := vX.Y.Z` and `PUBLIC_LATEST_PUBLISHED_DATE := YYYY-MM-DD`.
- `README.md`: release badge, public latest prose, `go get ...@vX.Y.Z`, go.dev signal map.
- `docs-site/src/content/docs/index.md`, `start/current-capabilities.md`, `start/quick-start.md`: public latest prose and version-qualified commands.
- `service/doc.go`: pkg.go.dev package docs if they mention a concrete version/date.
- `service/release_metadata_test.go`: guarded constants/markers for version and published date.

Use exact replacements and re-run `rg` until stale references are intentional historical changelog entries only.

## Validation sequence

Local, before commit/tag:

```sh
make release-metadata-smoke
go test ./...
go vet ./...
cd docs-site && npm run build
```

Full local gate before publish:

```sh
make release-smoke
```

If `CHANGELOG.md` already has the new release heading, create the local annotated tag before `make release-smoke` so `TestChangelogReleaseHeadingsHaveMatchingTags` can pass:

```sh
git commit -m "Release Goncho vX.Y.Z"
git tag -a vX.Y.Z -m "Goncho vX.Y.Z"
make release-smoke
```

## Publishing sequence

Only after explicit user confirmation:

```sh
git push origin main
gh run list --branch main --limit 5
# inspect any failed run before continuing:
# gh run view <run-id> --log-failed

git push origin vX.Y.Z
```

Create notes from the new changelog slice (manual extraction is acceptable; keep generated temp files out of git unless intentionally tracked), then publish:

```sh
gh release create vX.Y.Z --title "Goncho vX.Y.Z" --notes-file /tmp/goncho-vX.Y.Z-notes.md
```

Verify GitHub:

```sh
gh release view vX.Y.Z --json tagName,name,url,isPrerelease,publishedAt
```

Verify Go module proxy/go.dev package visibility:

```sh
GOPROXY=https://proxy.golang.org,direct go list -m -json github.com/TrebuchetDynamics/goncho@vX.Y.Z
GONOSUMDB=github.com/TrebuchetDynamics/goncho GOPROXY=direct go list -m -json github.com/TrebuchetDynamics/goncho@vX.Y.Z
```

If proxy lookup lags, report lag rather than re-tagging. Retry after a short wait; Go module tags are immutable consumer contracts.

## Failure handling

- Local validation failure: fix the underlying release/docs/code issue, rerun the failed command, then the full gate.
- GitHub Actions failure: inspect with `gh run view --log-failed`; fix on `main` before pushing the tag.
- Pushed tag with broken release: stop and report. Do not delete/recreate the remote tag unless the user explicitly accepts the Go module immutability risk.
- Go proxy lag: wait/retry. Do not change version numbers just because pkg.go.dev is delayed.
