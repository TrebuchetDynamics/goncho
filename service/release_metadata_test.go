package goncho

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestChangelogReleaseHeadingsHaveMatchingTags(t *testing.T) {
	if _, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err != nil {
		t.Skipf("git checkout unavailable: %v", err)
	}

	changelog, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatalf("ReadFile CHANGELOG.md: %v", err)
	}

	releaseHeading := regexp.MustCompile(`(?m)^## (v\d+\.\d+\.\d+) - `)
	matches := releaseHeading.FindAllStringSubmatch(string(changelog), -1)
	if len(matches) == 0 {
		t.Fatal("CHANGELOG.md has no tagged release headings")
	}

	var missing []string
	for _, match := range matches {
		version := match[1]
		out, err := exec.Command("git", "tag", "-l", version).Output()
		if err != nil {
			t.Fatalf("git tag -l %s: %v", version, err)
		}
		if strings.TrimSpace(string(out)) != version {
			missing = append(missing, version)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("CHANGELOG.md release headings without matching git tags: %s", strings.Join(missing, ", "))
	}
}

func TestReleaseSmokeDocsMentionMetadataGuard(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/start/quick-start.md",
		"docs-site/src/content/docs/operators/runbook.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			if !strings.Contains(text, "make release-smoke") {
				t.Fatalf("%s does not mention make release-smoke", path)
			}
			if !strings.Contains(strings.ToLower(text), "release metadata") {
				t.Fatalf("%s does not mention release metadata checks in release-smoke guidance", path)
			}
		})
	}
}

func TestStableE2EBenchmarkSmokeTargetIsDocumented(t *testing.T) {
	makefileRaw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	makefile := string(makefileRaw)
	start := strings.Index(makefile, "stable-e2e-bench-smoke:\n")
	if start < 0 {
		t.Fatal("Makefile missing stable-e2e-bench-smoke target")
	}
	end := strings.Index(makefile[start:], "\nrelease-metadata-smoke:")
	if end < 0 {
		t.Fatal("Makefile stable-e2e-bench-smoke target is not before release-metadata-smoke")
	}
	target := makefile[start : start+end]
	for _, want := range []string{
		"$(MAKE) install-smoke",
		"go test ./...",
		"go vet ./...",
		"mktemp -d",
		"tiny-longmemeval.jsonl",
		"--locomo-memories $(LOCOMO_SMOKE_MEMORIES)",
		"--beam-convert-in $(BEAM_SMOKE_RAW)",
		"--beam-paired-compare",
	} {
		if !strings.Contains(target, want) {
			t.Fatalf("Makefile stable e2e benchmark smoke target missing marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"docs/benchmarks/results",
		"docs/benchmarks/failures",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("Makefile stable e2e benchmark smoke target writes tracked benchmark path %q", forbidden)
		}
	}

	readmeRaw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	readme := string(readmeRaw)
	for _, want := range []string{
		"make stable-e2e-bench-smoke",
		"go test ./...",
		"go vet ./...",
		"install-smoke",
		"temporary outputs",
		"bench-longmemeval-s-smoke",
		"bench-locomo-smoke",
		"bench-beam-smoke",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md stable e2e benchmark smoke guidance missing marker %q", want)
		}
	}
}

func TestPublicDocsLinkGoReference(t *testing.T) {
	const goReferenceURL = "https://pkg.go.dev/github.com/TrebuchetDynamics/goncho"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), goReferenceURL) {
				t.Fatalf("%s does not link the public Go reference at %s", path, goReferenceURL)
			}
		})
	}
}

func TestPublicDocsMentionEcosystemSmoke(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/operators/runbook.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "make ecosystem-smoke") {
				t.Fatalf("%s does not mention make ecosystem-smoke", path)
			}
		})
	}
}

func TestPublicDocsLinkRetrievalBenchmarksReference(t *testing.T) {
	wantByPath := map[string][]string{
		"README.md": {
			"Retrieval Benchmarks",
			"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
		},
		"docs-site/src/content/docs/index.md": {
			"Retrieval Benchmarks",
			"/reference/retrieval-benchmarks/",
		},
		"docs-site/src/content/docs/start/current-capabilities.md": {
			"Retrieval Benchmarks",
			"/reference/retrieval-benchmarks/",
		},
		"docs-site/src/content/docs/start/quick-start.md": {
			"Retrieval Benchmarks",
			"/reference/retrieval-benchmarks/",
		},
	}
	for path, wants := range wantByPath {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, want := range wants {
				if !strings.Contains(text, want) {
					t.Fatalf("%s does not link benchmark methodology reference marker %q", path, want)
				}
			}
		})
	}
}

func TestPublicDocsSurfaceExternalAdapterContract(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, want := range []string{"external adapter contract", "agentmemory PR #583"} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s does not surface external adapter credibility marker %q", path, want)
				}
			}
		})
	}
}

