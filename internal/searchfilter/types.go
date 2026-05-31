package searchfilter

type Kind string

const (
	KindAll        Kind = "all"
	KindAnd        Kind = "and"
	KindOr         Kind = "or"
	KindNot        Kind = "not"
	KindComparison Kind = "comparison"
)

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

type Expression struct {
	Kind     Kind
	Children []Expression
	Field    string
	Operator Operator
	Values   []string
}

type Compiled struct {
	SessionIDs []string
	Sources    []string
	DenyAll    bool
}
