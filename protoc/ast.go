package protoc

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	AST_Program = iota + 1
	AST_INT_CONST
	AST_STRING_CONST
	AST_VarDecl
	AST_SrcComment
	AST_ConstDef
	AST_IdDef
	AST_BindDef
	AST_FuncDecl
	AST_BuiltinFunc
	AST_Assign
	AST_BinOp
	AST_NewOp
	AST_UnaryOp
	AST_Noop
	AST_CodeBlock
	AST_VarRef
	AST_ExternRef
	AST_DotRef
	AST_IndexedRef

	//data type
	AST_TP_Primitive
	AST_TP_Array
	AST_TP_Struct
	AST_TP_TypeDef
	AST_TP_TypeRef
	AST_TP_UndefType
	AST_TP_ExistIf
)

var verbPanic bool

func doPanic(format string, args ...interface{}) {
	desc := fmt.Sprintf(format, args...)
	if verbPanic {
		stack := string(debug.Stack())
		desc += "\nstacktrace: " + stack
	}
	panic(errors.New(desc))
}

type interpError struct {
	msg string
}

func (iterr *interpError) Error() string {
	return iterr.msg
}

func interpPanic(format string, args ...interface{}) {
	desc := fmt.Sprintf(format, args...)
	panic(&interpError{desc})
}

type AstNode interface {
	astType() int
	String() string
	desc() string
}

type AstBase struct {
	seq int
}

type AstProgram struct {
	AstBase
	mspace    string
	decl_list []AstNode
	tpMap     map[string]AstType
}

func (ast *AstProgram) astType() int {
	return AST_Program
}

func (ast *AstProgram) String() string {
	return fmt.Sprintf("AstProgram")
}

func (ast *AstProgram) desc() string {
	return fmt.Sprintf("not impl")
}

type AstConstDef struct {
	AstBase
	name string
	val  AstNode
	line int
}

func (ast *AstConstDef) astType() int {
	return AST_ConstDef
}

func (ast *AstConstDef) String() string {
	return fmt.Sprintf("AstConstDef")
}

func (ast *AstConstDef) desc() string {
	return fmt.Sprintf("const %s %s", ast.name, ast.val.desc())
}

type idItem struct {
	name    string
	bindMsg string
	isMsgId bool
	idVal   int
	base    bool
	line    int
}

type AstIdGroupDef struct {
	AstBase
	isMsgId bool
	name    string
	items   []*idItem
	notes   []*AstSrcComment
	line    int
}

func (ast *AstIdGroupDef) astType() int {
	return AST_IdDef
}

func (ast *AstIdGroupDef) String() string {
	return fmt.Sprintf("AstIdDef")
}

func (ast *AstIdGroupDef) desc() string {
	return fmt.Sprintf("idgroup, len: %d", len(ast.items))
}

type AstSrcComment struct {
	AstBase
	value string
	line  int
}

func (ast *AstSrcComment) astType() int {
	return AST_SrcComment
}

func (ast *AstSrcComment) String() string {
	return fmt.Sprintf("AstSrcComment")
}

func (ast *AstSrcComment) desc() string {
	return fmt.Sprintf("scomment: %s", ast.value)
}

type AstVarDecl struct {
	AstBase
	name            string
	type_           AstType
	limit           *AstVarNameRef
	max             *AstVarNameRef
	min             *AstVarNameRef
	equ             *AstVarNameRef
	xor             *AstVarNameRef
	existIf         AstNode
	existCondFollow bool
	dlim            bool
	comment         *AstSrcComment
	line            int
}

func (ast *AstVarDecl) astType() int {
	return AST_VarDecl
}

func (ast *AstVarDecl) String() string {
	return fmt.Sprintf("AstVarDecl")
}

func (ast *AstVarDecl) desc() string {
	return fmt.Sprintf("var %s:%s", ast.name, ast.type_.signature())
}

type AstBinOP struct {
	AstBase
	op    string
	left  AstNode
	right AstNode
	lval  int
	rval  int
	line  int
}

func (ast *AstBinOP) astType() int {
	return AST_BinOp
}

func (ast *AstBinOP) String() string {
	return fmt.Sprintf("AstBinOP")
}