func TestAgentMemoryBenchmarkDocsUseUpstreamSourceURL(t *testing.T) {
	const upstreamURL = "https://github.com/rohitg00/agentmemory"
	for _, path := range []string{
		"scripts/bench_agentmemory_locomo.py",
		"cmd/goncho-bench/locomo_backend_comparison.go",
		"docs/benchmarks/external-backend-adapters.md",
		"docs-site/src/content/docs/operators/runbook.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			if !strings.Contains(text, upstreamURL) {
				t.Fatalf("%s does not point agentmemory benchmark setup at upstream source %s", path, upstreamURL)
			}
			if strings.Contains(text, "github.com/XelHaku/agentmemory") {
				t.Fatalf("%s still points agentmemory benchmark setup at fork source", path)
			}
		})
	}
}

func TestPublicDocsMentionBackendComparisonSmoke(t *testing.T) {
	const smokeCommand = "make bench-locomo-backends-smoke"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention external backend comparison smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestBenchmarkDocsMentionConversationScopedBackendComparison(t *testing.T) {
	const marker = "conversation-scoped"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
		"docs-site/src/content/docs/operators/runbook.md",
		"docs/benchmarks/external-backend-adapters.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(strings.ToLower(string(raw)), marker) {
				t.Fatalf("%s does not mention %q backend comparison", path, marker)
			}
		})
	}
}

