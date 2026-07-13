package api

import (
	"strings"
)

// SearchToken represents a single token in a search expression.
type SearchToken struct {
	Text     string
	Operator TokenOperator
	FieldScope string // "name", "tags", "all", or "" for non-text scopes
}

// TokenOperator represents the operator for a search term.
type TokenOperator int

const (
	TokenInclude TokenOperator = iota
	TokenExclude
)

// ParseSearchExpression parses a search query string into tokens.
// Examples:
//   "2023 +mp4"        -> [2023(include), mp4(include)]
//   "2023 -apple"      -> [2023(include), apple(exclude)]
//   "holiday +beach"   -> [holiday(include), beach(include)]
//   "cat -dog +fish"   -> [cat(include), dog(exclude), fish(include)]
//
// Rules:
//   - Terms separated by spaces
//   - + prefix = explicit inclusion (same as default)
//   - - prefix = exclusion
//   - Quoted strings: "my photo" is treated as a single token
func ParseSearchExpression(query string) []SearchToken {
	if query == "" {
		return nil
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	var tokens []SearchToken
	remaining := query

	for len(remaining) > 0 {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		// Check for operators
		operator := TokenInclude
		if remaining[0] == '-' {
			operator = TokenExclude
			remaining = remaining[1:]
		} else if remaining[0] == '+' {
			operator = TokenInclude
			remaining = remaining[1:]
		}

		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		// Check for quoted string
		if remaining[0] == '"' || remaining[0] == '\'' {
			quote := remaining[0]
			remaining = remaining[1:]
			endIdx := strings.IndexAny(remaining, string(quote))
			if endIdx == -1 {
				// No closing quote, treat rest as term
				tokens = append(tokens, SearchToken{
					Text:     strings.TrimSpace(remaining),
					Operator: operator,
				})
				break
			}
			tokens = append(tokens, SearchToken{
				Text:     strings.TrimSpace(remaining[:endIdx]),
				Operator: operator,
			})
			remaining = remaining[endIdx+1:]
		} else {
			// Plain term - read until next space
			spaceIdx := strings.IndexAny(remaining, " \t")
			var term string
			if spaceIdx == -1 {
				term = remaining
				remaining = ""
			} else {
				term = remaining[:spaceIdx]
				remaining = remaining[spaceIdx:]
			}
			tokens = append(tokens, SearchToken{
				Text:     strings.TrimSpace(term),
				Operator: operator,
			})
		}
	}

	return tokens
}

// ApplyScopeToTokens applies a search scope to all tokens.
// Only "name", "tags", and "all" scopes support expression parsing.
func ApplyScopeToTokens(tokens []SearchToken, scope string) []SearchToken {
	if tokens == nil || len(tokens) == 0 {
		return tokens
	}

	switch scope {
	case "name":
		for i := range tokens {
			tokens[i].FieldScope = "name"
		}
	case "tags":
		for i := range tokens {
			tokens[i].FieldScope = "tags"
		}
	case "all":
		for i := range tokens {
			tokens[i].FieldScope = "all"
		}
	default:
		// For other scopes (place, location, date), clear field scope
		for i := range tokens {
			tokens[i].FieldScope = ""
		}
	}

	return tokens
}

// ExpressionResult holds the parsed result of a search expression for SQL generation.
type ExpressionResult struct {
	IncludeTerms []SearchToken
	ExcludeTerms []SearchToken
}

// BuildExpressionResult converts parsed tokens into include/exclude groups.
func BuildExpressionResult(tokens []SearchToken) ExpressionResult {
	result := ExpressionResult{
		IncludeTerms: []SearchToken{},
		ExcludeTerms: []SearchToken{},
	}

	for _, token := range tokens {
		if token.Operator == TokenExclude {
			result.ExcludeTerms = append(result.ExcludeTerms, token)
		} else {
			result.IncludeTerms = append(result.IncludeTerms, token)
		}
	}

	return result
}
