package contracts

// Kind identifies a node in a parsed search filter expression tree.
type Kind string

const (
	KindAll        Kind = "all"
	KindAnd        Kind = "and"
	KindOr         Kind = "or"
	KindNot        Kind = "not"
	KindComparison Kind = "comparison"
)

// Operator identifies a field comparison operator in a search filter.
type Operator string

const (
	OpEQ        Operator = "eq"
	OpGT        Operator = "gt"
	OpGTE       Operator = "gte"
	OpLT        Operator = "lt"
	OpLTE       Operator = "lte"
	OpNE        Operator = "ne"
	OpIn        Operator = "in"
	OpContains  Operator = "contains"
	OpIContains Operator = "icontains"
)

// Expression is the parsed search filter expression tree.
type Expression struct {
	Kind     Kind
	Children []Expression
	Field    string
	Operator Operator
	Values   []string
}

// Compiled is the enforceable subset of a parsed search filter.
type Compiled struct {
	SessionIDs []string
	Sources    []string
	DenyAll    bool
}
