package synth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadMeta(path string) (MetaContract, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MetaContract{}, nil, err
	}
	meta, err := parseMeta(string(raw))
	if err != nil {
		return MetaContract{}, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	meta.SourceDigest = DigestBytes(raw)
	if err := meta.Validate(); err != nil {
		return MetaContract{}, nil, err
	}
	return meta, raw, nil
}

func parseMeta(input string) (MetaContract, error) {
	var meta MetaContract
	for lineNumber, rawLine := range strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens, err := tokenize(line)
		if err != nil {
			return MetaContract{}, fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
		if len(tokens) == 0 {
			continue
		}
		switch tokens[0] {
		case "gooo":
			if len(tokens) != 3 || tokens[1] != "metamorphic_counterexample_synthesizer" || tokens[2] != "v1" {
				return MetaContract{}, fmt.Errorf("line %d: invalid header", lineNumber+1)
			}
			meta.Schema = MetaSchema
		case "authority":
			if len(tokens) == 2 {
				meta.Authority = tokens[1]
				continue
			}
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.AuthorityBoundary = Authority{
				RepositoryWrites:      parseInt(values, "repository_writes"),
				CommitAuthority:       parseInt(values, "commit_authority"),
				PushAuthority:         parseInt(values, "push_authority"),
				MergeAuthority:        parseInt(values, "merge_authority"),
				ReleaseAuthority:      parseInt(values, "release_authority"),
				LocalTestExecutions:   parseInt(values, "local_test_executions"),
				CallerOwnedOutputOnly: parseBool(values, "caller_owned_output_only"),
			}
		case "denominator":
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.Denominator = Denominator{ID: values["id"], Scenarios: parseInt(values, "scenarios"), Unit: values["unit"]}
		case "decision":
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.Statuses = splitCSV(values["statuses"])
		case "precedence":
			if len(tokens) != 2 {
				return MetaContract{}, fmt.Errorf("line %d: invalid precedence", lineNumber+1)
			}
			meta.Precedence = strings.Split(tokens[1], ">")
		case "unknown_fields":
			if len(tokens) != 2 {
				return MetaContract{}, fmt.Errorf("line %d: invalid unknown_fields", lineNumber+1)
			}
			meta.UnknownFields = strings.Split(tokens[1], ",")
		case "toolchain":
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.Toolchain = Toolchain{Go: values["go"], Digest: values["digest"]}
		case "runner_policy":
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.RunnerPolicy = RunnerPolicy{EvaluatorGeneratorRuntimeOnly: parseBool(values, "evaluator_generator_runtime_only")}
		case "activity":
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.Activities = append(meta.Activities, Activity{
				Ordinal: parseInt(values, "ordinal"), ID: values["id"], Input: values["input"], Output: values["output"], Edge: values["edge"],
			})
		case "relation":
			values, err := keyValues(tokens[1:])
			if err != nil {
				return MetaContract{}, lineError(lineNumber, err)
			}
			meta.Relations = append(meta.Relations, Relation{
				Ordinal: parseInt(values, "ordinal"), ID: values["id"], Kind: values["kind"],
				Candidates: splitPipe(values["candidates"]), Bound: parseInt(values, "bound"),
				Oracle: values["oracle"], Expected: values["expected"], Fixture: values["fixture"],
				RelationContractDigest: values["relation_contract_digest"],
			})
		default:
			return MetaContract{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber+1, tokens[0])
		}
	}
	return meta, nil
}

func (meta MetaContract) Validate() error {
	if meta.Schema != MetaSchema || meta.Authority != "metacode" {
		return errors.New(".gooo must declare metamorphic synthesizer metacode authority")
	}
	if meta.AuthorityBoundary != zeroAuthority() {
		return errors.New("product repository and local test authorities must all be zero")
	}
	if meta.Denominator.ID != "metamorphic-counterexample-synthesizer-v1" || meta.Denominator.Scenarios != ScenarioCount || meta.Denominator.Unit != "metamorphic_relation" {
		return errors.New("denominator must declare exactly eight metamorphic relations")
	}
	if !sameStrings(meta.Statuses, []string{StateClosed, StateUnknown, StateRefuted}) || !sameStrings(meta.Precedence, []string{StateRefuted, StateUnknown, StateClosed}) {
		return errors.New("statuses or precedence are not declared exactly")
	}
	if err := validateUnknownFields(meta.UnknownFields); err != nil {
		return err
	}
	if meta.Toolchain.Go != ToolchainVersion || meta.Toolchain.Digest != "sha256:go1.27" || !meta.RunnerPolicy.EvaluatorGeneratorRuntimeOnly {
		return errors.New("toolchain or evaluator boundary is incomplete")
	}
	if len(meta.Activities) != len(requiredActivities) {
		return fmt.Errorf("expected exactly %d declared activities", len(requiredActivities))
	}
	for i, activity := range meta.Activities {
		if activity.Ordinal != i+1 || activity.ID != requiredActivities[i] || activity.Input == "" || activity.Output == "" || activity.Edge == "" {
			return fmt.Errorf("activity %d is not the fixed metacode activity", i+1)
		}
	}
	if len(meta.Relations) != ScenarioCount {
		return fmt.Errorf("expected exactly %d declared relations", ScenarioCount)
	}
	seen := map[string]bool{}
	for i, relation := range meta.Relations {
		if relation.Ordinal != i+1 || relation.ID != requiredScenarioIDs[i] || seen[relation.ID] || relation.Kind == "" || relation.Bound <= 0 || len(relation.Candidates) == 0 || relation.Oracle == "" || relation.Expected == "" || relation.Fixture == "" || relation.RelationContractDigest == "" {
			return fmt.Errorf("relation %d is incomplete or not fixed", i+1)
		}
		if relation.Expected != StateClosed && relation.Expected != StateUnknown && relation.Expected != StateRefuted {
			return fmt.Errorf("relation %q has unsupported expected state", relation.ID)
		}
		seen[relation.ID] = true
	}
	return nil
}

