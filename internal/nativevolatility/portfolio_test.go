package nativevolatility

import "testing"

func TestPortfolioLearnsDiagnosticProducerAndAppliesCurrentHashes(t *testing.T) {
	context := portfolioTestContext()
	learning := Analyze(
		observation(entry("stable.bin", "1", ":stable"), entry("generated.jar", "2", ":generated:jar")),
		observation(entry("stable.bin", "1", ":stable"), entry("generated.jar", "3", ":generated:jar")),
	)
	portfolio, err := LearnPortfolio(Portfolio{}, context, digest("a"), learning, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(portfolio.QuarantinedProducers) != 1 || portfolio.Observations[0].PerformanceClaimed {
		t.Fatalf("unexpected diagnostic portfolio: %+v", portfolio)
	}

	current := Analyze(
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{
			entry("stable.bin", "4", ":stable"), entry("generated.jar", "5", ":generated:jar"),
		}},
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{
			entry("stable.bin", "4", ":stable"), entry("generated.jar", "5", ":generated:jar"),
		}},
	)
	application, err := ApplyPortfolio(portfolio, context, digest("c"), current)
	if err != nil {
		t.Fatal(err)
	}
	if len(application.QuarantinedOutputs) != 1 || len(application.TransportedOutputs) != 1 ||
		application.QuarantinedOutputs[0].SHA256 != digest("5") ||
		application.TransportedOutputs[0].SHA256 != digest("4") {
		t.Fatalf("application reused historical hashes or partitioned incorrectly: %+v", application)
	}
	reused := Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: application.TransportedOutputs}
	rebuilt := Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: application.QuarantinedOutputs}
	if err := VerifyPortfolioCandidate(application, portfolio, current, reused, rebuilt); err != nil {
		t.Fatal(err)
	}
}

func TestPortfolioAccumulatesOnlyAuthoritativeCompatibleEvidence(t *testing.T) {
	context := portfolioTestContext()
	first := Analyze(
		observation(entry("a", "1", ":a"), entry("b", "2", ":b")),
		observation(entry("a", "3", ":a"), entry("b", "2", ":b")),
	)
	portfolio, err := LearnPortfolio(Portfolio{}, context, digest("a"), first, true)
	if err != nil {
		t.Fatal(err)
	}
	second := Analyze(
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{
			entry("a", "4", ":a"), entry("b", "5", ":b"), entry("c", "6", ":c"),
		}},
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{
			entry("a", "4", ":a"), entry("b", "7", ":b"), entry("c", "6", ":c"),
		}},
	)
	portfolio, err = LearnPortfolio(portfolio, context, digest("b"), second, true)
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(portfolio.QuarantinedProducers, []string{":a", ":b"}) {
		t.Fatalf("portfolio union = %v", portfolio.QuarantinedProducers)
	}
	if _, err := LearnPortfolio(portfolio, context, digest("c"), second, false); err == nil {
		t.Fatal("non-authoritative learning was accepted")
	}
	drifted := context
	drifted.OutputContractSHA256 = digest("9")
	if _, err := LearnPortfolio(portfolio, drifted, digest("c"), second, true); err == nil {
		t.Fatal("context drift was accepted")
	}
	if _, err := LearnPortfolio(portfolio, context, digest("a"), second, true); err == nil {
		t.Fatal("different evidence for one revision was accepted")
	}
}

func TestPortfolioApplicationFailsClosedOnMissingProducerAndTampering(t *testing.T) {
	context := portfolioTestContext()
	learning := Analyze(
		observation(entry("volatile", "1", ":old"), entry("stable", "2", ":stable")),
		observation(entry("volatile", "3", ":old"), entry("stable", "2", ":stable")),
	)
	portfolio, err := LearnPortfolio(Portfolio{}, context, digest("a"), learning, true)
	if err != nil {
		t.Fatal(err)
	}
	current := Analyze(
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{entry("stable", "4", ":stable")}},
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{entry("stable", "4", ":stable")}},
	)
	if _, err := ApplyPortfolio(portfolio, context, digest("c"), current); err == nil {
		t.Fatal("missing historical producer was accepted")
	}

	current = Analyze(
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{
			entry("volatile", "4", ":old"), entry("stable", "5", ":stable"),
		}},
		Observation{SchemaVersion: ObservationSchema, BindingSHA256: digest("b"), Entries: []Entry{
			entry("volatile", "4", ":old"), entry("stable", "5", ":stable"),
		}},
	)
	application, err := ApplyPortfolio(portfolio, context, digest("c"), current)
	if err != nil {
		t.Fatal(err)
	}
	tampered := application
	tampered.TransportedOutputs = append([]Entry(nil), application.TransportedOutputs...)
	tampered.TransportedOutputs[0].SHA256 = digest("9")
	if err := ValidatePortfolioApplication(tampered, portfolio, current); err == nil {
		t.Fatal("tampered current transport was accepted")
	}
}

func TestPortfolioPreflightAuthorizesOnlyExactContext(t *testing.T) {
	context := portfolioTestContext()
	learning := Analyze(
		observation(entry("stable", "1", ":stable"), entry("volatile", "2", ":volatile")),
		observation(entry("stable", "1", ":stable"), entry("volatile", "3", ":volatile")),
	)
	portfolio, err := LearnPortfolio(Portfolio{}, context, digest("a"), learning, true)
	if err != nil {
		t.Fatal(err)
	}
	compatible, err := PreflightPortfolio(portfolio, context, context)
	if err != nil {
		t.Fatal(err)
	}
	if compatible.Decision != PortfolioDecisionCompatible ||
		!compatible.IndependentNativeObservationAuthorized || len(compatible.DriftedBindings) != 0 {
		t.Fatalf("compatible preflight = %+v", compatible)
	}

	driftedContext := context
	driftedContext.WrapperSHA256 = digest("8")
	driftedContext.OutputContractSHA256 = digest("9")
	retained, err := PreflightPortfolio(portfolio, context, driftedContext)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Decision != PortfolioDecisionRetained ||
		retained.IndependentNativeObservationAuthorized ||
		!equalStrings(retained.DriftedBindings, []string{
			"GRADLE_WRAPPER_SHA256", "OUTPUT_CONTRACT_SHA256",
		}) {
		t.Fatalf("drifted preflight = %+v", retained)
	}

	tampered := retained
	tampered.IndependentNativeObservationAuthorized = true
	if err := ValidatePortfolioPreflight(tampered, portfolio); err == nil {
		t.Fatal("drifted preflight authorized an independent native observation")
	}
	wrongLearned := context
	wrongLearned.RepositoryScopeSHA256 = digest("0")
	if _, err := PreflightPortfolio(portfolio, wrongLearned, context); err == nil {
		t.Fatal("portfolio accepted a mismatched learned context")
	}
}

func portfolioTestContext() PortfolioContext {
	return PortfolioContext{
		RepositoryScopeSHA256: digest("1"), WorkflowSHA256: digest("2"),
		WrapperSHA256: digest("3"), OutputContractSHA256: digest("4"),
	}
}
