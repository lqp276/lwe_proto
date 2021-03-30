package protoc

import (
	"fmt"
	"runtime/debug"

	"github.com/pkg/errors"
)

const (
	INTERP_MODE_GO = iota + 1
	INTERP_MODE_C
)

const (
	INTERP_VERSION = "version 1.0; author lqp"
)

func assert(cond bool) {
	if !cond {
		panic("cond assert failed!!!")
	}
}

type stackFrame struct {
	table   map[string]*vari
	level   int
	upLevel *stackFrame
	retVal  interface{}
	state   byte
	interp  *interpreter
}

type sourceCodeWriter interface {
	addNewLine()
	addLine(format string, args ...interface{})
	pushScope()
	popSope()
}

type memCodeWriter struct {
	lines []string
	scope int
}

func (w *memCodeWriter) reset() {
	w.lines = nil
	w.scope = 0
}

func (w *memCodeWriter) pushScope() {
	w.scope++
}

func (w *memCodeWriter) popSope() {
	w.scope--
}

func (w *memCodeWriter) addNewLine() {
	w.addLine("")
}

func (w *memCodeWriter) addLine(format string, args ...interface{}) {
	head := ""
	space := w.scope * 4
	for i := 0; i < space; i++ {
		head += " "
	}
	desc := fmt.Sprintf(head+format, args...)
	w.lines = append(w.lines, desc)
}

func (interp *interpreter) addNewLine() {
	if interp.lastNewLine {
		return
	}

	interp.addLine("")
	interp.lastNewLine = true
}

func (interp *interpreter) addLine(format string, args ...interface{}) {
	head := ""
	space := (len(interp.callStack) - 1) * 4
	for i := 0; i < space; i++ {
		head += " "
	}
	desc := fmt.Sprintf(head+format+"\n", args...)
	fmt.Printf(desc)
	interp.lastNewLine = false
}

func makeFrame(interp *interpreter, level int, upLevel *stackFrame) *stackFrame {
	tb := &stackFrame{level: level, upLevel: upLevel}
	tb.table = make(map[string]*vari)
	tb.interp = interp
	return tb
}

func (tb *stackFrame) insertVari(v *vari) {
	if tb.interp.debug {
		fmt.Printf("set variable: %v stack idx: %d\n", v, tb.level)
	}
	tb.table[v.name] = v
}

func (tb *stackFrame) lookup(name string, chained bool) *vari {
	if sym, ok := tb.table[name]; ok {
		return sym
	}

	if chained && tb.upLevel != nil {
		return tb.upLevel.lookup(name, chained)
	}

	return nil
}

type varClass interface {
	symName() string
	String() string
}

type vari struct {
	name  string
	type_ AstNode
	val   interface{}
}

func newIntVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	va.val = 0
	return va
}

func newStringVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	va.val = ""
	return va
}

func newStructVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	mv := make(map[string]interface{})

	strctTp := va.type_.(*AstStructType)
	for _, field := range strctTp.fields {
		switch ftp := field.type_.(type) {
		case *AstPrimType:
			switch ftp.name {
			case symTypeInt:
				mv[field.name] = 0
				break

			case symTypeString:
				mv[field.name] = ""
				break
			}

			break

		default:
			mv[field.name] = nil
			break
		}
	}

	va.val = mv
	return va
}

func newArrayVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	arrTp := ast.type_.(*AstArrayType)

	switch elem := arrTp.elemType.(type) {
	case *AstPrimType:
		switch elem.name {
		case symTypeInt:
			valArr := []interface{}{}
			va.val = valArr
			break

		case symTypeString:
			valArr := []interface{}{}
			va.val = valArr
			break

		default:
			doPanic("newArrayVari error, unknown primitive elem type: %T", elem)
		}

	case *AstArrayType, *AstStructType:
		va.val = []interface{}{}
		break

	default:
		doPanic("newArrayVari error, unknown elem type: %T", elem)
	}

	return va
}

const maxBufLine = 3

type interpreter struct {
	callStack   []*stackFrame
	stackSize   int
	curFrame    *stackFrame
	debug       bool
	Mode        int
	SrcFile     string
	lastNewLine bool
	binds       []*AstBindDef
	program     *AstProgram
}

func (interp *interpreter) pushStackFrame() *stackFrame {
	symTb := makeFrame(interp, interp.stackSize, interp.curFrame)
	interp.callStack = append(interp.callStack, symTb)
	interp.stackSize++
	interp.curFrame = symTb
	interp.lastNewLine = true
	return interp.curFrame
}

