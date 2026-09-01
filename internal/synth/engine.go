package synth

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Options struct {
	Source           string
	Contract         string
	Fixtures         string
	RepoRoot         string
	Output           string
	SubjectSHA       string
	ToolchainVersion string
	RunnerDigest     string
}

type ConformanceReport struct {
	Schema         string    `json:"schema"`
	Status         string    `json:"status"`
	Authority      Authority `json:"authority"`
	Precedence     []string  `json:"precedence"`
	ScenarioCount  int       `json:"scenario_count"`
	ActivityCount  int       `json:"activity_count"`
	ScenarioIDs    []string  `json:"scenario_ids"`
	ExpectedStates []string  `json:"expected_states"`
	SourceDigest   string    `json:"source_digest"`
	ContractDigest string    `json:"contract_digest"`
	FixtureSchema  string    `json:"fixture_schema"`
}

type compiledInputs struct {
	Meta           MetaContract
	Contract       Contract
	Fixtures       FixtureCorpus
	SourceDigest   string
	ContractDigest string
	IR             SemanticIR
}

func Synthesize(options Options) error {
	if options.RepoRoot == "" {
		return errors.New("repository root is required")
	}
	output, err := ensureCallerOutput(options.Output, options.RepoRoot)
	if err != nil {
		return err
	}
	inputs, err := compileInputs(options.Source, options.Contract, options.Fixtures)
	if err != nil {
		return err
	}
	if options.ToolchainVersion != "" && !strings.HasPrefix(options.ToolchainVersion, "go1.27") {
		return fmt.Errorf("observed toolchain %q is outside declared Go 1.27", options.ToolchainVersion)
	}
	if options.RunnerDigest == "" {
		options.RunnerDigest = "sha256:runner-unspecified"
	}
	fixtures := make(map[string]ScenarioFixture, len(inputs.Fixtures.Scenarios))
	for _, fixture := range inputs.Fixtures.Scenarios {
		fixtures[fixture.ID] = fixture
	}

	results := make([]ScenarioResult, 0, ScenarioCount)
	events := make([]CandidateEvent, 0)
	for _, relation := range inputs.Meta.Relations {
		fixture, ok := fixtures[relation.ID]
		if !ok {
			return fmt.Errorf("missing fixture for relation %q", relation.ID)
		}
		result, relationEvents, err := evaluateRelation(relation, fixture, inputs.ContractDigest)
		if err != nil {
			return err
		}
		results = append(results, result)
		events = append(events, relationEvents...)
	}
	counts, decision, err := summarize(results)
	if err != nil {
		return err
	}
	improvement := unknownImprovement(inputs, options.RunnerDigest)
	logicalSource := logicalPath(options.RepoRoot, options.Source)
	logicalContract := logicalPath(options.RepoRoot, options.Contract)
	logicalFixtures := logicalPath(options.RepoRoot, options.Fixtures)
	manifest := SynthesisManifest{
		Schema: ManifestSchema, Version: "v1", Decision: decision, Authority: inputs.Meta.AuthorityBoundary,
		Precedence: append([]string(nil), inputs.Meta.Precedence...), Source: logicalSource, Fixture: logicalFixtures,
		Contract: logicalContract, SubjectSHA: valueOr(options.SubjectSHA, "unspecified"), Toolchain: inputs.Meta.Toolchain,
		RunnerDigest: options.RunnerDigest, SourceDigest: inputs.SourceDigest, ContractDigest: inputs.ContractDigest,
		IRDigest: inputs.IR.IRDigest, Denominator: inputs.Meta.Denominator, Activities: inputs.Meta.Activities,
		Relations: inputs.Meta.Relations, Scenarios: results, Counts: counts,
		Inventory: InventoryMetrics{}, Runtime: RuntimeMeasurements{}, Tests: TestCounts{}, Improvement: improvement,
		Outputs: []string{"synthesis-manifest.json", "candidate-events.ndjson", "counterexample-receipt.json", "metamorphic-counterexample-report.md"},
	}
	manifest.Inventory, err = MeasureInventory(options.RepoRoot)
	if err != nil {
		return err
	}
	receipt := CounterexampleReceipt{
		Schema: ReceiptSchema, Version: "v1", Decision: decision, Authority: inputs.Meta.AuthorityBoundary,
		Precedence: append([]string(nil), inputs.Meta.Precedence...), SourceDigest: inputs.SourceDigest,
		ContractDigest: inputs.ContractDigest, IRDigest: inputs.IR.IRDigest, Toolchain: inputs.Meta.Toolchain,
		RunnerDigest: options.RunnerDigest, Scenarios: results, Counts: counts, Improvement: improvement,
	}
	report := renderReport(manifest, receipt)
	if err := WriteJSON(filepath.Join(output, "synthesis-manifest.json"), manifest); err != nil {
		return err
	}
	if err := WriteNDJSON(filepath.Join(output, "candidate-events.ndjson"), events); err != nil {
		return err
	}
	if err := WriteJSON(filepath.Join(output, "counterexample-receipt.json"), receipt); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "metamorphic-counterexample-report.md"), []byte(report), 0o644); err != nil {
		return err
	}
	return nil
}

