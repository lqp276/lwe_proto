package protoc

import (
	"fmt"
	"unicode"

	"github.com/pkg/errors"
)

//Token types
const (
	//const
	INT_CONST    = "INT_CONST"
	STRING_CONST = "STRING_CONST"

	//primitive type
	TYPE_INT    = "INT"
	TYPE_U1     = "U1"
	TYPE_U2     = "U2"
	TYPE_U3     = "U3"
	TYPE_U4     = "U4"
	TYPE_U5     = "U5"
	TYPE_U6     = "U6"
	TYPE_U7     = "U7"
	TYPE_U8     = "U8"
	TYPE_U16    = "U16"
	TYPE_U32    = "U32"
	TYPE_U64    = "U64"
	TYPE_V32    = "V32"
	TYPE_V64    = "V64"
	TYPE_CHAR   = "CHAR"
	TYPE_STRING = "STRING"
	TYPE_ANY    = "ANY"

	//special chars
	LPAREN   = "LPAREN"   //"("
	RPAREN   = "RPAREN"   //")"
	LBRACE   = "LBRACE"   // "{"
	RBRACE   = "RBRACE"   // "}"
	LBRACKET = "LBRACKET" //"["
	RBRACKET = "RBRACKET" //"["
	COLON    = "COLON"    // ":"
	COMMA    = "COMMA"    // ","
	SEMI     = "SEMI"     // ";"
	DOT      = "DOT"      // "."
	LF       = "LF"       //"\n"
	TYPE     = "TYPE"
	STRUCT   = "STRUCT"
	QUOTE2   = "\""

	//operator
	LIMIT    = "LIMIT"
	BY       = "BY"
	MAX      = "MAX"
	NEW      = "NEW"
	DEFMSG   = "DEFMSG"
	DEFID    = "DEFID"
	SCOMMENT = "SCOMMENT"
	DEFMID   = "DEFMID"
	DEFBIND  = "DEFBIND"
	NIL      = "nil"

	OR     = "OR" //"||"
	XOR    = "XOR"
	MEND   = "MEND"
	MSPACE = "MSPACE"

	AND = "AND" //"&&"

	DESC = "DESC" //->
	EQU  = "EQ"   //"=="
	NEQ  = "NEQ"  //"!="

	LT      = "LT"  //"<"
	LTE     = "LTE" //"<="
	GT      = "GT"  //">"
	GTE     = "GTE" //">="
	LSHIFT  = "LSHIFT"
	RSHIFT  = "RSHIFT"
	BIT_AND = "BIT_AND"
	BIT_OR  = "BIT_OR"

	PLUS  = "PLUS"  //"+"
	MINUS = "MINUS" //"-"

	MUL = "MUL" //"*"
	DIV = "DIV" //"/"

	NOT = "NOT" //"!"

	ID         = "ID"
	VAR        = "VAR"
	CONST      = "CONST"
	ASSIGN     = "ASSIGN"     //"="
	DEC_ASSIGN = "DEC_ASSIGN" //":="
	NONE       = "NONE"
	RETURN     = "RETURN"

	//flow control
	IF     = "IF"
	EXIST  = "EXIST"
	THIS   = "THIS"
	ELIF   = "ELIF"
	ELSE   = "ELSE"
	WHILE  = "WHILE"
	BREAK  = "BREAK"
	EXTERN = "EXTERN"

	FOLLOW = "FOLLOW"
	ABOVE  = "ABOVE"

	//EOF
	EOF = "EOF"
)

var keywords = map[string]string{
	"var":    VAR,
	"const":  CONST,
	"int":    TYPE_INT,
	"None":   NONE,
	"string": TYPE_STRING,
	"return": RETURN,
	"any":    TYPE_ANY,
	"type":   TYPE,
	"struct": STRUCT,
	"new":    NEW,
	"if":     IF,
	"elif":   ELIF,
	"else":   ELSE,
	"while":  WHILE,
	"break":  BREAK,
	"u1":     TYPE_U1,
	"u2":     TYPE_U2,
	"u3":     TYPE_U3,
	"u4":     TYPE_U4,
	"u5":     TYPE_U5,
	"u6":     TYPE_U6,
	"u7":     TYPE_U7,
	"u8":     TYPE_U8,
	"char":   TYPE_CHAR,
	"u16":    TYPE_U16,
	"u32":    TYPE_U32,
	"u64":    TYPE_U64,
	"v32":    TYPE_V32,
	"v64":    TYPE_V64,
	"defmsg": DEFMSG,
	"defid":  DEFID,
	"limit":  LIMIT,
	"by":     BY,
	"max":    MAX,
	"equal":  EQU,
	"xor":    XOR,
	"exist":  EXIST,
	"this":   THIS,
	"extern": EXTERN,
	"defmid": DEFMID,
	"bind":   DEFBIND,
	"nil":    NIL,
	"mend":   MEND, //mark message end, and do strict decode check
	"mspace": MSPACE,
}