func (ast *AstBinOP) desc() string {
	return fmt.Sprintf("(%s) %s (%s)", ast.left.desc(), ast.op, ast.right.desc())
}

type AstNewOP struct {
	AstBase
	opType AstType
	line   int
}

func (ast *AstNewOP) astType() int {
	return AST_NewOp
}

func (ast *AstNewOP) String() string {
	return fmt.Sprintf("AstNewOP")
}

func (ast *AstNewOP) desc() string {
	return fmt.Sprintf("new: %s", ast.opType.desc())
}

type AstUnaryOP struct {
	AstBase
	op   string
	dst  AstNode
	line int
}

func (ast *AstUnaryOP) astType() int {
	return AST_UnaryOp
}

func (ast *AstUnaryOP) String() string {
	return fmt.Sprintf("AstUnaryOP")
}

func (ast *AstUnaryOP) desc() string {
	return fmt.Sprintf("%s%s", ast.op, ast.dst.desc())
}

type AstReturn struct {
	AstBase
	expr AstNode
	line int
}

func (ast *AstReturn) astType() int {
	return AST_UnaryOp
}

func (ast *AstReturn) String() string {
	return fmt.Sprintf("AstReturn")
}

func (ast *AstReturn) desc() string {
	return fmt.Sprintf("return: %s", ast.expr.desc())
}

type AstNoopStat struct {
}

func (ast *AstNoopStat) astType() int {
	return AST_Noop
}

func (ast *AstNoopStat) String() string {
	return fmt.Sprintf("AstNoopStat")
}

func (ast *AstNoopStat) desc() string {
	return fmt.Sprintf("not impl")
}

type AstIntConst struct {
	AstBase
	value int
	line  int
}

func (ast *AstIntConst) astType() int {
	return AST_STRING_CONST
}

func (ast *AstIntConst) String() string {
	return fmt.Sprintf("AstIntConst")
}

func (ast *AstIntConst) desc() string {
	return fmt.Sprintf("int const: %d", ast.value)
}

type AstStringConst struct {
	AstBase
	value string
	line  int
}

func (ast *AstStringConst) astType() int {
	return AST_STRING_CONST
}

func (ast *AstStringConst) String() string {
	return fmt.Sprintf("AstStringConst")
}

func (ast *AstStringConst) desc() string {
	return fmt.Sprintf("string const: %s", ast.value)
}

type AstVarNameRef struct {
	AstBase
	this bool
	name string
	line int
	lvl  int
}

func (ast *AstVarNameRef) astType() int {
	return AST_VarRef
}

func (ast *AstVarNameRef) String() string {
	return fmt.Sprintf("AstVarRef")
}

func (ast *AstVarNameRef) desc() string {
	return fmt.Sprintf("%s", ast.name)
}

type AstExternVar struct {
	AstBase
	name  string
	type_ AstType
	line  int
	lvl   int
}

func (ast *AstExternVar) astType() int {
	return AST_ExternRef
}

func (ast *AstExternVar) String() string {
	return fmt.Sprintf("AstExternNameRef")
}

func (ast *AstExternVar) desc() string {
	return fmt.Sprintf("%s", ast.name)
}

type AstDotRef struct {
	AstBase
	host AstNode
	name string
	line int
}

func (ast *AstDotRef) astType() int {
	return AST_DotRef
}

func (ast *AstDotRef) String() string {
	return fmt.Sprintf("AstDotRef")
}

func (ast *AstDotRef) desc() string {
	return fmt.Sprintf("%s.%s", ast.host.desc(), ast.name)
}

type AstIndexedRef struct {
	AstBase
	host  AstNode
	index AstNode
	line  int
}

func (ast *AstIndexedRef) astType() int {
	return AST_IndexedRef
}

func (ast *AstIndexedRef) String() string {
	return fmt.Sprintf("AstIndexedRef")
}

func (ast *AstIndexedRef) desc() string {
	return fmt.Sprintf("%s[%s]", ast.host.desc(), ast.index.desc())
}

type AstTypeDef struct {
	AstBase
	name string
	impl AstType
}

func (ast *AstTypeDef) astType() int {
	return AST_TP_TypeDef
}

func (ast *AstTypeDef) String() string {
	return fmt.Sprintf("AstTypeDef")
}

