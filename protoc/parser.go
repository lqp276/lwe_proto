package protoc

import (
	"fmt"

	"github.com/pkg/errors"
)

/*
declarations : const_decl | msg_decl
const_decl: CONST ID expr
msg_decl: DEFMSG ID LBRACE
     field_decl*
RBRACE

field_decl: ID type_spec (limit by ID | max NICK_SIZE | const ID)* src_comment

expr   : expr_comp ((AND | OR) comp)*
expr_comp   : expr ((GT | GTE | LT | LTE | EQ) expr)
expr_add   : term ((PLUS | MINUS) term)*
expr_mul   : factor ((MUL | DIV) factor)*
factor : (PLUS|MINUS|NOT) factor
		 | INTEGER
		 | STRING
		 | var_ref
		 | func_call
		 | new_op
		 | LPAREN expr RPAREN

*/

const (
	//symbol type
	symTypeVoid   = "void"
	symTypeInt    = "int"
	symTypeU1     = "u1"
	symTypeU2     = "u2"
	symTypeU3     = "u3"
	symTypeU4     = "u4"
	symTypeU5     = "u5"
	symTypeU6     = "u6"
	symTypeU7     = "u7"
	symTypeU8     = "u8"
	symTypeChar   = "char"
	symTypeU16    = "u16"
	symTypeU32    = "u32"
	symTypeU64    = "u64"
	symTypeV32    = "v32"
	symTypeV64    = "v64"
	symTypeFloat  = "float"
	symTypeString = "string"
	symTypeArray  = "array"
	symTypeAny    = "any"
	symTypeStruct = "struct"

	entryFunc = "main"
)

type hskParser struct {
	lex       *hskLexer
	prevToken *Token
	curToken  *Token

	lookAhead []*Token
	lookPrevs []*Token
	tpMap     map[string]AstType
	pos_ah    int
	markers   []int
	lastError error
}

func (p *hskParser) getLastError() error {
	return p.lastError
}

func (p *hskParser) is_speculating() bool {
	return len(p.markers) > 0
}

func (p *hskParser) peekToken() *Token {
	next := p.pos_ah + 1
	if next < len(p.lookAhead) {
		return p.lookAhead[next]
	}

	//fill look ahead buf
	token := p.lex.getNextToken()
	p.lookAhead = append(p.lookAhead, token)
	return token
}

func (p *hskParser) mark_push() {
	//fmt.Printf("push prev: %s:%d, cur: %s:%d\n", p.prevToken.value, p.prevToken.line, p.curToken.value, p.curToken.line)
	p.lookPrevs = append(p.lookPrevs, p.prevToken)
	p.markers = append(p.markers, p.pos_ah)
}

func (p *hskParser) mark_pop() {
	mark := p.markers[len(p.markers)-1]
	p.markers = p.markers[0 : len(p.markers)-1]

	p.curToken = p.lookAhead[mark]
	p.pos_ah = mark

	p.prevToken = p.lookPrevs[len(p.lookPrevs)-1]
	p.lookPrevs = p.lookPrevs[0 : len(p.lookPrevs)-1]
	p.lastError = nil

	//fmt.Printf("recover prev: %s:%d, cur: %s:%d\n", p.prevToken.value, p.prevToken.line, p.curToken.value, p.curToken.line)
}

func (p *hskParser) eat(ttype string) {
	if p.curToken.type_ == ttype {
		p.pos_ah++
		p.prevToken = p.curToken

		if p.pos_ah < len(p.lookAhead) {
			p.curToken = p.lookAhead[p.pos_ah]
		} else {
			p.curToken = p.lex.getNextToken()
			if p.is_speculating() {
				p.lookAhead = append(p.lookAhead, p.curToken)
			} else {
				p.pos_ah = 0
				p.lookAhead = p.lookAhead[0:1]
				p.lookAhead[0] = p.curToken
			}
		}

		if p.curToken == nil {
			p.panic(fmt.Sprintf("parse failed, last token: '%s', lexer error: %s", ttype, p.lex.getLastError()))
		}

	} else {
		p.panic(fmt.Sprintf("parse error, expect '%s', find: '%s:%s', line: %d", ttype, p.curToken.type_, p.curToken.value, p.curToken.line))
	}
}