func TestBenchmarkDocsDocumentWrongBranchBackendRejections(t *testing.T) {
	wantMarkers := []string{
		"out-of-conversation `memory_id`",
		"`failure_bucket \"wrong_branch_retrieval\"`",
		"rejected before scoring",
		"not rescued by content matching or answer text",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
		"docs/benchmarks/external-backend-adapters.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not document wrong-branch backend rejection marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsDocumentBackendComparisonFailureBucketSummaries(t *testing.T) {
	wantMarkers := []string{
		"backend-comparison reports",
		"`failure_buckets`",
		"`Failure buckets`",
		"beside rank-based `failure_categories`",
		"stable-ID failure buckets",
		"without changing scoring or regenerating frozen LOCOMO artifacts",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
		"docs/benchmarks/external-backend-adapters.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not document backend comparison failure-bucket summary marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapNamesLocomoImprovementPriorities(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO improvement priorities",
		"multi-hop graph expansion",
		"query decomposition",
		"coverage-aware ranking",
		"temporal and speaker routing",
		"failure-audit buckets",
		"raise multi-hop recall_any@10 above `50%`",
		"raise multi-hop strict_recall@10 above `25%`",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not name LOCOMO improvement priority marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoGraphRecallSlice(t *testing.T) {
	wantMarkers := []string{
		"First graph-assisted implementation slice delivered",
		"TestGraphRecallConnectsOwnerThroughServiceRelation",
		"stable-ID companion memory",
		"relation path provenance",
		"before any LOCOMO full-run artifact is regenerated",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO graph recall slice marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoCoverageAwareSelectionSlice(t *testing.T) {
	wantMarkers := []string{
		"Coverage-aware graph companion selection delivered",
		"TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion",
		"prefers relation-path companion memories",
		"near-duplicate lexical hits",
		"without regenerating LOCOMO full-run artifacts",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO coverage-aware selection marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoQueryDecompositionSlice(t *testing.T) {
	wantMarkers := []string{
		"Query-decomposition recall slice delivered",
		"TestRecallQueryDecompositionRetrievesEachSubQuestionFact",
		"split into subqueries",
		"retrieve each required stable-ID fact",
		"merge results before scoring",
		"without regenerating LOCOMO full-run artifacts",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO query-decomposition marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoTemporalRoutingSlice(t *testing.T) {
	wantMarkers := []string{
		"Temporal current-truth routing slice delivered",
		"TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence",
		"current facts",
		"superseded evidence",
		"stable-ID memories",
		"without regenerating LOCOMO full-run artifacts",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO temporal routing marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoSpeakerRoutingSlice(t *testing.T) {
	wantMarkers := []string{
		"Speaker who-said-what routing slice delivered",
		"TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch",
		"who-said-what branches",
		"explicit speaker provenance",
		"stable-ID memories",
		"without regenerating LOCOMO full-run artifacts",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO speaker routing marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoFailureDrivenEvaluationSlice(t *testing.T) {
	wantMarkers := []string{
		"Failure-driven evaluation slice delivered",
		"TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets",
		"TestWriteLocomoFailureAuditEmitsFailureBucket",
		"wrong branch retrieval",
		"missing companion memories",
		"failure-audit buckets",
		"stable-ID memories",
		"without regenerating LOCOMO full-run artifacts",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO failure-driven evaluation marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkRoadmapSurfacesLocomoAnswerReadyCloseout(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO answer-ready closeout delivered",
		"current evidence chain supports a plain answer to how to improve Goncho",
		"target multi-hop recall_any@10 above `50%` and multi-hop strict_recall@10 above `25%`",
		"hybrid candidate generation",
		"graph expansion, query decomposition, coverage-aware selection, temporal routing, speaker routing, and failure-bucket audits",
		"backend-comparison `failure_buckets`",
		"without answer hints, LLM judges, answer-text scoring, or frozen artifact regeneration",
		"new date-stamped full LOCOMO run",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO answer-ready closeout marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkPlanDocumentsLocomoFailureDrivenEvaluation(t *testing.T) {
	const path = "docs/superpowers/plans/2026-05-22-locomo-failure-driven-evaluation.md"
	wantMarkers := []string{
		"# LOCOMO Failure-Driven Evaluation Implementation Plan",
		"TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets",
		"wrong branch retrieval",
		"missing companion memories",
		"failure-audit buckets",
		"stable inserted `memory_id`",
		"no answer hints, no LLM judges, no answer-text scoring",
		"Preserve frozen LOCOMO artifacts",
		"go test ./cmd/goncho-bench -run TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets -count=1",
		"go test ./... -count=1",
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	text := string(raw)
	for _, marker := range wantMarkers {
		if !strings.Contains(text, marker) {
			t.Fatalf("%s does not document LOCOMO failure-driven evaluation marker %q", path, marker)
		}
	}
}

func TestBenchmarkPlanDocumentsLocomoGraphAssistedMultiHopRecall(t *testing.T) {
	const path = "docs/superpowers/plans/2026-05-22-locomo-graph-assisted-multihop-recall.md"
	wantMarkers := []string{
		"# Graph-Assisted LOCOMO Multi-Hop Recall Implementation Plan",
		"Use superpowers:executing-plans",
		"TestGraphRecallConnectsOwnerThroughServiceRelation",
		"graph-expanded candidates must carry `EvidenceItem{Kind: \"graph\"`",
		"stable inserted `memory_id`",
		"no answer hints, no LLM judges, no answer-text scoring",
		"coverage-aware selection",
		"relation path provenance",
		"go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1",
		"go test ./... -count=1",
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	text := string(raw)
	for _, marker := range wantMarkers {
		if !strings.Contains(text, marker) {
			t.Fatalf("%s does not document LOCOMO graph-assisted recall marker %q", path, marker)
		}
	}
}

func TestBenchmarkPlanDocumentsLocomoQueryDecompositionRecall(t *testing.T) {
	const path = "docs/superpowers/plans/2026-05-22-locomo-query-decomposition-recall.md"
	wantMarkers := []string{
		"# LOCOMO Query Decomposition Recall Implementation Plan",
		"TestRecallQueryDecompositionRetrievesEachSubQuestionFact",
		"split multi-part questions",
		"merge and deduplicate by stable `memory_id`",
		"stable inserted `memory_id`",
		"no answer hints, no LLM judges, no answer-text scoring",
		"go test . -run TestRecallQueryDecompositionRetrievesEachSubQuestionFact -count=1",
		"go test ./... -count=1",
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	text := string(raw)
	for _, marker := range wantMarkers {
		if !strings.Contains(text, marker) {
			t.Fatalf("%s does not document LOCOMO query decomposition marker %q", path, marker)
		}
	}
}

func TestBenchmarkPlanDocumentsLocomoTemporalSpeakerRoutingRecall(t *testing.T) {
	const path = "docs/superpowers/plans/2026-05-22-locomo-temporal-speaker-routing-recall.md"
	wantMarkers := []string{
		"# LOCOMO Temporal and Speaker Routing Recall Implementation Plan",
		"TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence",
		"TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch",
		"changed facts, chronology, and who-said-what",
		"stable inserted `memory_id`",
		"superseded evidence remains preserved",
		"no answer hints, no LLM judges, no answer-text scoring",
		"go test . -run TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence -count=1",
		"go test ./... -count=1",
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	text := string(raw)
	for _, marker := range wantMarkers {
		if !strings.Contains(text, marker) {
			t.Fatalf("%s does not document LOCOMO temporal/speaker routing marker %q", path, marker)
		}
	}
}

func TestBenchmarkRoadmapNamesLocomoImplementationGate(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO implementation gate",
		"Recommendations are not approval to change retrieval behavior.",
		"Write an approved design or plan before production retrieval changes.",
		"Start implementation with a focused failing recall test",
		"TestGraphRecallConnectsOwnerThroughServiceRelation",
		"Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.",
		"Do not tune against LOCOMO gold IDs",
	}
	for _, path := range []string{
		"docs/benchmarks/ROADMAP.md",
		"docs-site/src/content/docs/roadmap/benchmark-roadmap.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not name LOCOMO implementation gate marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsRecommendLocomoImprovementLevers(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO improvement recommendations",
		"multi-hop recall_any@10 is `41.30%`",
		"multi-hop strict_recall@10 is `18.48%`",
		"single-hop strict_recall@10 is `13.48%`",
		"hybrid candidate generation",
		"multi-hop graph expansion",
		"query decomposition",
		"temporal and speaker routing",
		"coverage-aware ranking",
		"failure-driven evaluation",
		"retrieval improvements, not extra tools",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not recommend LOCOMO improvement lever marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoResultMetricSet(t *testing.T) {
	wantMarkers := []string{
		"NDCG@5",
		"NDCG@10",
		"latency min/p50/p95/max",
		"RSS",
		"database size",
		"memory token estimate",
		"Top-K",
		"failure categories",
		"leakage checks",
		"docs/benchmarks/results/locomo-2026-05-20-goncho.json",
		"docs/benchmarks/results/locomo-backend-comparison.json",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO result metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsDistinguishFrozenLocomoResultArtifacts(t *testing.T) {
	wantMarkers := []string{
		"frozen historical full-run evidence",
		"not regenerated by smoke targets",
		"regenerated smoke and backend-comparison artifacts",
		"latency/RSS measurements are host- and run-sensitive",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not distinguish frozen LOCOMO evidence with marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsNameLocomoReproductionCommands(t *testing.T) {
	wantMarkers := []string{
		"Full LOCOMO reproduction: `make bench-locomo`",
		"Retrieval smoke reproduction: `make bench-locomo-smoke`",
		"Backend smoke reproduction: `make bench-locomo-backends-smoke`",
		"Full backend comparison reproduction: `make bench-locomo-backends`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not name LOCOMO reproduction command marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsLinkLocomoFailureAuditArtifacts(t *testing.T) {
	wantMarkers := []string{
		"Failure audit evidence",
		"docs/benchmarks/failures/locomo-2026-05-20-categories.jsonl",
		"docs/benchmarks/failures/locomo-backend-comparison.jsonl",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not link LOCOMO failure-audit artifact marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsLabelSmokeFailureAuditArtifacts(t *testing.T) {
	wantMarkers := []string{
		"Smoke-only failure audit evidence",
		"docs/benchmarks/failures/locomo-smoke-categories.jsonl",
		"docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl",
		"not historical full-run evidence",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not label LOCOMO smoke failure-audit marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsLinkLocomoCandidateFailureComparisonAudit(t *testing.T) {
	wantMarkers := []string{
		"Candidate-generation failure comparison audit",
		"docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl",
		"BM25-win `missing_candidate`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not link LOCOMO candidate-generation audit marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsStateLocomoRetrievalOnlyScope(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO benchmark scope",
		"retrieval-only",
		"no answer generation",
		"no LLM judge",
		"ID-based scoring",
		"`answer_hint` fields are never indexed or scored",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not state LOCOMO retrieval-only scope marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsNameLocomoFullBaselineSet(t *testing.T) {
	wantMarkers := []string{
		"Full LOCOMO baseline set",
		"random, recency, BM25, SQLite FTS5, and Goncho",
		"frozen full LOCOMO run",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not name LOCOMO full-run baseline marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoSourceProvenance(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO source provenance",
		"https://github.com/snap-research/locomo",
		"3eb6f2c585f5e1699204e3c3bdf7adc5c28cb376",
		"Source SHA256: `79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4`",
		"Creative Commons Attribution-NonCommercial 4.0 International (CC BY-NC 4.0)",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO source provenance marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoConvertedDatasetEvidence(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO converted dataset evidence",
		"data/locomo/memories.jsonl",
		"data/locomo/questions.jsonl",
		"Questions: `1982`",
		"Memories: `5882`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO converted-dataset evidence marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoLeakageCheckCounts(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO leakage check counts",
		"Answer text present in memory content: `3026`",
		"Gold IDs present in memory content: `0`",
		"Question text present in memory content: `0`",
		"Answer-text presence is reported because LOCOMO answers may be literal spans from the gold memories",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO leakage-check marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoCategoryMetricGroups(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO category metric groups",
		"`adversarial_unanswerable`",
		"`multi_hop_retrieval`",
		"`open_domain_retrieval`",
		"`single_hop_retrieval`",
		"`temporal_retrieval`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO category metric group marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoCategoryQuestionCounts(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO category question counts",
		"`adversarial_unanswerable`: `446` questions",
		"`multi_hop_retrieval`: `92` questions",
		"`open_domain_retrieval`: `841` questions",
		"`single_hop_retrieval`: `282` questions",
		"`temporal_retrieval`: `321` questions",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO category question-count marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoGonchoCategoryMetrics(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO Goncho category metrics",
		"`adversarial_unanswerable`: recall_any@5 `61.66%`, recall_any@10 `71.52%`, MRR `48.90%`",
		"`multi_hop_retrieval`: recall_any@5 `35.87%`, recall_any@10 `41.30%`, MRR `24.76%`",
		"`open_domain_retrieval`: recall_any@5 `63.73%`, recall_any@10 `70.27%`, MRR `50.39%`",
		"`single_hop_retrieval`: recall_any@5 `47.16%`, recall_any@10 `59.22%`, MRR `31.91%`",
		"`temporal_retrieval`: recall_any@5 `66.98%`, recall_any@10 `71.96%`, MRR `54.47%`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO Goncho category metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoGonchoStrictCategoryMetrics(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO Goncho strict category metrics",
		"`adversarial_unanswerable`: strict_recall@5 `60.09%`, strict_recall@10 `69.73%`",
		"`multi_hop_retrieval`: strict_recall@5 `15.22%`, strict_recall@10 `18.48%`",
		"`open_domain_retrieval`: strict_recall@5 `60.76%`, strict_recall@10 `67.54%`",
		"`single_hop_retrieval`: strict_recall@5 `9.22%`, strict_recall@10 `13.48%`",
		"`temporal_retrieval`: strict_recall@5 `60.75%`, strict_recall@10 `65.11%`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO Goncho strict category metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoBM25CategoryMetrics(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO BM25 category metrics",
		"`adversarial_unanswerable`: recall_any@5 `61.88%`, recall_any@10 `71.52%`, MRR `48.92%`",
		"`multi_hop_retrieval`: recall_any@5 `35.87%`, recall_any@10 `41.30%`, MRR `24.76%`",
		"`open_domain_retrieval`: recall_any@5 `63.97%`, recall_any@10 `70.27%`, MRR `50.35%`",
		"`single_hop_retrieval`: recall_any@5 `46.81%`, recall_any@10 `58.87%`, MRR `31.71%`",
		"`temporal_retrieval`: recall_any@5 `66.67%`, recall_any@10 `72.59%`, MRR `54.60%`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO BM25 category metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoSQLiteFTS5CategoryMetrics(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO SQLite FTS5 category metrics",
		"`adversarial_unanswerable`: recall_any@5 `51.12%`, recall_any@10 `58.97%`, MRR `39.09%`",
		"`multi_hop_retrieval`: recall_any@5 `30.43%`, recall_any@10 `36.96%`, MRR `20.42%`",
		"`open_domain_retrieval`: recall_any@5 `52.68%`, recall_any@10 `60.05%`, MRR `41.87%`",
		"`single_hop_retrieval`: recall_any@5 `35.11%`, recall_any@10 `45.39%`, MRR `25.38%`",
		"`temporal_retrieval`: recall_any@5 `54.83%`, recall_any@10 `60.75%`, MRR `43.48%`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO SQLite FTS5 category metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoRandomCategoryMetrics(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO random baseline category metrics",
		"`adversarial_unanswerable`: recall_any@5 `1.35%`, recall_any@10 `2.47%`, MRR `0.88%`",
		"`multi_hop_retrieval`: recall_any@5 `3.26%`, recall_any@10 `5.43%`, MRR `1.58%`",
		"`open_domain_retrieval`: recall_any@5 `1.19%`, recall_any@10 `2.50%`, MRR `0.89%`",
		"`single_hop_retrieval`: recall_any@5 `2.48%`, recall_any@10 `3.55%`, MRR `0.97%`",
		"`temporal_retrieval`: recall_any@5 `0.00%`, recall_any@10 `0.62%`, MRR `0.08%`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO random category metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestBenchmarkDocsSurfaceLocomoRecencyCategoryMetrics(t *testing.T) {
	wantMarkers := []string{
		"LOCOMO recency baseline category metrics",
		"`adversarial_unanswerable`: recall_any@5 `0.45%`, recall_any@10 `0.45%`, MRR `0.28%`",
		"`multi_hop_retrieval`: recall_any@5 `0.00%`, recall_any@10 `1.09%`, MRR `0.16%`",
		"`open_domain_retrieval`: recall_any@5 `0.36%`, recall_any@10 `0.59%`, MRR `0.21%`",
		"`single_hop_retrieval`: recall_any@5 `0.71%`, recall_any@10 `1.77%`, MRR `0.29%`",
		"`temporal_retrieval`: recall_any@5 `0.31%`, recall_any@10 `0.93%`, MRR `0.14%`",
	}
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, marker := range wantMarkers {
				if !strings.Contains(text, marker) {
					t.Fatalf("%s does not surface LOCOMO recency category metric marker %q", path, marker)
				}
			}
		})
	}
}

func TestReleaseMetadataSmokeIncludesLocomoResultDocsGuards(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"BenchmarkDocsDocumentBackendComparisonFailureBucketSummaries",
		"BenchmarkRoadmapSurfacesLocomoAnswerReadyCloseout",
		"BenchmarkPlanDocumentsLocomoFailureDrivenEvaluation",
		"BenchmarkRoadmapSurfacesLocomoFailureDrivenEvaluationSlice",
		"BenchmarkRoadmapSurfacesLocomoSpeakerRoutingSlice",
		"BenchmarkRoadmapSurfacesLocomoTemporalRoutingSlice",
		"BenchmarkPlanDocumentsLocomoTemporalSpeakerRoutingRecall",
		"BenchmarkRoadmapSurfacesLocomoQueryDecompositionSlice",
		"BenchmarkPlanDocumentsLocomoQueryDecompositionRecall",
		"BenchmarkRoadmapSurfacesLocomoCoverageAwareSelectionSlice",
		"BenchmarkRoadmapSurfacesLocomoGraphRecallSlice",
		"BenchmarkPlanDocumentsLocomoGraphAssistedMultiHopRecall",
		"BenchmarkRoadmapNamesLocomoImprovementPriorities",
		"BenchmarkRoadmapNamesLocomoImplementationGate",
		"BenchmarkDocsRecommendLocomoImprovementLevers",
		"BenchmarkDocsSurfaceLocomoResultMetricSet",
		"BenchmarkDocsDistinguishFrozenLocomoResultArtifacts",
		"BenchmarkDocsNameLocomoReproductionCommands",
		"BenchmarkDocsLinkLocomoFailureAuditArtifacts",
		"BenchmarkDocsLabelSmokeFailureAuditArtifacts",
		"BenchmarkDocsLinkLocomoCandidateFailureComparisonAudit",
		"BenchmarkDocsStateLocomoRetrievalOnlyScope",
		"BenchmarkDocsNameLocomoFullBaselineSet",
		"BenchmarkDocsSurfaceLocomoSourceProvenance",
		"BenchmarkDocsSurfaceLocomoConvertedDatasetEvidence",
		"BenchmarkDocsSurfaceLocomoLeakageCheckCounts",
		"BenchmarkDocsSurfaceLocomoCategoryMetricGroups",
		"BenchmarkDocsSurfaceLocomoCategoryQuestionCounts",
		"BenchmarkDocsSurfaceLocomoGonchoCategoryMetrics",
		"BenchmarkDocsSurfaceLocomoGonchoStrictCategoryMetrics",
		"BenchmarkDocsSurfaceLocomoBM25CategoryMetrics",
		"BenchmarkDocsSurfaceLocomoSQLiteFTS5CategoryMetrics",
		"BenchmarkDocsSurfaceLocomoRandomCategoryMetrics",
		"BenchmarkDocsSurfaceLocomoRecencyCategoryMetrics",
		"ReleaseMetadataSmokeIncludesLocomoResultDocsGuards",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include LOCOMO result docs guard %q", want)
		}
	}
}

func TestPublicAdoptionDocsMentionPublicModuleSmoke(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "make public-module-smoke") {
				t.Fatalf("%s does not mention make public-module-smoke", path)
			}
		})
	}
}

func TestPublicDocsMentionLatestReleaseVersion(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "v0.3.0") {
				t.Fatalf("%s does not mention current public release v0.3.0", path)
			}
		})
	}
}

func TestPublicDocsUseLatestQualifiedGoGet(t *testing.T) {
	const qualifiedGoGet = "go get github.com/TrebuchetDynamics/goncho/service@latest"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), qualifiedGoGet) {
				t.Fatalf("%s does not mention version-qualified library adoption command %q", path, qualifiedGoGet)
			}
		})
	}
}

func TestPublicDocsMentionPublishedReleaseDate(t *testing.T) {
	const publishedDate = "published May 25, 2026"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), publishedDate) {
				t.Fatalf("%s does not mention public release date %q", path, publishedDate)
			}
		})
	}
}

func TestPublicDocsWarnRootGoInstallIsUnsupported(t *testing.T) {
	const rootInstallWarning = "not a root `go install` target"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), rootInstallWarning) {
				t.Fatalf("%s does not warn that the public module is %q", path, rootInstallWarning)
			}
		})
	}
}

func TestReadmeSurfacesPkgGoDevEvaluationPath(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## At a Glance",
		"If you are evaluating Goncho on pkg.go.dev",
		"First useful call",
		"compiled `NewService` example",
		"compiled `Service.Context`",
		"`Service.Search`",
		"`Service.Recall` examples",
		"What to read next",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface pkg.go.dev evaluation marker %q", want)
		}
	}
}

func TestReadmeSurfacesPkgGoDevAPIMap(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## API Map for pkg.go.dev Readers",
		"memory.OpenSqlite",
		"goncho.RunMigrations",
		"goncho.NewService",
		"svc.Conclude",
		"svc.Search",
		"svc.Recall",
		"svc.Context",
		"NewGonchoContextTool",
		"NewGonchoRecallTool",
		"goncho-bench@latest",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface pkg.go.dev API map marker %q", want)
		}
	}
}

func TestReadmeSurfacesImportPathGuide(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## Import Path Guide for pkg.go.dev Readers",
		"github.com/TrebuchetDynamics/goncho/service",
		"github.com/TrebuchetDynamics/goncho/memory",
		"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench",
		"Service library package",
		"SQLite opener",
		"Command only",
		"Stay on public service and tool APIs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface import path guide marker %q", want)
		}
	}
}

func TestReadmeSurfacesTrustBoundaryGuide(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## Trust Boundary for Agent Hosts",
		"Goncho can orient the agent",
		"The host remains authoritative",
		"Authorization and policy decisions",
		"Live filesystem, API, deployment, and credential state",
		"Money movement, destructive writes, and external side effects",
		"Treat retrieved memory as evidence to check",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface trust boundary guide marker %q", want)
		}
	}
}

