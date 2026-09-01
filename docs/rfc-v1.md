# Metamorphic counterexample synthesis protocol v1

The `.gooo` declaration is the authoritative metacode. It declares the
semantic relation, candidate domain, bound, oracle identity, expected state,
and the seven activity edges. The JSON contract is a fixed, checked mirror of
that denominator; it cannot add, remove, reorder, or rebind a relation.

The generator emits candidates in declared order. The evaluator runs each
candidate against the declared oracle twice and requires the same result. The
first reproducible preservation failure is retained as a counterexample with
candidate index, before/after source, IR, trace, and digest.

`CLOSED` is deliberately bounded: candidates generated equals candidates
executed and no counterexample exists in the fixed finite domain. It does not
claim that the relation is globally complete. `REFUTED` outranks `UNKNOWN`,
which outranks `CLOSED`. Every `UNKNOWN` retains the six-field frontier tuple.

Improvement is a separate claim. It remains `UNKNOWN` unless the same
scenario, source, contract, toolchain, and runner have a matching before/after
integer pair. No score or estimated percentage is emitted.