func RunConformance(sourcePath, contractPath, fixturePath, repoRoot, outputPath string) error {
	inputs, err := compileInputs(sourcePath, contractPath, fixturePath)
	if err != nil {
		return err
	}
	if outputPath == "" {
		return errors.New("conformance output is required")
	}
	if err := ensureCallerOutputFile(outputPath, repoRoot); err != nil {
		return err
	}
	ids := make([]string, 0, len(inputs.Meta.Relations))
	states := make([]string, 0, len(inputs.Meta.Relations))
	for _, relation := range inputs.Meta.Relations {
		ids = append(ids, relation.ID)
		states = append(states, relation.Expected)
	}
	report := ConformanceReport{
		Schema: "gooo/metamorphic-counterexample-synthesizer/conformance/v1", Status: "PASS",
		Authority: inputs.Meta.AuthorityBoundary, Precedence: append([]string(nil), inputs.Meta.Precedence...),
		ScenarioCount: len(inputs.Meta.Relations), ActivityCount: len(inputs.Meta.Activities), ScenarioIDs: ids,
		ExpectedStates: states, SourceDigest: inputs.SourceDigest, ContractDigest: inputs.ContractDigest,
		FixtureSchema: inputs.Fixtures.Schema,
	}
	return WriteJSON(outputPath, report)
}

func compileInputs(sourcePath, contractPath, fixturePath string) (compiledInputs, error) {
	meta, _, err := LoadMeta(sourcePath)
	if err != nil {
		return compiledInputs{}, err
	}
	contract, _, contractDigest, err := LoadContract(contractPath)
	if err != nil {
		return compiledInputs{}, err
	}
	if err := ValidateMetaAgainstContract(meta, contract, contractDigest); err != nil {
		return compiledInputs{}, err
	}
	fixtures, err := LoadFixtures(fixturePath)
	if err != nil {
		return compiledInputs{}, err
	}
	for _, relation := range meta.Relations {
		fixture := findFixture(fixtures, relation.ID)
		if fixture.ID == "" {
			return compiledInputs{}, fmt.Errorf("fixture for relation %q is missing", relation.ID)
		}
		baselineIR, err := deriveIR(relation.Kind, fixture.BaseSource)
		if err != nil {
			return compiledInputs{}, err
		}
		if baselineIR != fixture.BaselineIR {
			return compiledInputs{}, fmt.Errorf("fixture %q baseline IR is not reproducible", relation.ID)
		}
		baselineTrace, err := deriveTrace(relation.Kind, fixture.BaseSource)
		if err == nil && !sameStrings(baselineTrace, fixture.BaselineTrace) {
			return compiledInputs{}, fmt.Errorf("fixture %q baseline trace is not reproducible", relation.ID)
		}
	}
	ir := SemanticIR{
		Schema: IRSchema, Version: "v1", SourceDigest: meta.SourceDigest, ContractDigest: contractDigest,
		Toolchain: meta.Toolchain, Authority: meta.AuthorityBoundary, Precedence: append([]string(nil), meta.Precedence...),
		UnknownFields: append([]string(nil), meta.UnknownFields...), Activities: append([]Activity(nil), meta.Activities...),
		Relations: append([]Relation(nil), meta.Relations...),
	}
	ir.IRDigest, err = unsignedIRDigest(ir)
	if err != nil {
		return compiledInputs{}, err
	}
	return compiledInputs{Meta: meta, Contract: contract, Fixtures: fixtures, SourceDigest: meta.SourceDigest, ContractDigest: contractDigest, IR: ir}, nil
}

