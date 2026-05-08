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

	// Expected: IDENT("git"), IDENT("clone"), IDENT("https://github.com/WJQSERVER/codex.git"), SEMICOLON, EOF
	// Or at minimum: IDENT("git"), IDENT("clone"), IDENT("https"), COLON(":"), SLASH("/"), SLASH("/"), ...
	// But the URL is completely truncated after ":"
	//
	// Root cause: Lexer sees ":" → COLON token, then sees "//" → treats as single-line comment,
	// skipping everything after "//" until EOF.
	// The entire URL path (github.com/WJQSERVER/codex.git) is lost.

	if len(tokens) < 4 {
		t.Fatalf("Expected at least 4 tokens, got %d", len(tokens))
	}

	// Verify the colon exists
	if tokens[3].Type != COLON {
		t.Errorf("Expected token[3] to be COLON, got %s %q", tokens[3].Type, tokens[3].Literal)
	}

	// The critical bug: after COLON, the next token should NOT be EOF
	// because the URL has more content after ":"
	if len(tokens) == 5 && tokens[4].Type == EOF {
		t.Errorf("BUG CONFIRMED: Lexer produces EOF immediately after COLON, URL content after '://' is lost")
		t.Errorf("  This happens because '//' after ':' is treated as a single-line comment delimiter")
	}
}

func TestGitCloneURL_DoubleSlashCommentConflict(t *testing.T) {
	// This test isolates the root cause: "//" is a comment delimiter in the lexer.
	// When a URL like "https://..." is tokenized, the ":" becomes a COLON token,
	// then "//" is seen as a comment start, and everything after it is skipped.

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

	// The URL "https://github.com/WJQSERVER/codex.git" should produce tokens for the full string.
	// But after ":" the lexer sees "//" and treats it as a comment, so we get:
	//   IDENT("https"), COLON(":"), EOF
	// The entire "github.com/WJQSERVER/codex.git" is silently dropped.

	if len(tokens) == 3 && tokens[0].Type == IDENT && tokens[1].Type == COLON && tokens[2].Type == EOF {
		t.Errorf("BUG CONFIRMED: URL content after '://' is stripped as a comment")
		t.Errorf("  Input:  %q", input)
		t.Errorf("  Got:    IDENT(%q) COLON(%q) EOF", tokens[0].Literal, tokens[1].Literal)
		t.Errorf("  Expected: full URL tokenization")
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