type Token struct {
	type_  string
	value  string
	line   int
	column int
}

func (tok *Token) String() string {
	return fmt.Sprintf("Token(%s:\"%v\" pos: %d:%d)", tok.type_, tok.value, tok.line, tok.column)
}

type hskLexer struct {
	text      []rune
	pos       int
	posMax    int
	curChar   rune
	lineNo    int
	colNo     int
	lastError error
}

func (lex *hskLexer) advanceBy(cnt int) {
	for lex.pos < lex.posMax && cnt > 0 {
		cnt--
		lex.advance()
	}
}

func (lex *hskLexer) advance() {
	lex.pos++
	if lex.pos < lex.posMax {
		//skip current char
		if lex.curChar == '\n' {
			lex.lineNo++
			lex.colNo = 0
		}

		//fmt.Printf("advance skip: %s\n", string(lex.curChar))

		lex.colNo++
		lex.curChar = lex.text[lex.pos]
	} else {
		lex.curChar = 0
	}
}

func (lex *hskLexer) skipComment(mult bool) []rune {
	line := lex.lineNo
	pos := lex.pos
	for lex.pos < lex.posMax {
		if mult && lex.curChar == '*' && lex.peekChar(1) == '/' {
			lex.advanceBy(2)
			break
		}

		lex.advance()

		if !mult && lex.lineNo != line {
			break
		}
	}

	if mult {
		return lex.text[pos:lex.pos]
	}

	var skipped []rune
	for i := pos; i < lex.pos; i++ {
		ch := lex.text[i]
		if ch == '\n' {
			break
		}

		skipped = append(skipped, ch)
	}
	return skipped
}

func (lex *hskLexer) peekChar(idx int) rune {
	dst := lex.pos + idx
	if dst < lex.posMax && dst >= 0 {
		return lex.text[dst]
	} else {
		return 0
	}
}

func (lex *hskLexer) lexerError(desc string) {

}

func (lex *hskLexer) getInteger() string {
	hex := false
	numDigits := []rune{}
	if lex.curChar == '0' && (lex.peekChar(1) == 'x' || lex.peekChar(1) == 'X') {
		numDigits = append(numDigits, '0', 'x')
		lex.advanceBy(2)
		hex = true
	}

loop:
	for lex.curChar != 0 {
		if hex {
			switch lex.curChar {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
				'a', 'b', 'c', 'd', 'e', 'f',
				'A', 'B', 'C', 'D', 'E', 'F':
				numDigits = append(numDigits, lex.curChar)
				lex.advance()
				break

			default:
				break loop
			}
		} else {
			switch lex.curChar {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				numDigits = append(numDigits, lex.curChar)
				lex.advance()
				break

			default:
				break loop
			}
		}
	}

	return string(numDigits)
}

func (lex *hskLexer) escapedChar(escaped rune) rune {
	switch escaped {
	//case 'u':
	case 't':
		return '\t'

	case 'b':
		return '\b'

	case 'n':
		return '\n'

	case 'r':
		return '\r'

	case 'f':
		return '\f'

	default:
		return escaped
	}
}

func (lex *hskLexer) getString() string {
	lex.advance()

	var val []rune
	for lex.curChar != '"' {
		switch lex.curChar {
		case '\\':
			lex.advance()
			val = append(val, lex.escapedChar(lex.curChar))
			break

		default:
			val = append(val, lex.curChar)
		}

		lex.advance()
	}

	lex.advance()
	return string(val)
}

func (lex *hskLexer) skipSpaces() {
	for unicode.IsSpace(lex.curChar) {
		lex.advance()
	}
}

func (lex *hskLexer) getId() *Token {
	col := lex.colNo
	letters := []rune{}

	for unicode.IsLetter(lex.curChar) || unicode.IsDigit(lex.curChar) || lex.curChar == '_' {
		letters = append(letters, lex.curChar)
		lex.advance()
	}

	value := string(letters)
	type_, ok := keywords[value]
	if ok {
		return &Token{type_, value, lex.lineNo, col}
	}

	return &Token{ID, value, lex.lineNo, col}
}

type parseCtx struct {
	pos     int
	line    int
	col     int
	curChar rune
}

