package synth

import (
	"fmt"
	"strings"
)

func renderReport(manifest SynthesisManifest, receipt CounterexampleReceipt) string {
	var builder strings.Builder
	builder.WriteString("# Metamorphic counterexample synthesis report\n\n")
	fmt.Fprintf(&builder, "- decision: `%s`\n- fixed scenarios: `%d`\n- candidates generated: `%d`\n- candidates executed: `%d`\n- counterexamples preserved: `%d`\n- precedence: `%s`\n- source digest: `%s`\n- contract digest: `%s`\n- semantic IR digest: `%s`\n\n", receipt.Decision, receipt.Counts.Scenarios, receipt.Counts.CandidatesGenerated, receipt.Counts.CandidatesExecuted, receipt.Counts.Counterexamples, strings.Join(receipt.Precedence, " > "), receipt.SourceDigest, receipt.ContractDigest, receipt.IRDigest)
	builder.WriteString("CLOSED is limited to the declared finite candidate domain. It means every candidate in that domain was executed and no counterexample was observed; it does not establish global completeness.\n\n")
	builder.WriteString("## Scenario results\n\n")
	builder.WriteString("| scenario | expected | decision | bound | domain | generated | executed | counterexamples |\n")
	builder.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, scenario := range receipt.Scenarios {
		fmt.Fprintf(&builder, "| `%s` | `%s` | `%s` | `%d` | `%d` | `%d` | `%d` | `%d` |\n", scenario.ID, scenario.Expected, scenario.Decision, scenario.Bound, scenario.DomainSize, scenario.CandidatesGenerated, scenario.CandidatesExecuted, scenario.Counterexamples)
	}
	builder.WriteString("\n## REFUTED evidence\n\n")
	for _, scenario := range receipt.Scenarios {
		if scenario.FirstCounterexample == nil {
			continue
		}
		counterexample := scenario.FirstCounterexample
		fmt.Fprintf(&builder, "- `%s`: first counterexample candidate index `%d` (`%s`), digest `%s`, reproducible `%t`.\n", scenario.ID, counterexample.CandidateIndex, counterexample.CandidateID, counterexample.Digest, counterexample.Reproducible)
		fmt.Fprintf(&builder, "  - before source: `%s`\n  - after source: `%s`\n  - before IR: `%s`\n  - after IR: `%s`\n  - before trace: `%s`\n  - after trace: `%s`\n", counterexample.BeforeSource, counterexample.AfterSource, counterexample.BeforeIR, counterexample.AfterIR, strings.Join(counterexample.BeforeTrace, ", "), strings.Join(counterexample.AfterTrace, ", "))
	}
	builder.WriteString("\n## UNKNOWN evidence\n\n")
	for _, scenario := range receipt.Scenarios {
		if scenario.Unknown == nil {
			continue
		}
		unknown := scenario.Unknown
		fmt.Fprintf(&builder, "- `%s`: `%s` — stage `%s`, step `%s`, reason `%s`, unknown_class `%s`, next_operation `%s`, blocked_by `%s`.\n", scenario.ID, scenario.Decision, unknown.Stage, unknown.Step, unknown.Reason, unknown.UnknownClass, unknown.NextOperation, strings.Join(unknown.BlockedBy, ", "))
	}
	builder.WriteString("\n## Authority and improvement boundary\n\n")
	fmt.Fprintf(&builder, "Product repository writes, commit, push, merge, release, and local test executions are recorded as `%d`, `%d`, `%d`, `%d`, `%d`, and `%d`; output is caller-owned only.\n\n", manifest.Authority.RepositoryWrites, manifest.Authority.CommitAuthority, manifest.Authority.PushAuthority, manifest.Authority.MergeAuthority, manifest.Authority.ReleaseAuthority, manifest.Authority.LocalTestExecutions)
	fmt.Fprintf(&builder, "Improvement status is `%s`: a matching before/after integer pair for the same scenario, source, contract, toolchain, and runner was not supplied.\n", manifest.Improvement.Status)
	return builder.String()
}
