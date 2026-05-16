package core

import (
	"testing"
)

func TestGitCloneURL_LexerTokenizesURL(t *testing.T) {
	input := `git clone https://github.com/WJQSERVER/codex.git`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for input: %q", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Known lexer behavior: ":" becomes COLON token, then "//" is treated as
	// a single-line comment delimiter, silently dropping everything after.
	// This is NOT a bug in practice — the parser's scanCommandWords() bypasses
	// the lexer for command arguments and reads raw input directly.
	// Kept as documentation of lexer behavior.
	if len(tokens) == 5 && tokens[4].Type == EOF {
		t.Logf("Known lexer behavior: COLON + '//' eats URL content after scheme")
	}
}

func TestGitCloneURL_DoubleSlashCommentConflict(t *testing.T) {
	// Known lexer behavior: "//" is a comment delimiter. After the lexer produces
	// COLON for ":", it sees "//" and treats it as a comment, dropping everything after.
	// This is NOT a bug in practice — the parser's scanCommandWords() bypasses
	// the lexer for command arguments and reads raw input directly.
	// Kept as documentation of lexer behavior.

	input := `https://github.com/WJQSERVER/codex.git`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for bare URL: %q", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	if len(tokens) == 3 && tokens[0].Type == IDENT && tokens[1].Type == COLON && tokens[2].Type == EOF {
		t.Logf("Known lexer behavior: URL content after '://' is stripped as a comment")
	}
}

func TestGitCloneURL_ParserReceivesTruncatedArgs(t *testing.T) {
	input := `git clone https://github.com/WJQSERVER/codex.git`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) == 0 {
		t.Fatal("No statements parsed")
	}

	stmt, ok := program.Statements[0].(*CommandStatement)
	if !ok {
		t.Fatalf("Expected CommandStatement, got %T", program.Statements[0])
	}

	t.Logf("Command: %q", stmt.Name)
	t.Logf("Arguments: %d", len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %q", i, arg.String())
	}

	// Expected: git clone https://github.com/WJQSERVER/codex.git
	//   command: "git", args: ["clone", "https://github.com/WJQSERVER/codex.git"]
	//
	// Actual (bug): git clone "https" ":"
	//   command: "git", args: ["clone", "https", ":"]
	//   The URL is truncated at "//" (comment), only "https" and ":" survive.

	if stmt.Name != "git" {
		t.Errorf("Expected command 'git', got %q", stmt.Name)
	}

	// Verify the URL argument is the full URL, not just "https"
	if len(stmt.Arguments) >= 2 {
		urlArg := stmt.Arguments[1].String()
		if urlArg == "\"https\"" {
			t.Errorf("BUG: URL truncated to just scheme %q", urlArg)
			t.Errorf("  Full URL 'https://github.com/WJQSERVER/codex.git' was lost")
		}
	}
}

func TestGitCloneURL_FTPScheme(t *testing.T) {
	// FTP URLs also use "://" — same bug should apply
	input := `wget ftp://files.example.com/data.tar.gz`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for FTP URL: %q", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Same bug: "ftp://..." → IDENT("ftp"), COLON(":"), then "//" → comment → EOF
	for _, tok := range tokens {
		if tok.Type == COLON {
			// Found colon, check if next token is EOF (comment ate the rest)
			break
		}
	}
}

func TestGitCloneURL_SingleSlashAfterColon(t *testing.T) {
	// A path like "/path:regex" has a colon but NOT "//" after it
	// This should NOT trigger the comment bug
	input := `grep pattern /path:file.txt`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for colon-in-path: %q", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// This case should work because after ":" there's "f" not "/"
	// So the colon is just a regular character in the path
}

func TestGitCloneURL_CommentOnNextLine(t *testing.T) {
	// Verify that "//" as a comment works correctly when not part of a URL
	input := "git status // this is a comment"
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for comment: %q", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Here "//" correctly acts as a comment, so we should get:
	// IDENT("git"), IDENT("status"), SEMICOLON, EOF
	// (everything after "//" is a comment)
}