func (lex *hskLexer) peekToken() *Token {
	parseCtx := &parseCtx{lex.pos, lex.lineNo, lex.colNo, lex.curChar}
	token := lex.getNextToken()
	lex.pos = parseCtx.pos
	lex.lineNo = parseCtx.line
	lex.colNo = parseCtx.col
	lex.curChar = parseCtx.curChar
	return token
}

func (lex *hskLexer) getLastError() error {
	return lex.lastError
}

func (lex *hskLexer) getNextToken() (tok *Token) {
	for {
		lex.skipSpaces()
		if lex.curChar == 0 {
			return &Token{EOF, "EOF", lex.lineNo, lex.colNo}
		}

		col := lex.colNo
		line := lex.lineNo
		if unicode.IsDigit(lex.curChar) {
			val := lex.getInteger()
			return &Token{type_: INT_CONST, value: val, line: line, column: col}
		}

		if unicode.IsLetter(lex.curChar) {
			return lex.getId()
		}

		if lex.curChar == '"' {
			val := lex.getString()
			return &Token{type_: STRING_CONST, value: val, line: line, column: col}
		}

		switch lex.curChar {
		case '+':
			token := &Token{PLUS, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '-':
			token := &Token{MINUS, string(lex.curChar), lex.lineNo, lex.colNo}
			nextChar := lex.peekChar(1)
			if nextChar == '>' {
				token = &Token{DESC, "->", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
				return token
			}
			token = &Token{DIV, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '*':
			token := &Token{MUL, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '/':
			var token *Token
			nextChar := lex.peekChar(1)
			if nextChar == '*' {
				lex.skipComment(true)
				continue
			}

			if nextChar == '/' {
				if lex.peekChar(2) == '*' {
					line := lex.lineNo
					comm := lex.skipComment(false)
					token = &Token{SCOMMENT, string(comm[3:]), line, lex.colNo}
					return token
				} else {
					lex.skipComment(false)
					continue
				}
			}

			token = &Token{DIV, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case ':':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '=' {
				token = &Token{DEC_ASSIGN, ":=", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{COLON, ":", lex.lineNo, lex.colNo}
				lex.advanceBy(1)
			}
			return token

		case '=':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '=' {
				token = &Token{EQU, "==", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{ASSIGN, "=", lex.lineNo, lex.colNo}
				lex.advanceBy(1)
			}
			return token

		case '!':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '=' {
				token = &Token{NEQ, "!=", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{NOT, "!", lex.lineNo, lex.colNo}
				lex.advanceBy(1)
			}
			return token

		case '&':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '&' {
				token = &Token{AND, "&&", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{BIT_AND, "&", lex.lineNo, lex.colNo}
				lex.advance()
			}
			return token

		case '|':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '|' {
				token = &Token{OR, "||", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{BIT_OR, "|", lex.lineNo, lex.colNo}
				lex.advance()
			}
			return token

		case '<':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '=' {
				token = &Token{LTE, "<=", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else if nextChar == '<' {
				token = &Token{LSHIFT, "<<", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{LT, "<", lex.lineNo, lex.colNo}
				lex.advanceBy(1)
			}
			return token

		case '>':
			nextChar := lex.peekChar(1)
			var token *Token
			if nextChar == '=' {
				token = &Token{GTE, ">=", lex.lineNo, lex.colNo}
				lex.advanceBy(2)
			} else {
				token = &Token{GT, ">", lex.lineNo, lex.colNo}
				lex.advanceBy(1)
			}
			return token

		case '(':
			token := &Token{LPAREN, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case ')':
			token := &Token{RPAREN, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '{':
			token := &Token{LBRACE, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '}':
			token := &Token{RBRACE, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case ',':
			token := &Token{COMMA, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case ';':
			token := &Token{SEMI, string(lex.curChar), lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '\n':
			token := &Token{LF, "\\n", lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '[':
			token := &Token{LBRACKET, "[", lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case ']':
			token := &Token{RBRACKET, "]", lex.lineNo, lex.colNo}
			lex.advance()
			return token

		case '.':
			token := &Token{DOT, ".", lex.lineNo, lex.colNo}
			lex.advance()
			return token

		default:
			lex.lastError = errors.Errorf("unknown char \"%s\": %x at line: %d: %d", string(lex.curChar), lex.curChar, lex.lineNo, lex.colNo)
			return nil
		}
	}
}

func newLexer(text string) *hskLexer {
	lex := &hskLexer{}
	lex.text = []rune(text)
	lex.posMax = len(lex.text)
	if lex.posMax > 0 {
		lex.lineNo = 1
		lex.curChar = lex.text[0]
	}
	return lex
}
