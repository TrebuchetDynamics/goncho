# Deferred Connector Readiness

Status: deferred

This deferred connector status contract keeps local-first preview work honest before any host-specific connector is presented as supported.

A deferred connector must not advertise install automation until it has:

- a documented preview plan;
- local-first configuration boundaries;
- redaction and rollback notes;
- smoke coverage proving it can talk to `goncho-server` without mutating host state unexpectedly.

Until those checks exist, use generic local configuration only when the host owner opts in manually.
