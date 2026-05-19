# Goncho TODO

[BLOCKED] Full Go verification for local smoke docs — 2026-05-19 15:46:32 CST
  blocker: `go test ./...` cannot compile because an unrelated uncommitted `review_test.go` addition references missing review-resolution APIs.
  evidence: `./review_test.go:102:23: svc.ResolveReviewItem undefined`; also `undefined: ReviewResolutionParams` and `undefined: ReviewResolutionSuperseded`.
  unblocks when: review-resolution production API is implemented or the unrelated `review_test.go` WIP is removed/stashed by its owner.
  owner: person/agent currently editing `review_test.go`.
  workaround/pivot: validated docs with `cd docs-site && npm run build`; committed only docs/TODO changes, leaving `review_test.go` untouched.
  next check: 2026-05-19 17:00 CST