func (interp *interpreter) popStackFrame() *stackFrame {
	popFrame := interp.curFrame
	interp.callStack = interp.callStack[:len(interp.callStack)-1]
	interp.stackSize--
	interp.curFrame = interp.callStack[len(interp.callStack)-1]
	interp.curFrame.state = popFrame.state
	return interp.curFrame
}

func (w *interpreter) pushScope() {
	w.pushStackFrame()
}

func (w *interpreter) popSope() {
	w.popStackFrame()
}

func (interp *interpreter) hasMsgDefine(program *AstProgram) bool {
	for _, decl := range program.decl_list {
		switch decl.(type) {
		case *AstStructType:
			return true

		default:
			continue
		}
	}

	return false
}

func (interp *interpreter) visitTraverse(program *AstProgram) {
	for _, decl := range program.decl_list {
		switch node := decl.(type) {
		case *AstVarDecl:
			interp.visitVarDecl(node)
			break

		case *AstIdGroupDef:
			interp.visitIdGroupDefine(node)
			break

		case *AstConstDef:
			interp.visitConstDef(node)
			break

		case *AstSrcComment:
			interp.addLine("//" + node.value)
			break

		case *AstStructType:
			interp.visitMsgDefine(node)
			break

		case *AstTypeDef:
			interp.visitTypeDef(node)
			break

		case *AstExternVar:
			break

		case *AstBindDef:
			interp.visitBindDef(node)

		default:
			doPanic("unsupported ast type in program: %T", node)
		}
	}
}

type visitVarRef func(ref *AstVarNameRef) string
type visitBinOP func(ref *AstBinOP) string

func (interp *interpreter) traveseCond(top bool, node AstNode, refOp visitVarRef, binOp visitBinOP) string {
	switch ast := node.(type) {
	case *AstVarNameRef:
		return refOp(ast)

	case *AstBinOP:
		if top {
			return fmt.Sprintf("%s %s %s", interp.traveseCond(false, ast.left, refOp, binOp), binOp(ast), interp.traveseCond(false, ast.right, refOp, binOp))
		} else {
			return fmt.Sprintf("(%s %s %s)", interp.traveseCond(false, ast.left, refOp, binOp), binOp(ast), interp.traveseCond(false, ast.right, refOp, binOp))
		}

	case *AstIntConst:
		return fmt.Sprint(ast.value)

	default:
		return "??"
	}
}

func (interp *interpreter) wrapExist_Go(f *AstVarDecl, op func()) {
	if f.existIf != nil {
		interp.addLine("if %s {", interp.traveseCond(true, f.existIf, visitVarRef_Go, visitBinOP_Go))
		interp.pushStackFrame()
	}

	op()

	if f.existIf != nil {
		interp.popStackFrame()
		interp.addLine("}")
	}
}

func (interp *interpreter) visitProgram(program *AstProgram) {
	mode := ""
	switch interp.Mode {
	case INTERP_MODE_GO:
		mode = "golang"

	case INTERP_MODE_C:
		mode = "c single file"

	default:
		mode = "unknown mode"
	}
	interp.addLine("/*")
	interp.addLine(" * code auto generated from: %s @%s, Do NOT touch by hand!!!", interp.SrcFile, currentTimeString())
	interp.addLine(" * generator %s; mode: %s", INTERP_VERSION, mode)
	interp.addLine("*/")
	interp.program = program

	interp.visitTraverse(program)
	interp.visitBinds()
}

func (interp *interpreter) visitIdGroupDefine(node *AstIdGroupDef) {
	if interp.Mode == INTERP_MODE_GO {
		interp.visitIdGroupDefine_Go(node)
	}
}

func (interp *interpreter) visitBindDef(node *AstBindDef) {
	interp.binds = append(interp.binds, node)
}

func (interp *interpreter) visitBinds() {
	if len(interp.binds) == 0 {
		return
	}

	if interp.Mode == INTERP_MODE_GO {
		interp.visitBinds_Go(interp.binds)
	}
}

func (interp *interpreter) visitTypeDef(node *AstTypeDef) {
	if interp.Mode == INTERP_MODE_GO {
		interp.visitTypeDef_Go(node)
	}
}

func (interp *interpreter) visitConstDef(node *AstConstDef) {
	if interp.Mode == INTERP_MODE_GO {
		interp.visitConstDef_Go(node)
	}
}

func (interp *interpreter) visitMsgDefine(node *AstStructType) {
	if interp.Mode == INTERP_MODE_GO {
		interp.visitMsgDefine_Go(node)
	}
}

