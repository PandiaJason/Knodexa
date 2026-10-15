# Shared-conversation contamination pilot

Both Codex agents preserved the established handler design after performing unrelated image work with conflicting conventions in the same conversation. Both passed independent service-substitution, call-count, exception, and AST checks. There is no quality winner in this pilot.

Knodexa selected `service-design` automatically by `handlers/` file scope, without a task parent/reuse link. Its return boundary named `image-design` as an approach to set aside. Native memory used explicit scoped instructions in a Markdown file.

The experiment establishes the approach, resumes the same chat for an image task requiring embedded transforms, then resumes that same chat for new handler work requiring service delegation. Both arms use the same model, low reasoning effort, and standard-library code. Background generated memories are disabled. Return prompts explicitly request preservation of prior handler conventions in both arms. Knodexa metadata capture is harness-prepared, not automatic extraction, and excluded from timing.

Limitations: one pair, one intervening task, short conversation, simple contract, fixed arm order, and different generated histories. This does not reproduce a saturated context window or prove that Knodexa improves model behavior over native memory. Long repeated conflicting work, compaction, multiple domains, and repeated randomized trials remain to evaluate. The scope router and boundary have regression tests independent of these model outcomes.

Raw transcripts, final code, restored context, and usage metrics are beside this report. `audit/scope-checks.log` records the rebuild's standalone checks.
