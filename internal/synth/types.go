package synth

import "errors"

const (
	MetaSchema       = "gooo.metamorphic_counterexample_synthesizer/v1"
	ContractSchema   = "gooo/metamorphic-counterexample-synthesizer/contract/v1"
	FixtureSchema    = "gooo/metamorphic-counterexample-synthesizer/fixtures/v1"
	IRSchema         = "gooo/metamorphic-counterexample-synthesizer/ir/v1"
	ManifestSchema   = "gooo/metamorphic-counterexample-synthesizer/synthesis-manifest/v1"
	ReceiptSchema    = "gooo/metamorphic-counterexample-synthesizer/counterexample-receipt/v1"
	EventSchema      = "gooo/metamorphic-counterexample-synthesizer/candidate-event/v1"
	ReportSchema     = "gooo/metamorphic-counterexample-synthesizer/report/v1"
	ScenarioCount    = 8
	OracleReplays    = 2
	StateClosed      = "CLOSED"
	StateUnknown     = "UNKNOWN"
	StateRefuted     = "REFUTED"
	ToolchainVersion = "1.27"
)

var requiredUnknownFields = []string{
	"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by",
}

var requiredActivities = []string{
	"DeclareMetamorphicRelation",
	"BoundCandidateDomain",
	"GenerateCandidate",
	"ExecuteSemanticOracle",
	"PreserveCounterexample",
	"CloseBoundedClaim",
	"EmitSynthesisReceipt",
}

var requiredScenarioIDs = []string{
	"alpha-renaming",
	"commutative-normalization",
	"byte-identical-replay",
	"arithmetic-rewrite",
	"effect-reordering",
	"bound-exhausted",
	"missing-semantic-oracle",
	"stale-relation-contract",
}

type Authority struct {
	RepositoryWrites      int  `json:"repository_writes"`
	CommitAuthority       int  `json:"commit_authority"`
	PushAuthority         int  `json:"push_authority"`
	MergeAuthority        int  `json:"merge_authority"`
	ReleaseAuthority      int  `json:"release_authority"`
	LocalTestExecutions   int  `json:"local_test_executions"`
	CallerOwnedOutputOnly bool `json:"caller_owned_output_only"`
}

func zeroAuthority() Authority {
	return Authority{CallerOwnedOutputOnly: true}
}