func (p *hskParser) panic(format string, args ...interface{}) {
	desc := fmt.Sprintf(format, args...)
	p.lastError = errors.New(desc)
	panic(desc)
}

func (p *hskParser) eatSeperator() {
	if p.curToken.type_ == SEMI {
		p.eat(SEMI)
	} else {
		//check next token is in new line
		if p.curToken.type_ != EOF && p.prevToken != nil && p.prevToken.line == p.curToken.line {
			p.panic(fmt.Sprintf("missing seperator after '%s' line: %d", p.curToken.value, p.curToken.line))
		}
	}
}

//same as program
func (p *hskParser) Program() AstNode {
	return p.declarations()
}

func (p *hskParser) msg_bind() AstNode {
	p.eat(DEFBIND)
	p.eat(ID)

	ast := &AstBindDef{line: p.prevToken.line}
	ast.msgId = p.prevToken.value

	if p.curToken.type_ == NIL {
		p.eat(NIL)
		ast.msgName = ""
	} else {
		p.eat(ID)
		ast.msgName = p.prevToken.value
	}

	return ast
}

//const_decl | msg_decl
func (p *hskParser) declarations() AstNode {
	program := &AstProgram{}
	program.decl_list = []AstNode{}

	for p.curToken.type_ != EOF {
		if p.curToken.type_ == MSPACE {
			p.eat(MSPACE)
			p.eat(ID)
			program.mspace = p.prevToken.value
			p.eatSeperator()
		} else if p.curToken.type_ == CONST {
			ast := p.const_decl()
			p.eatSeperator()
			program.decl_list = append(program.decl_list, ast)
		} else if p.curToken.type_ == SCOMMENT {
			ast := &AstSrcComment{line: p.curToken.line, value: p.curToken.value}
			p.eat(SCOMMENT)
			program.decl_list = append(program.decl_list, ast)
		} else if p.curToken.type_ == TYPE {
			ast := p.type_def()
			program.decl_list = append(program.decl_list, ast)
		} else if p.curToken.type_ == DEFID {
			ast := p.id_decl()
			p.eatSeperator()
			program.decl_list = append(program.decl_list, ast)
		} else if p.curToken.type_ == DEFMID {
			ast := p.msgid_decl()
			p.eatSeperator()
			program.decl_list = append(program.decl_list, ast)
		} else if p.curToken.type_ == DEFBIND {
			ast := p.msg_bind()
			p.eatSeperator()
			program.decl_list = append(program.decl_list, ast)
		} else if p.curToken.type_ == EXTERN {
			p.eat(EXTERN)
			p.eat(ID)
			name := p.prevToken.value
			tp := p.type_spec()
			ast := &AstExternVar{line: p.prevToken.line, name: name, type_: tp}
			p.eatSeperator()
			program.decl_list = append(program.decl_list, ast)
		} else {
			ast := p.msg_decl()
			p.eatSeperator()

			//fmt.Printf("type def name: %s, type: %s\n", ast.name, ast.impl.signature())
			if old, ok := p.tpMap[ast.name]; ok {
				p.panic("duplicate type define, name: %s, type: %s, old type: %s", ast.name, ast.signature(), old.signature())
				return nil
			} else {
				p.tpMap[ast.name] = ast
			}

			program.decl_list = append(program.decl_list, ast)
		}
	}

	if p.lastError != nil {
		return nil
	}
	return program
}
func (p *hskParser) const_decl() AstNode {
	p.eat(CONST)
	p.eat(ID)
	name := p.prevToken.value
	line := p.prevToken.line
	expr := p.expr()

	ast := &AstConstDef{line: line, name: name, val: expr}
	return ast
}

