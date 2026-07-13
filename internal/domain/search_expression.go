package domain

// ExpressionResult holds the parsed result of a search expression for SQL generation.
// Used by the repository to build SQL with include AND exclude conditions.
type ExpressionResult struct {
	IncludeTerms []ExpressionTerm
	ExcludeTerms []ExpressionTerm
}

// ExpressionTerm represents a single term in a search expression.
type ExpressionTerm struct {
	Value      string
	FieldScope string // "name", "tags", or "" (all fields)
}