type Activity struct {
	Ordinal int    `json:"ordinal"`
	ID      string `json:"id"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Edge    string `json:"edge"`
}

type Relation struct {
	Ordinal                int      `json:"ordinal"`
	ID                     string   `json:"id"`
	Kind                   string   `json:"kind"`
	Candidates             []string `json:"candidates"`
	Bound                  int      `json:"bound"`
	Oracle                 string   `json:"oracle"`
	Expected               string   `json:"expected"`
	Fixture                string   `json:"fixture"`
	RelationContractDigest string   `json:"relation_contract_digest"`
}

type MetaContract struct {
	Schema            string       `json:"schema"`
	Authority         string       `json:"authority"`
	AuthorityBoundary Authority    `json:"authority_boundary"`
	Denominator       Denominator  `json:"denominator"`
	Statuses          []string     `json:"statuses"`
	Precedence        []string     `json:"precedence"`
	UnknownFields     []string     `json:"unknown_fields"`
	Toolchain         Toolchain    `json:"toolchain"`
	RunnerPolicy      RunnerPolicy `json:"runner_policy"`
	Activities        []Activity   `json:"activities"`
	Relations         []Relation   `json:"relations"`
	SourceDigest      string       `json:"source_digest"`
}

type Denominator struct {
	ID        string `json:"id"`
	Scenarios int    `json:"scenarios"`
	Unit      string `json:"unit"`
}

type Toolchain struct {
	Go     string `json:"go"`
	Digest string `json:"digest"`
}

type RunnerPolicy struct {
	EvaluatorGeneratorRuntimeOnly bool `json:"evaluator_generator_runtime_only"`
}

type Contract struct {
	Schema        string             `json:"schema"`
	ID            string             `json:"id"`
	Version       string             `json:"version"`
	ScenarioCount int                `json:"scenario_count"`
	Fixed         bool               `json:"fixed"`
	Statuses      []string           `json:"statuses"`
	Precedence    []string           `json:"precedence"`
	Unit          string             `json:"unit"`
	Activities    []Activity         `json:"activities"`
	Scenarios     []ContractScenario `json:"scenarios"`
}

type ContractScenario struct {
	Ordinal        int    `json:"ordinal"`
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	CandidateCount int    `json:"candidate_count"`
	Bound          int    `json:"bound"`
	Expected       string `json:"expected"`
}

type FixtureCorpus struct {
	Schema    string            `json:"schema"`
	Version   string            `json:"version"`
	Scenarios []ScenarioFixture `json:"scenarios"`
}

type ScenarioFixture struct {
	ID            string   `json:"id"`
	BaseSource    string   `json:"base_source"`
	BaselineIR    string   `json:"baseline_ir"`
	BaselineTrace []string `json:"baseline_trace"`
}

type SemanticIR struct {
	Schema         string     `json:"schema"`
	Version        string     `json:"version"`
	SourceDigest   string     `json:"source_digest"`
	ContractDigest string     `json:"contract_digest"`
	Toolchain      Toolchain  `json:"toolchain"`
	Authority      Authority  `json:"authority"`
	Precedence     []string   `json:"precedence"`
	UnknownFields  []string   `json:"unknown_fields"`
	Activities     []Activity `json:"activities"`
	Relations      []Relation `json:"relations"`
	IRDigest       string     `json:"ir_digest,omitempty"`
}

type UnknownRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

func (u *UnknownRecord) Valid() bool {
	return u != nil && u.Stage != "" && u.Step != "" && u.Reason != "" &&
		u.UnknownClass != "" && u.NextOperation != "" && len(u.BlockedBy) > 0
}

type CandidateEvidence struct {
	CandidateIndex int      `json:"candidate_index"`
	CandidateID    string   `json:"candidate_id"`
	BeforeSource   string   `json:"before_source"`
	AfterSource    string   `json:"after_source"`
	BeforeIR       string   `json:"before_ir"`
	AfterIR        string   `json:"after_ir"`
	BeforeTrace    []string `json:"before_trace"`
	AfterTrace     []string `json:"after_trace"`
	Digest         string   `json:"digest"`
	Reproducible   bool     `json:"reproducible"`
}

type CandidateEvent struct {
	Schema            string             `json:"schema"`
	ScenarioID        string             `json:"scenario_id"`
	CandidateIndex    int                `json:"candidate_index"`
	CandidateID       string             `json:"candidate_id"`
	Generated         int                `json:"generated"`
	Executed          int                `json:"executed"`
	Oracle            string             `json:"oracle"`
	OracleInvocations int                `json:"oracle_invocations"`
	BeforeSource      string             `json:"before_source"`
	AfterSource       string             `json:"after_source"`
	BeforeIR          string             `json:"before_ir"`
	AfterIR           string             `json:"after_ir"`
	BeforeTrace       []string           `json:"before_trace"`
	AfterTrace        []string           `json:"after_trace"`
	Preserved         bool               `json:"preserved"`
	Counterexample    *CandidateEvidence `json:"counterexample,omitempty"`
	Digest            string             `json:"digest"`
}

type ScenarioResult struct {
	ID                  string             `json:"id"`
	Kind                string             `json:"kind"`
	Decision            string             `json:"decision"`
	Expected            string             `json:"expected"`
	Bound               int                `json:"bound"`
	DomainSize          int                `json:"domain_size"`
	CandidatesGenerated int                `json:"candidates_generated"`
	CandidatesExecuted  int                `json:"candidates_executed"`
	Counterexamples     int                `json:"counterexamples"`
	Oracle              string             `json:"oracle"`
	OracleInvocations   int                `json:"oracle_invocations"`
	FirstCounterexample *CandidateEvidence `json:"first_counterexample,omitempty"`
	Unknown             *UnknownRecord     `json:"unknown,omitempty"`
}

type Counts struct {
	Scenarios           int `json:"scenarios"`
	Closed              int `json:"closed"`
	Unknown             int `json:"unknown"`
	Refuted             int `json:"refuted"`
	CandidatesGenerated int `json:"candidates_generated"`
	CandidatesExecuted  int `json:"candidates_executed"`
	Counterexamples     int `json:"counterexamples"`
}

type InventoryMetrics struct {
	RegularFiles      int `json:"regular_files"`
	Subdirectories    int `json:"subdirectories"`
	GoFiles           int `json:"go_files"`
	GoooFiles         int `json:"gooo_files"`
	GoPhysicalLines   int `json:"go_physical_lines"`
	GoooPhysicalLines int `json:"gooo_physical_lines"`
}

type RuntimeMetric struct {
	WallMS     int64 `json:"wall_ms"`
	PeakRSSKiB int64 `json:"peak_rss_kib"`
}

type RuntimeMeasurements struct {
	Build       RuntimeMetric `json:"build"`
	Vet         RuntimeMetric `json:"vet"`
	Test        RuntimeMetric `json:"test"`
	Conformance RuntimeMetric `json:"conformance"`
	Integration RuntimeMetric `json:"integration"`
}

type TestCounts struct {
	Total    int `json:"total"`
	Executed int `json:"executed"`
	Reused   int `json:"reused"`
	Failed   int `json:"failed"`
	Unknown  int `json:"unknown"`
}

type ImprovementClaim struct {
	Status          string         `json:"status"`
	Scenario        string         `json:"scenario"`
	SourceDigest    string         `json:"source_digest"`
	ContractDigest  string         `json:"contract_digest"`
	ToolchainDigest string         `json:"toolchain_digest"`
	RunnerDigest    string         `json:"runner_digest"`
	MatchedPair     bool           `json:"matched_pair"`
	Unknown         *UnknownRecord `json:"unknown,omitempty"`
}

type SynthesisManifest struct {
	Schema         string              `json:"schema"`
	Version        string              `json:"version"`
	Decision       string              `json:"decision"`
	Authority      Authority           `json:"authority"`
	Precedence     []string            `json:"precedence"`
	Source         string              `json:"source"`
	Fixture        string              `json:"fixture"`
	Contract       string              `json:"contract"`
	SubjectSHA     string              `json:"subject_sha"`
	Toolchain      Toolchain           `json:"toolchain"`
	RunnerDigest   string              `json:"runner_digest"`
	SourceDigest   string              `json:"source_digest"`
	ContractDigest string              `json:"contract_digest"`
	IRDigest       string              `json:"ir_digest"`
	Denominator    Denominator         `json:"denominator"`
	Activities     []Activity          `json:"activities"`
	Relations      []Relation          `json:"relations"`
	Scenarios      []ScenarioResult    `json:"scenarios"`
	Counts         Counts              `json:"counts"`
	Inventory      InventoryMetrics    `json:"inventory"`
	Runtime        RuntimeMeasurements `json:"runtime"`
	Tests          TestCounts          `json:"tests"`
	Improvement    ImprovementClaim    `json:"improvement"`
	Outputs        []string            `json:"outputs"`
}

type CounterexampleReceipt struct {
	Schema         string           `json:"schema"`
	Version        string           `json:"version"`
	Decision       string           `json:"decision"`
	Authority      Authority        `json:"authority"`
	Precedence     []string         `json:"precedence"`
	SourceDigest   string           `json:"source_digest"`
	ContractDigest string           `json:"contract_digest"`
	IRDigest       string           `json:"ir_digest"`
	Toolchain      Toolchain        `json:"toolchain"`
	RunnerDigest   string           `json:"runner_digest"`
	Scenarios      []ScenarioResult `json:"scenarios"`
	Counts         Counts           `json:"counts"`
	Improvement    ImprovementClaim `json:"improvement"`
}

func validateUnknownFields(fields []string) error {
	if !sameStrings(fields, requiredUnknownFields) {
		return errors.New("UNKNOWN must preserve the required six fields in order")
	}
	return nil
}

func sameStrings(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}