func TestReadmeSurfacesHostIntegrationChecklist(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## Host Integration Checklist",
		"Open SQLite with `memory.OpenSqlite`",
		"Run `goncho.RunMigrations` before `goncho.NewService`",
		"Set `WorkspaceID` and `ObserverPeerID`",
		"Pass explicit `ProfileID`, `Peer`, and `SessionKey`",
		"Call `svc.Context` before tool execution",
		"Write conclusions with evidence",
		"Verify live state before acting",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface host integration checklist marker %q", want)
		}
	}
}

func TestReadmeSurfacesGoDevSignalMap(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"### go.dev Signal Map",
		"Valid go.mod file",
		"Redistributable license",
		"v0.3.0 / Latest",
		"published May 25, 2026",
		"make package-doc-smoke",
		"make public-module-smoke",
		"Imported by count is an adoption signal",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface go.dev signal map marker %q", want)
		}
	}
}

func TestReadmeSurfacesVersioningAndAdoptionNotes(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"### Versioning and Adoption Notes",
		"pre-1.0 stability",
		"go get github.com/TrebuchetDynamics/goncho/service@v0.3.0",
		"do not treat `@latest` as a deployment lock",
		"Imported by 0",
		"reverse-dependency count is not a correctness gate",
		"make ecosystem-smoke",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface versioning and adoption marker %q", want)
		}
	}
}

