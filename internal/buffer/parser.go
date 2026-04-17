package buffer

import (
	"unicode"
)

// Parser handles command line parsing
type Parser struct{}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseInput parses the input buffer into structured components
func (p *Parser) ParseInput(buf *LineBuf) *ParseResult {
	text := buf.String()
	if text == "" {
		return &ParseResult{}
	}

	result := &ParseResult{
		Tokens: p.tokenize(text),
	}

	if len(result.Tokens) > 0 {
		result.Command = result.Tokens[0].Value
	}

	return result
}

// tokenize splits input into tokens
func (p *Parser) tokenize(text string) []Token {
	var tokens []Token
	var current []rune
	inQuote := false
	quoteChar := rune(0)

	for i, r := range text {
		if inQuote {
			if r == quoteChar {
				tokens = append(tokens, Token{Type: TokenArg, Value: string(current)})
				current = nil
				inQuote = false
			} else {
				current = append(current, r)
			}
		} else {
			if r == '"' || r == '\'' {
				inQuote = true
				quoteChar = r
			} else if r == ' ' || r == '\t' {
				if len(current) > 0 {
					tokens = append(tokens, p.classifyToken(string(current)))
					current = nil
				}
			} else {
				current = append(current, r)
			}
		}

		// Handle end of string
		if i == len(text)-1 && len(current) > 0 {
			tokens = append(tokens, p.classifyToken(string(current)))
		}
	}

	return tokens
}

// classifyToken determines the type of a token
func (p *Parser) classifyToken(value string) Token {
	if len(value) > 0 && value[0] == '-' {
		return Token{Type: TokenFlag, Value: value}
	}
	return Token{Type: TokenArg, Value: value}
}

// TokenType represents the type of a token
type TokenType int

const (
	TokenCommand TokenType = iota
	TokenSubcommand
	TokenFlag
	TokenArg
	TokenOption
)

// Token represents a parsed token
type Token struct {
	Type  TokenType
	Value string
}

// ParseResult holds the parsed result
type ParseResult struct {
	Command    string
	Subcommand string
	Tokens     []Token
}

// GetCurrentContext returns the context for autocomplete
// based on what's currently being typed
func (p *Parser) GetCurrentContext(buf *LineBuf) Context {
	text := buf.String()
	if text == "" {
		return Context{Level: ContextCommand}
	}

	// Strip trailing whitespace for analysis
	trimmed := text
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == ' ' || trimmed[len(trimmed)-1] == '\t') {
		trimmed = trimmed[:len(trimmed)-1]
	}

	if trimmed == "" {
		return Context{Level: ContextCommand}
	}

	// Count tokens
	tokens := p.tokenize(trimmed)
	numTokens := len(tokens)

	if numTokens == 1 {
		// Partial command being typed
		if buf.String()[len(buf.String())-1] == ' ' {
			return Context{Level: ContextSubcommand, Command: tokens[0].Value}
		}
		return Context{Level: ContextCommandPartial, Command: tokens[0].Value}
	}

	// Get last non-space character
	lastRune := rune(0)
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] != ' ' && text[i] != '\t' {
			lastRune = rune(text[i])
			break
		}
	}

	if lastRune == ' ' || lastRune == '\t' {
		// Ends with space - expecting subcommand, flag, or arg
		if numTokens == 2 {
			return Context{Level: ContextSubcommand, Command: tokens[0].Value}
		}
		return Context{Level: ContextArg}
	}

	// Ends with text - partial token
	token := tokens[len(tokens)-1]
	if token.Value[0] == '-' {
		return Context{Level: ContextFlagPartial, Command: tokens[0].Value}
	}
	// With exactly two tokens the second one is a partial subcommand
	// (e.g. "git c" — user is still typing the subcommand name).
	if numTokens == 2 {
		return Context{Level: ContextSubcommandPartial, Command: tokens[0].Value, Subcommand: token.Value}
	}
	return Context{Level: ContextArgPartial, Command: tokens[0].Value, Subcommand: tokens[1].Value}
}

// Context represents what kind of suggestion to provide
type Context struct {
	Level       ContextLevel
	Command     string
	Subcommand  string
	Flag        string
	PartialWord string
}

// ContextLevel represents the current parsing context
type ContextLevel int

const (
	ContextCommand ContextLevel = iota
	ContextCommandPartial
	ContextSubcommand
	ContextSubcommandPartial
	ContextFlag
	ContextFlagPartial
	ContextArg
	ContextArgPartial
)

// IsWordChar returns true if rune is a valid word character
func IsWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/'
}