func (p *hskParser) type_def() *AstTypeDef {
	/*
		type_def: TYPE ID type_ref
		type_ref:  (LBRACKET RBRACKET)* (INT | ID | STRING | struct_def)
		struct_def : STRUCT LBRACE fields_def type_spec RBRACE
		fields_def : (ID (COMMA ID)* COLON type_spec)*
	*/

	p.eat(TYPE)
	p.eat(ID)

	name := p.prevToken.value
	ast := &AstTypeDef{}
	ast.name = name
	ast.impl = p.type_spec()

	/* 	if strct, ok := ast.impl.(*AstStructType); ok {
		strct.name = name
	} */

	//fmt.Printf("type def name: %s, type: %s\n", ast.name, ast.impl.signature())
	if old, ok := p.tpMap[name]; ok {
		if ast.impl.astType() != AST_TP_UndefType && old.astType() == AST_TP_UndefType {
			//resolve it
			oldUndef := old.(*AstUndefType)
			if oldUndef.name == ast.name && oldUndef.resolved == nil {
				oldUndef.resolved = ast.impl
				return ast
			}
		}

		p.panic("duplicate type define, name: %s, type: %s, old type: %s", name, ast.impl.signature(), old.signature())
		return nil
	} else {
		p.tpMap[name] = ast.impl
	}

	return ast
}

//DEFID ID LBRACE (id_decl (COMMA id_decl)*)? RBRACE
func (p *hskParser) id_decl() *AstIdGroupDef {
	ast := &AstIdGroupDef{}
	p.eat(DEFID)
	ast.name = p.curToken.value
	ast.line = p.curToken.line
	p.eat(ID)
	p.eat(LBRACE)

	for p.curToken.type_ == SCOMMENT {
		p.eat(SCOMMENT)
		ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
	}

	if p.curToken.type_ == ID {
		ast.items = append(ast.items, p.msgidItem_decl())
		for {
			if p.curToken.type_ == COMMA {
				p.eat(COMMA)
				if p.curToken.type_ == SCOMMENT {
					p.eat(SCOMMENT)
					ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
				} else if p.curToken.type_ == RBRACE {
					break
				}
				ast.items = append(ast.items, p.msgidItem_decl())
			} else if p.curToken.type_ == SCOMMENT {
				p.eat(SCOMMENT)
				ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
			} else {
				break
			}
		}
	}

	for p.curToken.type_ == SCOMMENT {
		p.eat(SCOMMENT)
		ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
	}
	p.eat(RBRACE)
	return ast
}

func (p *hskParser) msgid_decl() *AstIdGroupDef {
	ast := &AstIdGroupDef{}
	p.eat(DEFMID)
	ast.isMsgId = true
	ast.name = p.curToken.value
	ast.line = p.curToken.line
	p.eat(ID)
	p.eat(LBRACE)

	for p.curToken.type_ == SCOMMENT {
		p.eat(SCOMMENT)
		ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
	}

	if p.curToken.type_ == ID {
		ast.items = append(ast.items, p.msgidItem_decl())
		for {
			if p.curToken.type_ == COMMA {
				p.eat(COMMA)
				if p.curToken.type_ == SCOMMENT {
					p.eat(SCOMMENT)
					ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
				} else if p.curToken.type_ == RBRACE {
					break
				}
				ast.items = append(ast.items, p.msgidItem_decl())
			} else if p.curToken.type_ == SCOMMENT {
				p.eat(SCOMMENT)
				ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
			} else {
				break
			}
		}
	}

	for p.curToken.type_ == SCOMMENT {
		p.eat(SCOMMENT)
		ast.notes = append(ast.notes, &AstSrcComment{line: p.prevToken.line, value: p.prevToken.value})
	}
	p.eat(RBRACE)
	return ast
}

func (p *hskParser) msgidItem_decl() *idItem {
	// ID (EQU INT_CONST)?
	item := &idItem{}
	item.name = p.curToken.value
	item.line = p.curToken.line

	p.eat(ID)
	if p.curToken.type_ == ASSIGN {
		p.eat(ASSIGN)
		p.eat(INT_CONST)
		item.base = true
		item.idVal = intConstVal(p.prevToken.value)
	}

	return item
}

