# Project memory tree

Generated snapshot of recorded project understanding: task goals, agreed approaches, inherited rules, references, and skills. This does not expose hidden model reasoning. Refresh after changing canonical memory.

```mermaid
flowchart TD
n0["first-handler: first-handler"]
n1["image-pipeline: image-pipeline"]
n2["service-contract: service-contract"]
n3["thumbnail-work: thumbnail-work"]
n4["webhook-handler: webhook-handler"]
n2 --> n0
n1 --> n3
n2 -. reuse approach .-> n4
```

| Task | Parent | Established approach sources | Modules |
|---|---|---|---|
| first-handler | service-contract | service-contract | [AGENTS.md](tasks/first-handler/AGENTS.md) · [SKILL.md](tasks/first-handler/SKILL.md) |
| image-pipeline |  | image-pipeline | [AGENTS.md](tasks/image-pipeline/AGENTS.md) · [SKILL.md](tasks/image-pipeline/SKILL.md) |
| service-contract |  | service-contract | [AGENTS.md](tasks/service-contract/AGENTS.md) · [SKILL.md](tasks/service-contract/SKILL.md) |
| thumbnail-work | image-pipeline | image-pipeline | [AGENTS.md](tasks/thumbnail-work/AGENTS.md) · [SKILL.md](tasks/thumbnail-work/SKILL.md) |
| webhook-handler |  | service-contract | [AGENTS.md](tasks/webhook-handler/AGENTS.md) · [SKILL.md](tasks/webhook-handler/SKILL.md) |