func (ast *AstTypeDef) desc() string {
	return fmt.Sprintf("not impl")
}

type AstExistIf struct {
	AstBase
	line    int
	selfVar *AstVarNameRef
	op      string
	varRef  *AstVarNameRef
}

func (ast *AstExistIf) astType() int {
	return AST_TP_ExistIf
}

func (ast *AstExistIf) String() string {
	return fmt.Sprintf("AstExistIf")
}

func (ast *AstExistIf) desc() string {
	return fmt.Sprintf("not impl")
}

type AstType interface {
	AstNode
	signature() string
}

type AstPrimType struct {
	name string
}

func (ast *AstPrimType) astType() int {
	return AST_TP_Primitive
}

func (ast *AstPrimType) String() string {
	return fmt.Sprintf("AstPrimType: " + ast.name)
}

func (ast *AstPrimType) signature() string {
	switch ast.name {
	case symTypeAny:
		return "*"

	case symTypeVoid:
		return "V"

	case symTypeInt,
		symTypeU1,
		symTypeU2,
		symTypeU4,
		symTypeU8,
		symTypeU16,
		symTypeU32,
		symTypeU64,
		symTypeV32,
		symTypeV64:
		return "I"

	case symTypeString:
		return "S"

	default:
		doPanic("unknown primitive type: " + ast.name)
	}
	return "-"
}

func (ast *AstPrimType) desc() string {
	return ast.name
}

func newPrimType(name string) *AstPrimType {
	return &AstPrimType{name: name}
}

type AstArrayType struct {
	elemType AstType
}

func (ast *AstArrayType) astType() int {
	return AST_TP_Array
}

func (ast *AstArrayType) String() string {
	return fmt.Sprintf("AstArrayType")
}

func (ast *AstArrayType) signature() string {
	return "[" + ast.elemType.signature()
}

func (ast *AstArrayType) desc() string {
	return "[]" + ast.elemType.desc()
}

type AstStructType struct {
	AstBase
	name   string
	fields []*AstVarDecl
	notes  []*AstSrcComment
	line   int
}

func (ast *AstStructType) astType() int {
	return AST_TP_Struct
}

func (ast *AstStructType) String() string {
	return fmt.Sprintf("AstStructType: " + ast.name)
}

func (ast *AstStructType) signature() string {
	return "s" + ast.name + ";"
}

func (ast *AstStructType) desc() string {
	return "struct: " + ast.name
}

type AstUndefType struct {
	name     string
	resolved AstType
}

func (ast *AstUndefType) astType() int {
	return AST_TP_UndefType
}

func (ast *AstUndefType) String() string {
	return fmt.Sprintf("AstUndefType: " + ast.name)
}

func (ast *AstUndefType) signature() string {
	if ast.resolved != nil {
		return ast.resolved.signature()
	} else {
		return "?" + ast.name + ";"
	}
}

func (ast *AstUndefType) desc() string {
	if ast.resolved != nil {
		return ast.resolved.desc()
	} else {
		return "undefine: " + ast.name
	}
}

func realType(tp AstType) AstType {
	if rtype, ok := tp.(*AstUndefType); ok {
		return rtype.resolved
	}

	return tp
}

func intConstVal(str string) int {
	var val uint64
	if strings.HasPrefix(str, "0x") {
		val, _ = strconv.ParseUint(str[2:], 16, 64)
	} else {
		val, _ = strconv.ParseUint(str, 10, 64)
	}

	return int(val)
}

func currentTimeString() string {
	t := time.Now()
	return t.Format("2006-01-02 15:04:05")
}

func getLimitFieldMax(node *AstStructType, name string) *AstVarNameRef {
	for _, f := range node.fields {
		if f.name == name {
			return f.max
		}
	}

	return nil
}

type AstBindDef struct {
	AstBase
	msgName string
	msgId   string
	line    int
}

func (ast *AstBindDef) astType() int {
	return AST_BindDef
}

func (ast *AstBindDef) String() string {
	return fmt.Sprintf("AstBindDef")
}

func (ast *AstBindDef) desc() string {
	return fmt.Sprintf("bind, %s -> %s", ast.msgName, ast.msgId)
}