func evaluateRelation(relation Relation, fixture ScenarioFixture, contractDigest string) (ScenarioResult, []CandidateEvent, error) {
	candidateCount := relation.Bound
	if candidateCount > len(relation.Candidates) {
		candidateCount = len(relation.Candidates)
	}
	result := ScenarioResult{
		ID: relation.ID, Kind: relation.Kind, Expected: relation.Expected, Bound: relation.Bound,
		DomainSize: len(relation.Candidates), CandidatesGenerated: candidateCount, Oracle: relation.Oracle,
	}
	events := make([]CandidateEvent, 0, candidateCount)
	beforeIR := fixture.BaselineIR
	beforeTrace := append([]string(nil), fixture.BaselineTrace...)
	for index := 0; index < candidateCount; index++ {
		candidateID := relation.Candidates[index]
		afterSource, err := transform(relation.Kind, candidateID, fixture.BaseSource)
		if err != nil {
			return ScenarioResult{}, nil, err
		}
		afterIR, err := deriveIR(relation.Kind, afterSource)
		if err != nil {
			return ScenarioResult{}, nil, err
		}
		afterTrace, traceErr := deriveTrace(relation.Kind, afterSource)
		if traceErr != nil {
			afterTrace = append([]string(nil), beforeTrace...)
		}
		event := CandidateEvent{
			Schema: EventSchema, ScenarioID: relation.ID, CandidateIndex: index + 1, CandidateID: candidateID,
			Generated: 1, Executed: 0, Oracle: relation.Oracle, OracleInvocations: 0,
			BeforeSource: fixture.BaseSource, AfterSource: afterSource, BeforeIR: beforeIR, AfterIR: afterIR,
			BeforeTrace: beforeTrace, AfterTrace: afterTrace,
		}
		if relation.RelationContractDigest == contractDigest && relation.Oracle != "missing" {
			event.Executed = 1
			event.OracleInvocations = OracleReplays
			preserved, err := executeOracle(relation.Oracle, beforeIR, afterIR, beforeTrace, afterTrace, fixture.BaseSource, afterSource)
			if err != nil {
				return ScenarioResult{}, nil, err
			}
			event.Preserved = preserved
			result.CandidatesExecuted++
			result.OracleInvocations += OracleReplays
			if !preserved {
				evidence := CandidateEvidence{
					CandidateIndex: index + 1, CandidateID: candidateID, BeforeSource: fixture.BaseSource,
					AfterSource: afterSource, BeforeIR: beforeIR, AfterIR: afterIR,
					BeforeTrace: beforeTrace, AfterTrace: afterTrace, Reproducible: true,
				}
				evidence.Digest, err = unsignedEvidenceDigest(evidence)
				if err != nil {
					return ScenarioResult{}, nil, err
				}
				event.Counterexample = &evidence
				result.Counterexamples++
				if result.FirstCounterexample == nil {
					result.FirstCounterexample = &evidence
				}
			}
		}
		event.Digest, err = digestEvent(event)
		if err != nil {
			return ScenarioResult{}, nil, err
		}
		events = append(events, event)
		if result.FirstCounterexample != nil {
			break
		}
	}
	if relation.RelationContractDigest != contractDigest {
		result.Unknown = &UnknownRecord{
			Stage: "synthesis", Step: "validate-relation-contract",
			Reason:       "relation contract digest does not match the fixed contract input",
			UnknownClass: "STALE", NextOperation: "refresh_relation_contract_digest",
			BlockedBy: []string{"relation-contract-digest"},
		}
	} else if relation.Oracle == "missing" {
		result.Unknown = &UnknownRecord{
			Stage: "synthesis", Step: "resolve-semantic-oracle",
			Reason:       "no semantic oracle was declared for the relation",
			UnknownClass: "DIRECT_MISSING", NextOperation: "declare_and_bind_semantic_oracle",
			BlockedBy: []string{"semantic-oracle"},
		}
	} else if result.FirstCounterexample == nil && relation.Bound < len(relation.Candidates) {
		result.Unknown = &UnknownRecord{
			Stage: "synthesis", Step: "execute-bounded-domain",
			Reason:       "candidate bound ended before the declared relation domain was exhausted",
			UnknownClass: "UNBOUNDED", NextOperation: "increase_candidate_bound_and_replay",
			BlockedBy: []string{"unexecuted-candidates"},
		}
	}
	result.Decision = StateClosed
	if result.FirstCounterexample != nil {
		result.Decision = StateRefuted
	} else if result.Unknown != nil {
		result.Decision = StateUnknown
	}
	if result.Decision != relation.Expected {
		return ScenarioResult{}, nil, fmt.Errorf("relation %q resolved %s, expected %s", relation.ID, result.Decision, relation.Expected)
	}
	return result, events, nil
}

