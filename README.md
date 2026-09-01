# Gooo Metamorphic Counterexample Synthesizer

This repository is a small, evidence-bounded metamorphic runner for Gooo.
The `.gooo` declaration owns the semantic invariants, candidate transformation
space, finite bounds, and proof boundary. Go supplies only the evaluator,
candidate generator, and runtime that execute that declaration.

The fixed v1 corpus contains exactly eight scenarios: three bounded CLOSED
relations, two reproducible REFUTED relations, and three UNKNOWN frontiers for
an exhausted bound, a missing oracle, and a stale relation contract. A CLOSED
claim means only that every candidate in the declared finite domain was
executed and produced zero counterexamples. It does not claim global
completeness.

Each synthesis writes only to a caller-owned empty output directory:

- `synthesis-manifest.json`
- `candidate-events.ndjson`
- `counterexample-receipt.json`
- `metamorphic-counterexample-report.md`

REFUTED receipts retain the first reproducible counterexample's candidate
index, before/after source, IR, trace, and digest. UNKNOWN receipts retain the
required `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and
`blocked_by` fields. Resolution is `REFUTED > UNKNOWN > CLOSED`.

All repository, commit, push, merge, and release authority is recorded as
zero inside the product contract. Generated evidence is caller-owned output;
the GitHub delivery workflow is the only verification authority.