func (interp *interpreter) visitVarDecl(node *AstVarDecl) {
	if sym := interp.curFrame.lookup(node.name, false); sym != nil {
		doPanic("error variable inited in level: %d, name: %s, type: %s, already exist: %s",
			interp.curFrame.level, node.name, node.type_, sym.name)
		return
	}

	//ok
	switch tp := node.type_.(type) {
	case *AstPrimType:
		switch tp.name {
		case symTypeInt:
			va := newIntVari(interp.curFrame.level, node)
			interp.curFrame.insertVari(va)
			break

		case symTypeString:
			va := newStringVari(interp.curFrame.level, node)
			interp.curFrame.insertVari(va)
			break

		case symTypeVoid:
			break

		default:
			doPanic("unknown AstPrimType when interpret: %s", node.type_)
		}
		break

	case *AstStructType:
		va := newStructVari(interp.curFrame.level, node)
		interp.curFrame.insertVari(va)
		break

	case *AstArrayType:
		va := newArrayVari(interp.curFrame.level, node)
		interp.curFrame.insertVari(va)
		break

	default:
		doPanic("unknown type when interpret: %s", node.type_)

	}
}

func (interp *interpreter) setAstVal(dst AstNode, val interface{}) {
	switch rTp := dst.(type) {
	case *AstIndexedRef:
		if ret := interp.visitAst(rTp.host); ret != nil {
			arr := ret.([]interface{})
			idx := interp.visitAst(rTp.index).(int)
			arr[idx] = val
		} else {
			interpPanic("hskl runtime error, nil reference: %s, line: %d", rTp.host.desc(), rTp.line)
		}
		break

	case *AstVarNameRef:
		sym := interp.curFrame.lookup(rTp.name, true)
		if sym == nil {
			doPanic("error in varRef, symbol not found: %s", rTp.name)
		}
		sym.val = val
		break

	case *AstDotRef:
		if ret := interp.visitAst(rTp.host); ret != nil {
			mv := ret.(map[string]interface{})
			mv[rTp.name] = val
		} else {
			interpPanic("hskl runtime error, nil reference: %s, line: %d", rTp.host.desc(), rTp.line)
		}

		break

	default:
		doPanic("error ast in setAstVal: %T", dst)
	}
}

func (interp *interpreter) visitBinOP(node *AstBinOP) interface{} {
	var lhs interface{}
	var rhs interface{}

	switch node.left.(type) {
	case *AstBinOP, *AstUnaryOP, *AstStringConst, *AstIntConst, *AstVarNameRef:
		lhs = interp.visitAst(node.left)
		break

	default:
		doPanic("error in binop left, unknown ast type: %s, line: %d", node.left, node.line)
	}

	switch node.right.(type) {
	case *AstBinOP, *AstUnaryOP, *AstStringConst, *AstIntConst, *AstVarNameRef:
		rhs = interp.visitAst(node.right)
		break

	default:
		doPanic("error in binop right, unknown ast type: %s, line: %d", node.right, node.line)
	}

	if s1, ok := lhs.(string); ok {
		//must be string add
		s2 := rhs.(string)
		return s1 + s2
	}

	lhv := lhs.(int)
	rhv := rhs.(int)

	node.lval = lhv
	node.rval = rhv
	//fmt.Printf("test: %d %s %d\n", lhv, node.op, rhv)
	switch node.op {
	case PLUS:
		return lhv + rhv

	case MINUS:
		return lhv - rhv

	case MUL:
		return lhv * rhv

	case DIV:
		if rhv != 0 {
			return lhv / rhv
		} else {
			interpPanic("div by zero: %s, line: %d", node.desc(), node.line)
		}

	case LSHIFT:
		return lhv << rhv

	case AND:
		if lhv == 0 {
			return lhv
		} else {
			return rhv
		}

	case OR:
		if lhv != 0 {
			return lhv
		} else {
			return rhv
		}

	case NOT:
		if lhv == 0 {
			return 1
		} else {
			return 0
		}

	case EQU:
		if lhv == rhv {
			return 1
		} else {
			return 0
		}

	case NEQ:
		if lhv != rhv {
			return 1
		} else {
			return 0
		}

	case LT:
		if lhv < rhv {
			return 1
		} else {
			return 0
		}
	case LTE:
		if lhv <= rhv {
			return 1
		} else {
			return 0
		}
	case GT:
		if lhv > rhv {
			return 1
		} else {
			return 0
		}
	case GTE:
		if lhv >= rhv {
			return 1
		} else {
			return 0
		}
	}

	return 0
}

