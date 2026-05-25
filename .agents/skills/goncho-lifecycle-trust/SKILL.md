---
name: goncho-lifecycle-trust
description: Implement Goncho lifecycle and trust behavior. Use when working on temporal validity, supersession, conflict review, audit trails, verification, stale-memory warnings, quarantine, or redaction.
---

# Goncho Lifecycle and Trust

## Quick start

Load `goncho-tdd-implementation`, then prove that lifecycle/trust state changes what users see or what APIs allow.

## Workflow

1. **Choose one trust failure**
   - Expired or not-yet-valid memory.
   - Newer evidence superseding old truth.
   - Conflicting claims requiring review.
   - Low-confidence or low-authority evidence.
   - Stale code/path verification.
   - Prompt-injection or secret quarantine/redaction.
2. **Write the contract test**
   - Prefer user-visible behavior: search ranking, context warnings, review API output, audit entries, or tool response.
3. **Implement minimally**
   - Preserve historical evidence.
   - Make current truth distinguishable from past/conflicting truth.
   - Explain warnings or review-required state in API/context output.
4. **Verify**
   - Run the narrow test, relevant trust/review tests, then `go test ./...`.

## Behavior rule

Lifecycle state is not complete unless it affects at least one of: storage mutation, search ranking, context inclusion, review/audit output, API warning fields, or tool responses.

## Skill contract

### Entry protocol
- Trivial: inspect or explain current lifecycle/trust behavior directly.
- Medium ambiguity: propose one trust failure and ask only which consumer must observe it first.
- High ambiguity/risk: stop before destructive overwrites, raw secret export, auth weakening, or silent conflict suppression.

### Topology check
- State/ownership: lifecycle fields, evidence chains, review state, audit log, verification source.
- Feedback/validation: behavior proof in search/context/review/audit/tool output.
- Blast radius: retrieval ranking, host decisions, import/export, server/team ACLs, and public API contracts.
- Timing/ordering: supersession order, expiry windows, verification freshness, quarantine before indexing.

### Verification gate
Done requires preserved history, distinguishable current truth, visible stale/conflict/review warnings where applicable, and `go test ./...` pass or blocker output.

### Red lines
- Do not overwrite old truth destructively.
- Do not silently ignore or drop conflicts.
- Do not add review tables without queryable API/tool behavior.
- Do not treat confidence/authority as decorative metadata.
- Do not expose redacted/quarantined secret content in normal outputs.

### Output contract
End with: trust failure covered, user-visible behavior, test names, validation commands, and unresolved review/security risks.