func TestReadmeSurfacesMinimalEmbeddedSkeleton(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## Minimal Embedded Skeleton",
		"Copy this skeleton into a new Go module",
		"package main",
		"memory.OpenSqlite",
		"goncho.RunMigrations",
		"goncho.NewService",
		"svc.Context",
		"orientation pack",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface minimal embedded skeleton marker %q", want)
		}
	}
}

func TestPackageDocsIncludeCompiledNewServiceExample(t *testing.T) {
	raw, err := os.ReadFile("service/example_service_test.go")
	if err != nil {
		t.Fatalf("ReadFile service/example_service_test.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"func ExampleNewService()",
		"memory.OpenSqlite",
		"goncho.RunMigrations",
		"goncho.NewService",
		"svc.SetProfile",
		"svc.Profile",
		"// Output:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("service/example_service_test.go does not include pkg.go.dev example marker %q", want)
		}
	}
}

func TestPackageDocsIncludeCompiledContextExample(t *testing.T) {
	raw, err := os.ReadFile("service/example_service_test.go")
	if err != nil {
		t.Fatalf("ReadFile service/example_service_test.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"func ExampleService_Context()",
		"svc.Conclude",
		"svc.Context",
		"orientation.Representation",
		"Current conclusions:",
		"// Output:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("service/example_service_test.go does not include pkg.go.dev context example marker %q", want)
		}
	}
}