func (p *hskParser) msg_decl() *AstStructType {
	p.eat(DEFMSG)
	p.eat(ID)

	name := p.prevToken.value
	line := p.prevToken.line
	ast := &AstStructType{name: name, line: line}
	p.eat(LBRACE)

	for p.curToken.type_ != RBRACE {
		if p.curToken.type_ == SCOMMENT {
			note := &AstSrcComment{line: p.curToken.line, value: p.curToken.value}
			p.eat(SCOMMENT)
			ast.notes = append(ast.notes, note)
		} else {
			ast.fields = append(ast.fields, p.field_decl())
		}
	}

	fmap := make(map[string]*AstVarDecl)

	for _, field := range ast.fields {
		if of, ok := fmap[field.name]; ok {
			doPanic("duplicate filed name: %s with type: %s, line: %d, prev type: %s, line: %d",
				field.name, field.type_.signature(), field.line, of.type_.signature(), of.line)
			break
		}
	}

	p.eat(RBRACE)
	return ast
}

//field_decl: ID type_spec (limit by ID | max NICK_SIZE | equal ID)* src_comment
func (p *hskParser) field_decl() *AstVarDecl {
	ast := &AstVarDecl{name: p.curToken.value, line: p.curToken.line}
	p.eat(ID)
	ast.type_ = p.type_spec()

	if p.curToken.type_ == DESC {
		p.eat(DESC)
		has := false
		for {
			if p.curToken.type_ == LIMIT {
				p.eat(LIMIT)
				p.eat(BY)
				token := p.curToken
				p.eat(ID)
				limAst := &AstVarNameRef{line: token.line, name: p.prevToken.value}
				ast.limit = limAst
				has = true
			} else if p.curToken.type_ == MAX {
				p.eat(MAX)
				token := p.curToken
				p.eat(ID)
				ast.max = &AstVarNameRef{line: token.line, name: p.prevToken.value}
				has = true
			} else if p.curToken.type_ == EQU {
				p.eat(EQU)
				token := p.curToken
				p.eat(ID)
				ast.equ = &AstVarNameRef{line: token.line, name: p.prevToken.value}
				has = true
			} else if p.curToken.type_ == XOR {
				p.eat(XOR)
				token := p.curToken
				p.eat(ID)
				ast.xor = &AstVarNameRef{line: token.line, name: p.prevToken.value}
				has = true
			} else if p.curToken.type_ == EXIST {
				p.eat(EXIST)

				if p.curToken.type_ == FOLLOW {
					p.eat(FOLLOW)
					p.eat(ABOVE)
					ast.existCondFollow = true
				} else {
					p.eat(IF)
					ast.existIf = p.expr()
				}

				has = true
			} else {
				break
			}
		}

		if !has {
			p.panic("expect desc terms, but recv: %s:%s, line: %d", p.curToken.type_, p.curToken.value, p.curToken.line)
			return nil
		}
	}

	if p.curToken.type_ == SCOMMENT {
		cmt := &AstSrcComment{line: p.curToken.line, value: p.curToken.value}
		ast.comment = cmt
		p.eat(SCOMMENT)
	}

	return ast
}

