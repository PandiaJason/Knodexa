# Memory mechanisms adapted for coding continuity

Reviewed primary documentation on 2026-10-14. This is a representative survey,
not an exhaustive review of every memory product or a comparative benchmark.
The adaptations below are engineering choices for Knodexa's problem, not claims
that the cited products use identical implementations.

| Source | Documented mechanism | Knodexa adaptation and status |
|---|---|---|
| [Letta memory blocks](https://docs.letta.com/v1-sdk/memory/memory-blocks) | Persistent named blocks, shared context and read-only blocks | Existing scoped agreements and pre-execution pinned guard; now expose a clear mandatory/advisory policy in restored context. Knodexa does not enforce OS-level read-only memory. |
| [LangGraph memory overview](https://docs.langchain.com/oss/python/concepts/memory) | Namespaced long-term memory; semantic, episodic and procedural memories; updating collections has tradeoffs | Existing task/file namespaces and distinct decisions, failures, lessons and agreements. New bounded deferred pointers let agents expand budget-pruned detail while mandatory constraints and acceptance remain intact. |
| [Zep unified context](https://help.getzep.com/concepts) | Temporal facts, invalidation and provenance | New validated supersession records replacement, explicit reason and recorded invalidation time on decisions and lessons. Historic bodies remain inspectable; retired items are excluded from active retrieval. This is not a full bitemporal fact graph. |
| [Mem0 quickstart](https://docs.mem0.ai/platform/quickstart) | Agent memory operations, scoped search and updates | Explicit evidence-backed admission already exists. New CLI/MCP retirement makes replacing obsolete knowledge an agent-accessible operation; no inferred deletion of project agreements. |
| [Cognee search API](https://docs.cognee.ai/api-reference/introduction) | Multiple retrieval modes including graph, lexical and temporal search | Existing identity phrases, source anchors and relationships; new lesson recall includes selected agreement sources and ranks relationships/file relevance ahead of confidence. No embedding or general semantic-search equivalence is claimed. |
| [MCP knowledge graph memory server](https://github.com/modelcontextprotocol/servers/blob/main/src/memory/README.md) | Atomic observations, directed relations, targeted node reads and shared-file write serialization | Existing canonical atomic records, directed links and targeted detail expansion. New context provenance exposes source/evidence and highest-priority match reason. Cross-process read-modify-write serialization remains a separate gap; atomic file replacement alone does not solve it. |

## Implemented in this iteration

1. Relevant lessons survive competition with high-confidence weak matches.
   Recall priority is selected task/agreement source, explicit relationship, file
   overlap, tag overlap, then confidence and stable ID. Only VERIFIED lessons
   qualify. Ownership and file overlap are retrieval evidence, not permission for
   advisory lessons to override an agreement. This is a heuristic, not a learned
   relevance model.
2. A task restores lessons from its selected agreement sources even without
   identical touched-file metadata. Unrelated source lessons are not admitted
   merely for having high confidence.
3. Decision/lesson summaries carry source and evidence IDs. Compact contexts
   retain a count and up to eight pointers for budget-pruned detail; the readable
   handoff shows up to three sample IDs. These cover retrieved-but-pruned items,
   not every item outside the five-lesson retrieval limit. Full context with a
   larger budget and targeted detail reads provide expansion.
4. Supersession validates a different, existing VERIFIED replacement of the same
   kind and a nonblank reason before modifying canonical memory. Original content
   and evidence remain; replacement, invalidation time and reason persist through
   index rebuild. Retired knowledge cannot be silently re-retired.

```sh
knodexa supersede L-old --by=L-new --reason='Verified interface migration'
knodexa show L-old --json
knodexa context <task-id> --json
```

MCP: `knodexa_supersede_knowledge` accepts `node_id`, `replacement_id`, `reason`.

## Validation and remaining work

`GOCACHE=/private/tmp/knodexa-go-cache ./scripts/check.sh` runs Go vet, race tests
and standalone CLI smoke. New regressions cover relevance under a result limit,
selected-source lessons, rejection of unverified/missing/self/cross-kind
replacements, immutable historical body, exclusion after rebuild, CLI/MCP parity,
and preservation of constraints, acceptance criteria and failure memory under an
impossible context budget. No paid model comparisons were run in this iteration.

Next priorities are real host integration, concurrent-agent write safety,
reviewable agreement capture and measured semantic-retrieval recall. These are
not implemented by merely adding a protocol or adopting vocabulary from another
tool. Existing comparisons against competent scoped file memory remain tied or
inconclusive; superiority over all memory systems is unproven.