func TestPackageDocsIncludeCompiledSearchExample(t *testing.T) {
	raw, err := os.ReadFile("service/example_service_test.go")
	if err != nil {
		t.Fatalf("ReadFile service/example_service_test.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"func ExampleService_Search()",
		"svc.Conclude",
		"svc.Search",
		"results.Results[0].Source",
		"results.Results[0].Content",
		"// Output:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("service/example_service_test.go does not include pkg.go.dev search example marker %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmePkgGoDevGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesPkgGoDevEvaluationPath",
		"ReleaseMetadataSmokeIncludesReadmePkgGoDevGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README pkg.go.dev guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeAPIMapGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesPkgGoDevAPIMap",
		"ReleaseMetadataSmokeIncludesReadmeAPIMapGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README API map guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeImportPathGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesImportPathGuide",
		"ReleaseMetadataSmokeIncludesReadmeImportPathGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README import path guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeTrustBoundaryGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesTrustBoundaryGuide",
		"ReleaseMetadataSmokeIncludesReadmeTrustBoundaryGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README trust boundary guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeHostIntegrationGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesHostIntegrationChecklist",
		"ReleaseMetadataSmokeIncludesReadmeHostIntegrationGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README host integration guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeGoDevSignalGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesGoDevSignalMap",
		"ReleaseMetadataSmokeIncludesReadmeGoDevSignalGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README go.dev signal guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeVersioningGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesVersioningAndAdoptionNotes",
		"ReleaseMetadataSmokeIncludesReadmeVersioningGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README versioning guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmeMinimalSkeletonGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesMinimalEmbeddedSkeleton",
		"ReleaseMetadataSmokeIncludesReadmeMinimalSkeletonGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README minimal skeleton guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageExampleGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocsIncludeCompiledNewServiceExample",
		"ReleaseMetadataSmokeIncludesPackageExampleGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package example guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesContextExampleGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocsIncludeCompiledContextExample",
		"ReleaseMetadataSmokeIncludesContextExampleGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include context example guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesSearchExampleGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocsIncludeCompiledSearchExample",
		"ReleaseMetadataSmokeIncludesSearchExampleGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include search example guard %q", want)
		}
	}
}