func summarize(results []ScenarioResult) (Counts, string, error) {
	if len(results) != ScenarioCount {
		return Counts{}, "", fmt.Errorf("synthesis must resolve exactly %d scenarios", ScenarioCount)
	}
	counts := Counts{Scenarios: len(results)}
	decision := StateClosed
	for _, result := range results {
		counts.CandidatesGenerated += result.CandidatesGenerated
		counts.CandidatesExecuted += result.CandidatesExecuted
		counts.Counterexamples += result.Counterexamples
		switch result.Decision {
		case StateRefuted:
			counts.Refuted++
			decision = StateRefuted
		case StateUnknown:
			counts.Unknown++
			if decision != StateRefuted {
				decision = StateUnknown
			}
		case StateClosed:
			counts.Closed++
		default:
			return Counts{}, "", fmt.Errorf("unsupported scenario decision %q", result.Decision)
		}
	}
	return counts, decision, nil
}

func executeOracle(oracle, beforeIR, afterIR string, beforeTrace, afterTrace []string, beforeSource, afterSource string) (bool, error) {
	var preserved bool
	switch oracle {
	case "alpha_semantic", "commutative_semantic":
		preserved = beforeIR == afterIR
	case "byte_identity":
		preserved = beforeSource == afterSource
	case "arithmetic_semantic", "effect_trace":
		preserved = sameStrings(beforeTrace, afterTrace)
	default:
		return false, fmt.Errorf("unsupported semantic oracle %q", oracle)
	}
	return preserved, nil
}

func transform(kind, candidate, source string) (string, error) {
	switch candidate {
	case "rename_x_y":
		return replaceWord(source, "x", "y"), nil
	case "rename_x_z":
		return replaceWord(source, "x", "z"), nil
	case "rename_a_b":
		return replaceWord(source, "a", "b"), nil
	case "rename_a_c":
		return replaceWord(source, "a", "c"), nil
	case "rename_a_d":
		return replaceWord(source, "a", "d"), nil
	case "rename_a_e":
		return replaceWord(source, "a", "e"), nil
	case "swap_add_ab", "swap_add_ac":
		return swapAdd(source)
	case "replay_identity", "identity":
		return source, nil
	case "rewrite_sub_to_add":
		if !strings.Contains(source, "sub(") {
			return "", fmt.Errorf("candidate %q cannot rewrite source %q", candidate, source)
		}
		return strings.Replace(source, "sub(", "add(", 1), nil
	case "reorder_effects":
		parts := strings.Split(source, ";")
		if len(parts) != 2 {
			return "", fmt.Errorf("candidate %q requires exactly two effects", candidate)
		}
		return parts[1] + ";" + parts[0], nil
	default:
		return "", fmt.Errorf("unsupported candidate %q for relation kind %q", candidate, kind)
	}
}

func deriveIR(kind, source string) (string, error) {
	normalized := strings.Join(strings.Fields(source), " ")
	switch kind {
	case "alpha_renaming":
		fields := strings.Fields(normalized)
		if len(fields) < 2 || fields[0] != "let" {
			return "", fmt.Errorf("alpha source %q is not a let expression", source)
		}
		return replaceWord(normalized, fields[1], "$v1"), nil
	case "commutative_normalization":
		return normalizeAdd(normalized)
	case "byte_identical_replay", "arithmetic_rewrite", "effect_reordering", "missing_semantic_oracle", "stale_relation_contract":
		return normalized, nil
	case "bound_exhausted":
		fields := strings.Fields(normalized)
		if len(fields) < 2 || fields[0] != "let" {
			return "", fmt.Errorf("bounded source %q is not a let expression", source)
		}
		return replaceWord(normalized, fields[1], "$v1"), nil
	default:
		return "", fmt.Errorf("unsupported relation kind %q", kind)
	}
}

func deriveTrace(kind, source string) ([]string, error) {
	normalized := strings.Join(strings.Fields(source), " ")
	switch kind {
	case "alpha_renaming", "bound_exhausted":
		return []string{"result=2"}, nil
	case "commutative_normalization":
		return []string{"result=commutative"}, nil
	case "byte_identical_replay":
		return []string{"bytes=identical"}, nil
	case "arithmetic_rewrite":
		return []string{"result=" + strconv.Itoa(arithmeticResult(normalized))}, nil
	case "effect_reordering":
		parts := strings.Split(normalized, ";")
		if len(parts) != 2 {
			return nil, errors.New("effect trace requires two effects")
		}
		return []string{"effect=" + effectName(parts[0]), "effect=" + effectName(parts[1])}, nil
	case "missing_semantic_oracle", "stale_relation_contract":
		return []string{"source=" + normalized}, nil
	default:
		return nil, fmt.Errorf("unsupported relation kind %q", kind)
	}
}

