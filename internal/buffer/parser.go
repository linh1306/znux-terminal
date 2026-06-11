package buffer

import (
	"bytes"
	"strings"
	"unicode"

	"mvdan.cc/sh/v3/syntax"
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

// tokenize splits input into shell words. It uses mvdan.cc/sh for complete
// shell fragments and falls back to a small tolerant scanner while the user is
// still typing incomplete input.
func (p *Parser) tokenize(text string) []Token {
	if tokens, ok := p.syntaxTokens(text); ok {
		return tokens
	}
	return p.fallbackTokenize(text)
}

func (p *Parser) syntaxTokens(text string) ([]Token, bool) {
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := parser.Parse(strings.NewReader(text+"\n"), "")
	if err != nil {
		return nil, false
	}

	var lastCall *syntax.CallExpr
	syntax.Walk(file, func(node syntax.Node) bool {
		if call, ok := node.(*syntax.CallExpr); ok {
			lastCall = call
		}
		return true
	})
	if lastCall == nil || len(lastCall.Args) == 0 {
		return nil, true
	}

	tokens := make([]Token, 0, len(lastCall.Args))
	for _, word := range lastCall.Args {
		value := word.Lit()
		if value == "" {
			value = p.renderWord(word)
		}
		if value == "" {
			continue
		}
		tokens = append(tokens, p.classifyToken(value))
	}
	return tokens, true
}

func (p *Parser) renderWord(word *syntax.Word) string {
	var buf bytes.Buffer
	printer := syntax.NewPrinter()
	if err := printer.Print(&buf, word); err != nil {
		return ""
	}
	return buf.String()
}

// fallbackTokenize splits input into tokens when the shell parser reports
// incomplete syntax such as an unclosed quote.
func (p *Parser) fallbackTokenize(text string) []Token {
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

	endsWithSpace := text[len(text)-1] == ' ' || text[len(text)-1] == '\t'

	// Tokenize trimmed text so trailing spaces don't produce empty tokens
	trimmed := text
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == ' ' || trimmed[len(trimmed)-1] == '\t') {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if trimmed == "" {
		return Context{Level: ContextCommand}
	}

	tokens := p.tokenize(trimmed)
	if len(tokens) == 0 {
		return Context{Level: ContextCommand}
	}

	command := tokens[0].Value

	if len(tokens) == 1 {
		if endsWithSpace {
			// "git " → suggest subcommands
			return Context{Level: ContextSubcommand, Command: command}
		}
		// "gi" → partial command name
		return Context{Level: ContextCommandPartial, Command: command}
	}

	// subcommand is the second token if it doesn't start with '-'
	subcommand := ""
	if len(tokens[1].Value) > 0 && tokens[1].Value[0] != '-' {
		subcommand = tokens[1].Value
	}

	if endsWithSpace {
		// "git checkout " or "git checkout main " → suggest args/branches
		return Context{Level: ContextArg, Command: command, Subcommand: subcommand}
	}

	// Ends with a partial token being typed
	last := tokens[len(tokens)-1]
	if len(last.Value) > 0 && last.Value[0] == '-' {
		return Context{Level: ContextFlagPartial, Command: command, Subcommand: subcommand, Flag: last.Value}
	}
	if len(tokens) == 2 {
		// "git che" → still typing the subcommand
		return Context{Level: ContextSubcommandPartial, Command: command, Subcommand: last.Value}
	}
	// "git checkout ma" → partial arg after a known subcommand
	return Context{Level: ContextArgPartial, Command: command, Subcommand: subcommand}
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