func TestPublicDocsFrameRootModuleAsLibrary(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "service package is a library package") {
				t.Fatalf("%s does not frame the service package as a library package", path)
			}
		})
	}
}

func TestEcosystemSmokeIncludesPublicReleaseMetadata(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"public-release-smoke:",
		"$(MAKE) public-release-smoke",
		"go list -m -json github.com/TrebuchetDynamics/goncho@latest",
		`"Version":`,
		`"Time":`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include public release metadata smoke marker %q", want)
		}
	}
}

func TestPublicReleaseSmokeChecksDocumentedLatestMetadata(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PUBLIC_LATEST_VERSION := v0.3.0",
		"PUBLIC_LATEST_PUBLISHED_DATE := 2026-05-25",
		`"Version": "$(PUBLIC_LATEST_VERSION)"`,
		`"Time": "$(PUBLIC_LATEST_PUBLISHED_DATE)`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not pin public release metadata smoke marker %q", want)
		}
	}
}

func TestPublicDocsExplainDocumentedLatestPublicReleaseSmoke(t *testing.T) {
	const marker = "documented public `@latest` version and published date"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), marker) {
				t.Fatalf("%s does not explain public-release smoke marker %q", path, marker)
			}
		})
	}
}

func TestLocalModuleSmokeChecksGoModMetadata(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"local-module-smoke:",
		"$(MAKE) local-module-smoke",
		"go list -m -json",
		`"Path": "github.com/TrebuchetDynamics/goncho"`,
		`"GoVersion":`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include local go.mod metadata smoke marker %q", want)
		}
	}
}