func (p *hskParser) type_spec() AstType {
	//type_spec : INT | STRING |  ID | LBRACKET RBRACKET type_spec

	if p.curToken.type_ == TYPE_INT {
		p.eat(TYPE_INT)
		return p.tpMap[symTypeInt]
	} else if p.curToken.type_ == TYPE_STRING {
		p.eat(TYPE_STRING)
		return p.tpMap[symTypeString]
	} else if p.curToken.type_ == TYPE_ANY {
		p.eat(TYPE_ANY)
		return p.tpMap[symTypeAny]
	} else if p.curToken.type_ == TYPE_U1 {
		p.eat(TYPE_U1)
		return p.tpMap[symTypeU1]
	} else if p.curToken.type_ == TYPE_U2 {
		p.eat(TYPE_U2)
		return p.tpMap[symTypeU2]
	} else if p.curToken.type_ == TYPE_U3 {
		p.eat(p.curToken.type_)
		return p.tpMap[symTypeU3]
	} else if p.curToken.type_ == TYPE_U4 {
		p.eat(TYPE_U4)
		return p.tpMap[symTypeU4]
	} else if p.curToken.type_ == TYPE_U5 {
		p.eat(p.curToken.type_)
		return p.tpMap[symTypeU5]
	} else if p.curToken.type_ == TYPE_U6 {
		p.eat(p.curToken.type_)
		return p.tpMap[symTypeU6]
	} else if p.curToken.type_ == TYPE_U7 {
		p.eat(p.curToken.type_)
		return p.tpMap[symTypeU7]
	} else if p.curToken.type_ == TYPE_U8 {
		p.eat(TYPE_U8)
		return p.tpMap[symTypeU8]
	} else if p.curToken.type_ == TYPE_CHAR {
		p.eat(TYPE_CHAR)
		return p.tpMap[symTypeChar]
	} else if p.curToken.type_ == TYPE_U16 {
		p.eat(TYPE_U16)
		return p.tpMap[symTypeU16]
	} else if p.curToken.type_ == TYPE_U32 {
		p.eat(TYPE_U32)
		return p.tpMap[symTypeU32]
	} else if p.curToken.type_ == TYPE_U64 {
		p.eat(TYPE_U64)
		return p.tpMap[symTypeU64]
	} else if p.curToken.type_ == TYPE_V32 {
		p.eat(TYPE_V32)
		return p.tpMap[symTypeV32]
	} else if p.curToken.type_ == TYPE_V64 {
		p.eat(TYPE_V64)
		return p.tpMap[symTypeV64]
	} else if p.curToken.type_ == LBRACKET {
		p.eat(LBRACKET)
		p.eat(RBRACKET)
		ast := &AstArrayType{}
		ast.elemType = p.type_spec()
		return ast
	} else if p.curToken.type_ == ID {
		p.eat(ID)
		tp := p.tpMap[p.prevToken.value]
		if tp != nil {
			return tp
		}

		ast := &AstUndefType{}
		ast.name = p.prevToken.value

		p.tpMap[ast.name] = ast
		return ast
	}

	p.panic("error type spec: %s, line: %d", p.curToken.value, p.curToken.line)
	return nil
}

/*
expr   : expr_and (OR expr_and)*
expr_and   : expr_equ (AND expr_equ)*
expr_equ   : expr_comp ((EQU | NEQ) expr_comp)
expr_comp   : expr_shift ((GT | GTE | LT | LTE) expr_shift)*
expr_shift :  expr_add ((LSHIFT | RSHIFT) expr_add)*
expr_add   : term ((PLUS | MINUS) term)*
expr_mul   : factor ((MUL | DIV) factor)*
factor : (PLUS | MINUS | NOT) factor | INT_CONST | LPAREN expr RPAREN
*/
func (p *hskParser) expr() AstNode {
	node := p.expr_and()

	for p.curToken.type_ == OR {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_and(), line: token.line}
	}

	return node
}

func (p *hskParser) expr_and() AstNode {
	node := p.expr_bitOr()

	for p.curToken.type_ == AND {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_bitOr(), line: token.line}
	}

	return node
}

func (p *hskParser) expr_bitOr() AstNode {
	node := p.expr_bitAnd()

	for p.curToken.type_ == BIT_OR {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_bitAnd(), line: token.line}
	}

	return node
}

func (p *hskParser) expr_bitAnd() AstNode {
	node := p.expr_equ()

	for p.curToken.type_ == BIT_AND {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_equ(), line: token.line}
	}

	return node
}