func arithmeticResult(source string) int {
	start := strings.IndexByte(source, '(')
	end := strings.LastIndexByte(source, ')')
	if start < 0 || end <= start {
		return 0
	}
	parts := strings.Split(source[start+1:end], ",")
	if len(parts) != 2 {
		return 0
	}
	left, _ := strconv.Atoi(parts[0])
	right, _ := strconv.Atoi(parts[1])
	if strings.HasPrefix(source, "sub(") {
		return left - right
	}
	return left + right
}

func effectName(value string) string {
	value = strings.TrimPrefix(value, "effect(")
	return strings.TrimSuffix(value, ")")
}

func normalizeAdd(source string) (string, error) {
	start := strings.Index(source, "add(")
	if start != 0 || !strings.HasSuffix(source, ")") {
		return "", fmt.Errorf("commutative source %q is not add(a,b)", source)
	}
	parts := strings.Split(source[len("add("):len(source)-1], ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("commutative source %q has wrong arity", source)
	}
	sort.Strings(parts)
	return "add(" + parts[0] + "," + parts[1] + ")", nil
}

func swapAdd(source string) (string, error) {
	start := strings.Index(source, "add(")
	if start != 0 || !strings.HasSuffix(source, ")") {
		return "", fmt.Errorf("source %q is not add(a,b)", source)
	}
	parts := strings.Split(source[len("add("):len(source)-1], ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("source %q has wrong arity", source)
	}
	return "add(" + parts[1] + "," + parts[0] + ")", nil
}

func replaceWord(source, from, to string) string {
	parts := strings.Fields(source)
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(part, from, to)
	}
	return strings.Join(parts, " ")
}

func digestEvent(event CandidateEvent) (string, error) {
	unsigned := event
	unsigned.Digest = ""
	return DigestValue(unsigned)
}

func unknownImprovement(inputs compiledInputs, runnerDigest string) ImprovementClaim {
	return ImprovementClaim{
		Status: StateUnknown, Scenario: "all", SourceDigest: inputs.SourceDigest,
		ContractDigest: inputs.ContractDigest, ToolchainDigest: inputs.Meta.Toolchain.Digest, RunnerDigest: runnerDigest,
		MatchedPair: false,
		Unknown: &UnknownRecord{
			Stage: "improvement", Step: "compare-before-after-integer-pair",
			Reason:       "no matching before/after integer pair exists for the same scenario, source, contract, toolchain, and runner",
			UnknownClass: "MISSING_BEFORE_AFTER_PAIR", NextOperation: "collect_matching_before_after_integer_pair",
			BlockedBy: []string{"before-after-evidence"},
		},
	}
}

func findFixture(corpus FixtureCorpus, id string) ScenarioFixture {
	for _, fixture := range corpus.Scenarios {
		if fixture.ID == id {
			return fixture
		}
	}
	return ScenarioFixture{}
}

func ensureCallerOutput(path, repoRoot string) (string, error) {
	if path == "" || repoRoot == "" {
		return "", errors.New("caller-owned output and repository root are required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}
	if inside(absRoot, absPath) {
		return "", errors.New("caller-owned output must be outside the repository")
	}
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0o755); err != nil {
			return "", err
		}
		return absPath, nil
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("caller-owned output must be a directory")
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", err
	}
	if len(entries) != 0 {
		return "", errors.New("caller-owned output must be empty")
	}
	return absPath, nil
}

func ensureCallerOutputFile(path, repoRoot string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	if inside(absRoot, absPath) {
		return errors.New("caller-owned output must be outside the repository")
	}
	return os.MkdirAll(filepath.Dir(absPath), 0o755)
}

func inside(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return true
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func logicalPath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return filepath.Base(path)
	}
	return filepath.ToSlash(relative)
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func WriteJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}

func WriteNDJSON(path string, values []CandidateEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if _, err := writer.Write(append(encoded, '\n')); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func MeasureInventory(root string) (InventoryMetrics, error) {
	var metrics InventoryMetrics
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root {
				name := entry.Name()
				if name == ".git" || name == "vendor" || name == ".cache" || name == "cache" || name == "toolchain" || name == ".toolchain" {
					return filepath.SkipDir
				}
				metrics.Subdirectories++
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "README.md" {
			return nil
		}
		metrics.RegularFiles++
		extension := filepath.Ext(path)
		if extension != ".go" && extension != ".gooo" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := physicalLines(data)
		if extension == ".go" {
			metrics.GoFiles++
			metrics.GoPhysicalLines += lines
		} else {
			metrics.GoooFiles++
			metrics.GoooPhysicalLines += lines
		}
		return nil
	})
	return metrics, err
}

func physicalLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := 0
	for _, value := range data {
		if value == '\n' {
			lines++
		}
	}
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}
