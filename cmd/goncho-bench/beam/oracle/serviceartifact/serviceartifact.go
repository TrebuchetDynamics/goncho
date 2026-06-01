package serviceartifact

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/filecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/results"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/scoring"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/summary"
)

// SummaryFile is the stable JSON contract for BEAM service summary artifacts.
type SummaryFile = filecontract.SummaryFile

// SummaryMetadata describes a BEAM service summary artifact run.
type SummaryMetadata = filecontract.SummaryMetadata

// AbilityStats summarizes scores for one ability bucket.
type AbilityStats = filecontract.AbilityStats

// PairedOutcome reuses the shared paired-comparison artifact row contract.
type PairedOutcome = filecontract.PairedOutcome

// FailureAuditRow is the stable JSONL contract for failed BEAM service cases.
type FailureAuditRow = filecontract.FailureAuditRow

// ResultsFile is the stable JSON contract for BEAM service result artifacts.
type ResultsFile = filecontract.ResultsFile

// ResultsMetadata describes a BEAM service results artifact run.
type ResultsMetadata = filecontract.ResultsMetadata

// ConversationResults groups question results for one BEAM conversation.
type ConversationResults = filecontract.ConversationResults

// QuestionResult is one stable BEAM service result row.
type QuestionResult = filecontract.QuestionResult

// CaseScoreFunc supplies the artifact score used by builders that can be
// pure-recall scored or external-judgment scored.
type CaseScoreFunc = scoring.CaseScoreFunc

// CaseJudgment contains optional external answer/judge fields for one result row.
type CaseJudgment = results.CaseJudgment

// CaseJudgmentFunc supplies optional external judgment fields for a report case.
type CaseJudgmentFunc = results.CaseJudgmentFunc

// ResultsOptions contains metadata and callbacks needed to build service results.
type ResultsOptions = results.Options

// SummaryOptions contains run metadata supplied by the oracle service facade.
type SummaryOptions = summary.Options
