package synth

import "testing"

func TestResolutionPrecedence(t *testing.T) {
	results := []ScenarioResult{
		{Decision: StateClosed}, {Decision: StateUnknown}, {Decision: StateRefuted},
		{Decision: StateClosed}, {Decision: StateUnknown}, {Decision: StateClosed},
		{Decision: StateClosed}, {Decision: StateClosed},
	}
	counts, decision, err := summarize(results)
	if err != nil {
		t.Fatal(err)
	}
	if decision != StateRefuted || counts.Closed != 5 || counts.Unknown != 2 || counts.Refuted != 1 {
		t.Fatalf("unexpected summary: %+v %s", counts, decision)
	}
}

func TestAlphaRenamingPreservesSemanticIR(t *testing.T) {
	before, err := deriveIR("alpha_renaming", "let x = 1 in x + 1")
	if err != nil {
		t.Fatal(err)
	}
	afterSource, err := transform("alpha_renaming", "rename_x_y", "let x = 1 in x + 1")
	if err != nil {
		t.Fatal(err)
	}
	after, err := deriveIR("alpha_renaming", afterSource)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("alpha-renaming changed normalized IR: %q != %q", before, after)
	}
}

func TestArithmeticRewriteProducesReproducibleCounterexample(t *testing.T) {
	before, err := deriveTrace("arithmetic_rewrite", "sub(4,3)")
	if err != nil {
		t.Fatal(err)
	}
	after, err := deriveTrace("arithmetic_rewrite", "add(4,3)")
	if err != nil {
		t.Fatal(err)
	}
	if sameStrings(before, after) {
		t.Fatalf("arithmetic rewrite unexpectedly preserved trace: %v", before)
	}
}

func TestUnknownRecordRequiresAllSixFields(t *testing.T) {
	valid := &UnknownRecord{Stage: "synthesis", Step: "step", Reason: "reason", UnknownClass: "class", NextOperation: "next", BlockedBy: []string{"blocker"}}
	if !valid.Valid() {
		t.Fatal("valid UNKNOWN tuple rejected")
	}
	valid.BlockedBy = nil
	if valid.Valid() {
		t.Fatal("incomplete UNKNOWN tuple accepted")
	}
}
