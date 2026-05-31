package searchfilter

import "github.com/TrebuchetDynamics/goncho/internal/searchfilter/contracts"

type Kind = contracts.Kind

const (
	KindAll        = contracts.KindAll
	KindAnd        = contracts.KindAnd
	KindOr         = contracts.KindOr
	KindNot        = contracts.KindNot
	KindComparison = contracts.KindComparison
)

type Operator = contracts.Operator

const (
	OpEQ        = contracts.OpEQ
	OpGT        = contracts.OpGT
	OpGTE       = contracts.OpGTE
	OpLT        = contracts.OpLT
	OpLTE       = contracts.OpLTE
	OpNE        = contracts.OpNE
	OpIn        = contracts.OpIn
	OpContains  = contracts.OpContains
	OpIContains = contracts.OpIContains
)

type Expression = contracts.Expression

type Compiled = contracts.Compiled