func LoadContract(path string) (Contract, []byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, nil, "", err
	}
	var contract Contract
	if err := json.Unmarshal(raw, &contract); err != nil {
		return Contract{}, nil, "", err
	}
	if err := contract.Validate(); err != nil {
		return Contract{}, nil, "", err
	}
	return contract, raw, DigestBytes(raw), nil
}

func (contract Contract) Validate() error {
	if contract.Schema != ContractSchema || contract.ID != "metamorphic-counterexample-synthesizer-v1" || contract.Version != "v1" || !contract.Fixed || contract.ScenarioCount != ScenarioCount || contract.Unit != "metamorphic_relation" {
		return errors.New("contract header is not fixed for v1")
	}
	if !sameStrings(contract.Statuses, []string{StateClosed, StateUnknown, StateRefuted}) || !sameStrings(contract.Precedence, []string{StateRefuted, StateUnknown, StateClosed}) {
		return errors.New("contract statuses or precedence are not fixed")
	}
	if len(contract.Activities) != len(requiredActivities) {
		return errors.New("contract activity denominator is not seven")
	}
	for i, activity := range contract.Activities {
		if activity.Ordinal != i+1 || activity.ID != requiredActivities[i] || activity.Input == "" || activity.Output == "" || activity.Edge == "" {
			return fmt.Errorf("contract activity %d is invalid", i+1)
		}
	}
	if len(contract.Scenarios) != ScenarioCount {
		return errors.New("contract scenario denominator is not eight")
	}
	for i, scenario := range contract.Scenarios {
		if scenario.Ordinal != i+1 || scenario.ID != requiredScenarioIDs[i] || scenario.Kind == "" || scenario.CandidateCount <= 0 || scenario.Bound <= 0 || scenario.Expected == "" {
			return fmt.Errorf("contract scenario %d is invalid", i+1)
		}
	}
	return nil
}

func ValidateMetaAgainstContract(meta MetaContract, contract Contract, contractDigest string) error {
	if meta.Denominator.ID != contract.ID || !sameStrings(meta.Precedence, contract.Precedence) {
		return errors.New(".gooo and JSON contract do not share the same denominator and precedence")
	}
	for i, sourceRelation := range meta.Relations {
		contractRelation := contract.Scenarios[i]
		if sourceRelation.ID != contractRelation.ID || sourceRelation.Kind != contractRelation.Kind || len(sourceRelation.Candidates) != contractRelation.CandidateCount || sourceRelation.Bound != contractRelation.Bound || sourceRelation.Expected != contractRelation.Expected {
			return fmt.Errorf("relation %q does not match its fixed contract row", sourceRelation.ID)
		}
		if sourceRelation.RelationContractDigest != contractDigest && sourceRelation.ID != "stale-relation-contract" {
			return fmt.Errorf("relation %q has a stale contract digest", sourceRelation.ID)
		}
	}
	return nil
}

func LoadFixtures(path string) (FixtureCorpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return FixtureCorpus{}, err
	}
	var corpus FixtureCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		return FixtureCorpus{}, err
	}
	if corpus.Schema != FixtureSchema || corpus.Version != "v1" || len(corpus.Scenarios) != ScenarioCount {
		return FixtureCorpus{}, errors.New("fixture corpus must contain exactly eight v1 scenarios")
	}
	for i, scenario := range corpus.Scenarios {
		if scenario.ID != requiredScenarioIDs[i] || scenario.BaseSource == "" || scenario.BaselineIR == "" || len(scenario.BaselineTrace) == 0 {
			return FixtureCorpus{}, fmt.Errorf("fixture %d is incomplete or out of order", i+1)
		}
	}
	return corpus, nil
}

func tokenize(line string) ([]string, error) {
	var tokens []string
	var builder strings.Builder
	inQuote := false
	escaped := false
	flush := func() {
		if builder.Len() > 0 {
			tokens = append(tokens, builder.String())
			builder.Reset()
		}
	}
	for _, character := range line {
		if escaped {
			builder.WriteRune(character)
			escaped = false
			continue
		}
		if character == '\\' && inQuote {
			escaped = true
			continue
		}
		if character == '"' {
			inQuote = !inQuote
			continue
		}
		if character == ' ' || character == '\t' {
			if inQuote {
				builder.WriteRune(character)
			} else {
				flush()
			}
			continue
		}
		builder.WriteRune(character)
	}
	if escaped || inQuote {
		return nil, errors.New("unterminated quoted value")
	}
	flush()
	return tokens, nil
}

func keyValues(tokens []string) (map[string]string, error) {
	values := make(map[string]string, len(tokens))
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("expected key=value, got %q", token)
		}
		values[parts[0]] = parts[1]
	}
	return values, nil
}

func parseInt(values map[string]string, key string) int {
	value, _ := strconv.Atoi(values[key])
	return value
}

func parseBool(values map[string]string, key string) bool {
	value, _ := strconv.ParseBool(values[key])
	return value
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func splitPipe(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "|")
}

func lineError(lineNumber int, err error) error {
	return fmt.Errorf("line %d: %w", lineNumber+1, err)
}