func TestPackageDocSurfacesPkgGoDevLandingContent(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Use Goncho when",
		"Quick start",
		"verification before action",
		"goncho.NewService",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not include pkg.go.dev landing marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocPointsPkgGoDevReadersToCompiledExamples(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"pkg.go.dev examples",
		"ExampleNewService",
		"ExampleService_Context",
		"ExampleService_Search",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not point pkg.go.dev readers to compiled examples marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesInstallAndCommandBoundary(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"go get github.com/TrebuchetDynamics/goncho/service@latest",
		"service package is a library package",
		"go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface install and command boundary marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesImportPathGuide(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Import path guide",
		"github.com/TrebuchetDynamics/goncho/service is the service library package",
		"github.com/TrebuchetDynamics/goncho/memory is the SQLite opener",
		"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench is command-only",
		"do not import cmd/goncho-bench into an agent host",
		"stay on public service and tool APIs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface import path guide marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesTrustBoundaryGuide(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Trust boundary for host agents",
		"Goncho can orient the agent",
		"host remains authoritative",
		"Authorization and policy decisions",
		"Live filesystem, API, deployment, and credential state",
		"Money movement, destructive writes, and external side effects",
		"Treat retrieved memory as evidence to check",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface trust boundary guide marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesHostIntegrationChecklist(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Host integration checklist",
		"memory.OpenSqlite",
		"RunMigrations before NewService",
		"WorkspaceID and ObserverPeerID",
		"ProfileID, Peer, and SessionKey",
		"Service.Context before tool execution",
		"evidence-backed conclusions",
		"Verify live state before acting",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface host integration checklist marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesPrimaryAPIPath(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Primary API path",
		"Service.Conclude",
		"Service.Search",
		"Service.Recall",
		"Service.Context",
		"public tool constructors",
		"NewGonchoContextTool",
		"NewGonchoRecallTool",
		"NewGonchoHandoffTool",
		"database internals",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface primary API path marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesGoDevPackageSignals(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"go.dev package signals",
		"v0.3.0",
		"valid go.mod",
		"redistributable MIT license",
		"make package-doc-smoke",
		"make public-module-smoke",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface go.dev package signal marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocSurfacesVersioningAndAdoptionNotes(t *testing.T) {
	out, err := exec.Command("go", "doc", "./service").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Versioning and adoption notes",
		"pre-1.0",
		"go get github.com/TrebuchetDynamics/goncho/service@v0.3.0",
		"@latest is a discovery shortcut, not a deployment lock",
		"Stable version",
		"Imported by 0",
		"reverse-dependency count",
		"make ecosystem-smoke",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not surface versioning and adoption marker %q\n%s", want, text)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocExamplesGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocPointsPkgGoDevReadersToCompiledExamples",
		"ReleaseMetadataSmokeIncludesPackageDocExamplesGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc examples guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocInstallGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesInstallAndCommandBoundary",
		"ReleaseMetadataSmokeIncludesPackageDocInstallGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc install guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocImportPathGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesImportPathGuide",
		"ReleaseMetadataSmokeIncludesPackageDocImportPathGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc import path guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocTrustBoundaryGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesTrustBoundaryGuide",
		"ReleaseMetadataSmokeIncludesPackageDocTrustBoundaryGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc trust boundary guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocHostIntegrationGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesHostIntegrationChecklist",
		"ReleaseMetadataSmokeIncludesPackageDocHostIntegrationGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc host integration guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocAPIPathGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesPrimaryAPIPath",
		"ReleaseMetadataSmokeIncludesPackageDocAPIPathGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc API path guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocGoDevSignalGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesGoDevPackageSignals",
		"ReleaseMetadataSmokeIncludesPackageDocGoDevSignalGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc go.dev signal guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocVersioningGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocSurfacesVersioningAndAdoptionNotes",
		"ReleaseMetadataSmokeIncludesPackageDocVersioningGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc versioning guard %q", want)
		}
	}
}

func TestPackageDocSmokeChecksLocalGoDoc(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"package-doc-smoke:",
		"$(MAKE) package-doc-smoke",
		"go doc ./service >/dev/null",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include package documentation smoke marker %q", want)
		}
	}
}

func TestDocsSiteSmokeBuildsPublicDocs(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"docs-site-smoke:",
		"$(MAKE) docs-site-smoke",
		"cd docs-site && npm run build",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include docs-site smoke marker %q", want)
		}
	}
}

func TestPublicDocsMentionDocsSiteSmoke(t *testing.T) {
	const smokeCommand = "make docs-site-smoke"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention docs-site smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestPublicDocsMentionPackageDocSmoke(t *testing.T) {
	const smokeCommand = "make package-doc-smoke"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention package documentation smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestPublicDocsMentionLocalModuleSmoke(t *testing.T) {
	const smokeCommand = "make local-module-smoke"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention local go.mod metadata smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestPublicDocsMentionPublicReleaseSmoke(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "make public-release-smoke") {
				t.Fatalf("%s does not mention make public-release-smoke", path)
			}
		})
	}
}
