package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-metamorphic-counterexample-synthesizer/internal/synth"
)

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("command is required: synthesize or conformance"))
	}
	switch os.Args[1] {
	case "synthesize":
		runSynthesize(os.Args[2:])
	case "conformance":
		runConformance(os.Args[2:])
	default:
		fail(fmt.Errorf("unknown command %q", os.Args[1]))
	}
}

func runSynthesize(args []string) {
	flags := flag.NewFlagSet("synthesize", flag.ContinueOnError)
	options := synth.Options{}
	flags.StringVar(&options.Source, "source", ".gooo/metamorphic-counterexample-synthesizer.gooo", "authoritative .gooo source")
	flags.StringVar(&options.Contract, "contract", "contracts/metamorphic-counterexample-synthesizer-v1.json", "fixed relation contract")
	flags.StringVar(&options.Fixtures, "fixtures", "fixtures/scenarios.json", "fixed scenario fixtures")
	flags.StringVar(&options.RepoRoot, "repo-root", ".", "input repository root")
	flags.StringVar(&options.Output, "out", "", "empty caller-owned output directory")
	flags.StringVar(&options.SubjectSHA, "subject-sha", "", "subject revision")
	flags.StringVar(&options.ToolchainVersion, "toolchain-version", "", "observed CI Go version")
	flags.StringVar(&options.RunnerDigest, "runner-digest", "", "digest of the CI runner binary")
	if err := flags.Parse(args); err != nil {
		fail(err)
	}
	if err := synth.Synthesize(options); err != nil {
		fail(err)
	}
}

func runConformance(args []string) {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	source := flags.String("source", ".gooo/metamorphic-counterexample-synthesizer.gooo", "authoritative .gooo source")
	contract := flags.String("contract", "contracts/metamorphic-counterexample-synthesizer-v1.json", "fixed relation contract")
	fixtures := flags.String("fixtures", "fixtures/scenarios.json", "fixed scenario fixtures")
	repoRoot := flags.String("repo-root", ".", "input repository root")
	output := flags.String("output", "", "caller-owned conformance output")
	if err := flags.Parse(args); err != nil {
		fail(err)
	}
	if err := synth.RunConformance(*source, *contract, *fixtures, *repoRoot, *output); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