func (interp *interpreter) visitUnaryOP(node *AstUnaryOP) interface{} {
	var rhs interface{}
	switch node.dst.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef:
		rhs = interp.visitAst(node.dst)
		break

	default:
		doPanic("error in unaryop dst, unknown ast type: %s, line: %d", node.dst, node.line)
	}

	intVal := rhs.(int)
	switch node.op {
	case PLUS:
		return rhs

	case MINUS:
		return -intVal

	case NOT:
		if intVal == 0 {
			return 1
		} else {
			return 0
		}

	default:
		doPanic("unknown unary operator: %s, line: %d", node.op, node.line)
	}

	return nil
}

func (se *interpreter) visitNewOP(node *AstNewOP) interface{} {
	switch tp := realType(node.opType).(type) {
	case *AstStructType:
		return make(map[string]interface{})

	case *AstArrayType:
		return []interface{}{}

	default:
		doPanic("error type when interpret new op: %s, line: %d", tp.desc(), node.line)
	}

	return nil
}

func (interp *interpreter) visitReturn(node *AstReturn) interface{} {
	if node.expr != nil {
		return interp.visitAst(node.expr)
	}

	return nil
}

func (interp *interpreter) visitIntConst(node *AstIntConst) interface{} {
	return node.value
}

func (interp *interpreter) visitStringConst(node *AstStringConst) interface{} {
	return node.value
}

func (interp *interpreter) visitIndexedRef(node *AstIndexedRef) interface{} {
	idxTp := interp.visitAst(node.index)
	primTp, ok := idxTp.(int)
	if !ok {
		interpPanic("hskl runtime error, nil reference: %s, line: %d", node.host.desc(), node.line)
		return nil
	}

	//check host
	hostTp := interp.visitAst(node.host)
	if hostTp == nil {
		interpPanic("hskl runtime error, nil array reference, line: %d", node.line)
		return nil
	}

	arrTp, ok := hostTp.([]interface{})
	if !ok {
		doPanic("error in indexedRef: %s, host should be array, actual: %T",
			node.host, arrTp)
		return nil
	}

	if primTp < len(arrTp) {
		return arrTp[primTp]
	}
	doPanic("error in indexedRef: %v, index out of bound, index: %d, arr len: %d",
		primTp, primTp, len(arrTp))
	return nil
}

func (interp *interpreter) visitDotRef(node *AstDotRef) interface{} {
	hType := interp.visitAst(node.host) //should return map
	if hType == nil {
		interpPanic("hskl runtime error, nil reference: %s, line: %d", node.host.desc(), node.line)
		return nil
	}

	mv, ok := hType.(map[string]interface{})
	if !ok {
		doPanic("error in dotRef: %s, host should be map[string]interface{}, actual: %T",
			node.host, mv)
		return nil
	}

	//check host
	return mv[node.name]
}

func (interp *interpreter) visitVarRef(node *AstVarNameRef) interface{} {
	sym := interp.curFrame.lookup(node.name, true)
	if sym == nil {
		doPanic("error in varRef, symbol not found: %s", node.name)
		return nil
	}

	return sym.val
}

func (interp *interpreter) DoInterpret(root AstNode) (result error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			desc := r.(error).Error() + "\n" + stack
			if rte, ok := r.(*interpError); ok {
				result = errors.New(rte.msg)
			} else {
				result = errors.New(desc)
			}
		}
	}()

	switch node := root.(type) {
	case *AstProgram:
		interp.visitProgram(node)
		break

	default:
		return errors.Errorf("root ast type should be program, actual recv: %T", root)
	}

	return nil
}

func (interp *interpreter) visitAst(ast AstNode) interface{} {
	//fmt.Printf("visit ast: %T\n", ast)
	switch statement := ast.(type) {
	case *AstVarDecl:
		interp.visitVarDecl(statement)
		break

	case *AstBinOP:
		return interp.visitBinOP(statement)

	case *AstUnaryOP:
		return interp.visitUnaryOP(statement)

	case *AstNewOP:
		return interp.visitNewOP(statement)

	case *AstReturn:
		return interp.visitReturn(statement)

	case *AstIntConst:
		return interp.visitIntConst(statement)

	case *AstStringConst:
		return interp.visitStringConst(statement)

	case *AstVarNameRef:
		return interp.visitVarRef(statement)

	case *AstIndexedRef:
		return interp.visitIndexedRef(statement)

	case *AstDotRef:
		return interp.visitDotRef(statement)

	case nil:
		return nil

	default:
		doPanic("unknown ast when interpret: %T", ast)
		break
	}

	return nil
}

func NewInterpreter() *interpreter {
	inter := &interpreter{}

	symTb := makeFrame(inter, 0, nil)
	inter.callStack = []*stackFrame{symTb}
	inter.stackSize = 1
	inter.curFrame = inter.callStack[0]
	return inter
}