func (p *hskParser) expr_equ() AstNode {
	node := p.expr_comp()
	for p.curToken.type_ == EQU || p.curToken.type_ == NEQ {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_comp(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_comp() AstNode {
	node := p.expr_shift()
	for p.curToken.type_ == GT || p.curToken.type_ == GTE ||
		p.curToken.type_ == LT || p.curToken.type_ == LTE {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_shift(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_shift() AstNode {
	//expr_shift :  expr_add ((LSHIFT | RSHIFT) expr_add)*
	node := p.expr_add()
	for p.curToken.type_ == LSHIFT || p.curToken.type_ == RSHIFT {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_add(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_add() AstNode {
	node := p.expr_mul()
	for p.curToken.type_ == PLUS || p.curToken.type_ == MINUS {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_mul(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_mul() AstNode {
	node := p.factor()
	for p.curToken.type_ == MUL || p.curToken.type_ == DIV {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.factor(), line: token.line}
	}
	return node
}

func (p *hskParser) factor() AstNode {
	/*
		factor : (PLUS|MINUS|NOT) factor
				| INTEGER
				| STRING
				| var_ref
				| LPAREN expr RPAREN
	*/
	if p.curToken.type_ == PLUS || p.curToken.type_ == MINUS || p.curToken.type_ == NOT {
		ast := &AstUnaryOP{}
		ast.op = p.curToken.type_
		ast.line = p.curToken.line
		p.eat(p.curToken.type_)

		ast.dst = p.factor()
		return ast
	} else if p.curToken.type_ == INT_CONST {
		p.eat(INT_CONST)
		ast := &AstIntConst{value: intConstVal(p.prevToken.value)}
		return ast
	} else if p.curToken.type_ == STRING_CONST {
		p.eat(STRING_CONST)
		ast := &AstStringConst{value: p.prevToken.value}
		return ast
	} else if p.curToken.type_ == ID || p.curToken.type_ == THIS {
		ast := p.var_ref()
		return ast
	} else if p.curToken.type_ == LPAREN {
		p.eat(LPAREN)
		ast := p.expr()
		p.eat(RPAREN)
		return ast
	} else {
		msg := fmt.Sprintf("parse factor failed, cur token: '%s', line: %d", p.curToken.value, p.curToken.line)
		//fmt.Println(msg)
		p.panic(msg)
		return nil
	}
}

func (p *hskParser) var_ref() AstNode {
	//var_ref : ID (LBRACKET expr  RBRACKET | DOT ID)*
	this := false
	if p.curToken.type_ == THIS {
		this = true
		p.eat(THIS)
		p.eat(DOT)
	}

	var ast AstNode
	ast = &AstVarNameRef{this: this, name: p.curToken.value, line: p.curToken.line}
	p.eat(ID)

loop:
	for {
		switch p.curToken.type_ {
		case LBRACKET:
			top := &AstIndexedRef{}
			top.line = p.curToken.line
			top.host = ast
			p.eat(LBRACKET)
			top.index = p.expr()
			p.eat(RBRACKET)
			ast = top
			break

		case DOT:
			top := &AstDotRef{}
			top.line = p.curToken.line
			top.host = ast
			p.eat(DOT)
			top.name = p.curToken.value
			p.eat(ID)
			ast = top
			break

		default:
			break loop
		}
	}

	return ast
}

func isIntType(tp AstType) (bool, int) {
	switch ft := tp.(type) {
	case *AstPrimType:
		switch ft.name {
		case symTypeU1:
			return true, 1

		case symTypeU2:
			return true, 2

		case symTypeU3:
			return true, 3

		case symTypeU4:
			return true, 4

		case symTypeU5:
			return true, 5

		case symTypeU6:
			return true, 6

		case symTypeU7:
			return true, 7

		case symTypeU8:
			return true, 8

		case symTypeChar:
			return true, 8

		case symTypeU16:
			return true, 16

		case symTypeU32, symTypeV32:
			return true, 32

		case symTypeU64, symTypeV64:
			return true, 64
		}
	}

	return false, 0
}

func isVarInt(tp AstType) bool {
	switch ft := tp.(type) {
	case *AstPrimType:
		switch ft.name {
		case symTypeV32, symTypeV64:
			return true
		}
	}

	return false
}

var builtinTypeArr = []string{
	symTypeAny, symTypeInt,
	symTypeString, symTypeVoid,
	symTypeU1,
	symTypeU2,
	symTypeU3,
	symTypeU4,
	symTypeU5,
	symTypeU6,
	symTypeU7,
	symTypeU8, symTypeChar,
	symTypeU16, symTypeU32, symTypeU64,
	symTypeV32, symTypeV64,
}

func NewParser(text string) *hskParser {
	p := &hskParser{}
	p.lex = newLexer(text)

	p.curToken = p.lex.getNextToken()
	p.lookAhead = append(p.lookAhead, p.curToken)
	for p.curToken.type_ == LF {
		p.eat(LF)
	}

	p.tpMap = make(map[string]AstType)
	for _, val := range builtinTypeArr {
		p.tpMap[val] = &AstPrimType{name: val}
	}

	return p
}
