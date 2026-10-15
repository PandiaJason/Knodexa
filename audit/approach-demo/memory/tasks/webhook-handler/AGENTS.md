# Task webhook-handler: webhook-handler

Goal: 

Resume with `knodexa resume webhook-handler --json`. Restore this branch's established approach before coding. Read reference implementations and referenced skill documents. Record rationale and evidence for deliberate deviations. These generated modules are snapshots; consult canonical context for current decisions, failures, and lessons.

## Approach inherited from service-contract

Handlers delegate to injected services

- Preserve service injection instead of embedding transformations in handlers
- Reference: `reference.py`, recorded SHA256 `46464a9cab084389a92f20ed5675c61a3d3e24f5a966820735d1c79c938d091e`.
- Structural check: `@touched`, required literal `service.transform(value)`, forbidden literal ``.
- Structural check: `@touched`, required literal ``, forbidden literal `.upper()`.

