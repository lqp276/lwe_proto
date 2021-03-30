package protoc

import (
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/pkg/errors"
)

const ()

type symbolTable struct {
	table   map[string]symbolClass
	level   int
	upLevel *symbolTable
}

func newSymTable(level int, upLevel *symbolTable) *symbolTable {
	tb := &symbolTable{level: level, upLevel: upLevel}
	tb.table = make(map[string]symbolClass)
	for _, val := range builtinTypeArr {
		tb.table[val] = newBuiltinSymbol(val)
	}

	return tb
}

func (tb *symbolTable) insertSymbol(sym symbolClass, debug bool) {
	if _, ok := tb.table[sym.symName()]; ok {
		doPanic("duplicate symbol: '%s'", sym.symName())
	}

	if debug {
		fmt.Printf("insert symbol: %s level: %d\n", sym, tb.level)
	}
	tb.table[sym.symName()] = sym
}

func (tb *symbolTable) lookup(name string, chained bool) symbolClass {
	if sym, ok := tb.table[name]; ok {
		return sym
	}

	if chained && tb.upLevel != nil {
		return tb.upLevel.lookup(name, chained)
	}

	return nil
}

type symbolClass interface {
	symName() string
	String() string
	level() int
}

type symbol struct {
	name  string
	cate  string
	type_ AstType
	lvl   int
}

type builtinSymbol struct {
	symbol
}

func (sym *builtinSymbol) String() string {
	return fmt.Sprintf("builtinSymbol{%s %s}", sym.name, sym.type_)
}

func (sym *builtinSymbol) symName() string {
	return sym.name
}

func (sym *builtinSymbol) level() int {
	return 0
}

func newBuiltinSymbol(name string) *builtinSymbol {
	sym := &builtinSymbol{}
	sym.name = name
	sym.type_ = nil
	sym.lvl = 0
	return sym
}

type varSymbol struct {
	symbol
	line int
}

func (sym *varSymbol) String() string {
	return fmt.Sprintf("varSymbol{%s %s @line %d}", sym.name, sym.type_, sym.line)
}

func (sym *varSymbol) symName() string {
	return sym.name
}

func (sym *varSymbol) level() int {
	return sym.lvl
}

func newVarSymbol(name string, type_ AstType, lvl int, line int) *varSymbol {
	sym := &varSymbol{}
	sym.name = name
	if rtype, ok := type_.(*AstUndefType); ok {
		sym.type_ = rtype.resolved
	} else {
		sym.type_ = type_
	}
	sym.lvl = lvl
	sym.line = line
	return sym
}

type semanticAnalyzer struct {
	symbolStack    []*symbolTable
	stackSize      int
	curSymbolTable *symbolTable
	firstPass      bool
	debug          bool
	brkStack       []bool
	midMap         map[string]*idItem
}

func (p *semanticAnalyzer) pushBrk() {
	p.brkStack = append(p.brkStack, true)
}

func (p *semanticAnalyzer) popBrk() {
	p.brkStack = p.brkStack[:len(p.brkStack)-1]
}

func (p *semanticAnalyzer) allowBrk() bool {
	return len(p.brkStack) > 0
}

func (se *semanticAnalyzer) pushSymbolTable() *symbolTable {
	symTb := newSymTable(se.stackSize, se.curSymbolTable)
	se.symbolStack = append(se.symbolStack, symTb)
	se.stackSize += 1
	se.curSymbolTable = symTb
	return se.curSymbolTable
}

func (se *semanticAnalyzer) popSymbolTable() *symbolTable {
	se.symbolStack = se.symbolStack[:len(se.symbolStack)-1]
	se.stackSize -= 1
	se.curSymbolTable = se.symbolStack[len(se.symbolStack)-1]
	return se.curSymbolTable
}

func (se *semanticAnalyzer) visitProgram(program *AstProgram) {
	//se.resolveTypes(program)

	if program.mspace == "" {
		doPanic("mid space not specified")
	}

	//just inflate symbol table with type symbol
	for _, decl := range program.decl_list {
		switch node := decl.(type) {
		case *AstVarDecl:
			if se.firstPass {
				se.visitVarDecl(node)
			}
			break

		case *AstIdGroupDef:
			if se.firstPass {
				se.visitIdGroupDefine(node)
			}
			break

		case *AstConstDef:
			if se.firstPass {
				se.visitConstDef(node)
			}
			break

		case *AstStructType:
			if se.firstPass {
				se.visitMsgDefine(node)
			}
			break

		case *AstExternVar:
			if se.firstPass {
				se.visitExternVar(node)
			}
			break

		case *AstBindDef:
			if !se.firstPass {
				se.visitBindDef(node)
			}

		case *AstSrcComment:
			break

		case *AstTypeDef:
			break

		default:
			doPanic("unsupported ast type in program: %T", node)
		}
	}

	se.firstPass = false
}

func (se *semanticAnalyzer) visitVarDecl(node *AstVarDecl) {
	if sym := se.curSymbolTable.lookup(node.name, false); sym != nil {
		doPanic("error var symbol defined in level: %d, name: %s, type: %s, already exist: %s",
			se.curSymbolTable.level, node.name, node.type_, sym.symName())
		return
	}

	if primTp, ok := node.type_.(*AstPrimType); ok {
		if tp := se.curSymbolTable.lookup(primTp.name, true); tp == nil {
			doPanic("variable type not defined in level: %d, name: %s", se.curSymbolTable.level, node.type_)
			return
		}
	}

	//ok
	sym := newVarSymbol(node.name, node.type_, se.curSymbolTable.level, node.line)
	se.curSymbolTable.insertSymbol(sym, se.debug)
}

func (se *semanticAnalyzer) visitConstDef(node *AstConstDef) {
	if sym := se.curSymbolTable.lookup(node.name, false); sym != nil {
		doPanic("error symbol defined in level: %d, name: %s, line: %d, type: %s, already exist: %s",
			se.curSymbolTable.level, node.name, node.line, "AstConstDef", sym.symName())
		return
	}

	tp := se.visitAst(node.val).(AstType)

	//ok
	sym := newVarSymbol(node.name, tp, se.curSymbolTable.level, node.line)
	se.curSymbolTable.insertSymbol(sym, se.debug)
}

func (se *semanticAnalyzer) visitIdGroupDefine(node *AstIdGroupDef) {
	if len(node.items) == 0 {
		return
	}

	if sym := se.curSymbolTable.lookup(node.name, false); sym != nil {
		doPanic("error symbol defined in level: %d, name: %s, line: %d, type: %s, already exist: %s",
			se.curSymbolTable.level, node.name, node.line, "AstIdGroupDef", sym.symName())
		return
	}

	tp := &AstPrimType{name: symTypeInt}

	//ok
	sym := newVarSymbol(node.name, tp, se.curSymbolTable.level, node.line)
	se.curSymbolTable.insertSymbol(sym, se.debug)

	val := 0
	for i, id := range node.items {
		id.isMsgId = node.isMsgId
		if oid, ok := se.midMap[id.name]; ok {
			doPanic("id already defined, name: %s, orig line: %d, here: %d", id.name, oid.line, id.line)
		} else {
			se.midMap[id.name] = id
		}
		if id.base {
			if i > 0 && id.idVal <= val {
				doPanic(" \"%s\":id -> %d must be great than \"%s\" -> %s",
					id.name, id.idVal, node.items[i-1].name, node.items[i-1].name)
			}
			val = id.idVal
		} else {
			val++
			id.idVal = val
		}
	}
}

func (se *semanticAnalyzer) visitMsgDefine(node *AstStructType) {
	if sym := se.curSymbolTable.lookup(node.name, false); sym != nil {
		doPanic("error symbol defined in level: %d, name: %s, line: %d, type: %s, already exist: %s",
			se.curSymbolTable.level, node.name, node.line, "AstIdGroupDef", sym.symName())
		return
	}

	//ok
	sym := newVarSymbol(node.name, node, se.curSymbolTable.level, node.line)
	se.curSymbolTable.insertSymbol(sym, se.debug)

	visit := func(ast AstNode, name string) {
		if ast == nil || reflect.ValueOf(ast).IsNil() {
			return
		}

		if se.visitAst(ast).(AstType).signature() != "I" {
			doPanic("visit msg define error, '%s' should be type int", name)
			return
		}
	}

	se.pushSymbolTable()
	inAggr := false
	bits := 0
	for _, f := range node.fields {
		xorOk := true
		ok, bn := isIntType(f.type_)
		if inAggr {
			xorOk = false
			if !ok {
				doPanic("fields in aggregate, but follow filed: \"%s\" line: %d is not int", f.name, f.line)
			} else {
				bits += bn
				if bits < 8 {
					//go on
				} else if bits == 8 {
					bits = 0
					inAggr = false
				} else {
					doPanic("fields in aggregate, but field series not fit in 8 bits boundary: \"%s\" line: %d", f.name, f.line)
				}
			}
		} else if ok {
			if bn%8 != 0 {
				bits = bn
				inAggr = true
			}
		} else {
			//not int
			xorOk = false
			inAggr = false
			bits = 0
		}

		se.visitAst(f)
		if f.type_.astType() == AST_TP_Array {
			if f.limit == nil {
				doPanic("\"%s\" must limited by one field or const, line: %d", f.name, f.line)
				return
			}
		}

		if f.existIf != nil {
			if ok {
				//do more check?
				if !xorOk {
					doPanic("not allow exist if in bit field, name: %s, line: %d", f.name, f.line)
				}
			}
		}

		visit(f.equ, "equal")
		visit(f.limit, "limit")
		visit(f.max, "max")
		visit(f.min, "min")
		if f.xor != nil {
			if isVarInt(f.type_) {
				doPanic("var int and xor are exclusive, line: %d", f.line)
			}

			if xorOk {
				visit(f.xor, "xor")
			} else {
				doPanic("fields xor not in 8 bit boundary, field: \"%s\" line: %d", f.name, f.line)
			}
		}
	}

	se.popSymbolTable()
}

func isAnyType(ast AstNode) bool {
	if tp, ok := ast.(*AstPrimType); ok {
		if tp.name == symTypeAny {
			return true
		}
	}

	return false
}

func isCompatiWithPrimType(primTp *AstPrimType, has interface{}) bool {
	if primTp.name == symTypeAny {
		return true
	}

	return false
}

func isTypeCompatiable(want string, has interface{}) bool {
	if has == nil {
		return false
	}

	strHas, ok := has.(string)
	if !ok {
		return false
	}

	p1 := newSigParser(want)
	p2 := newSigParser(strHas)

	for {
		s1 := p1.getNextElem()
		s2 := p2.getNextElem()

		if s1 == nil && s2 == nil {
			return true
		}

		if s1 == nil {
			return false
		}

		if s1.tp == symTypeAny {
			return true
		}

		//require exact match
		if s1.tp != s2.tp {
			return false
		}

		//same type, check value
		if s1.value != s2.value {
			return false
		}
	}
}

func (se *semanticAnalyzer) visitBinOP(node *AstBinOP) interface{} {
	var lhs AstType
	var rhs AstType
	switch node.left.(type) {
	case *AstBinOP, *AstUnaryOP, *AstDotRef, *AstIndexedRef,
		*AstStringConst, *AstIntConst, *AstVarNameRef:
		lhs = se.visitAst(node.left).(AstType)
		break

	default:
		doPanic("error in binop left, unknown ast type: %s, line: %d", node.left, node.line)
	}

	switch node.right.(type) {
	case *AstBinOP, *AstUnaryOP, *AstDotRef, *AstIndexedRef,
		*AstStringConst, *AstIntConst, *AstVarNameRef:
		rhs = se.visitAst(node.right).(AstType)
		break

	default:
		doPanic("error in binop right, unknown ast type: %s, line: %d", node.right, node.line)
	}

	if lhs.signature() != rhs.signature() {
		doPanic("assign with incompatiable type, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
		return nil
	}

	sp := newSigParser(lhs.signature())
	first := sp.getNextElem()
	if first == nil {
		doPanic("get signature elem error, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
		return nil
	}

	switch first.tp {
	case symTypeInt:
		break

	case symTypeVoid, symTypeAny, symTypeArray, symTypeStruct:
		doPanic("error binop on type: %s, lhs: %s, rhs: %s, line: %d", first.tp, lhs, rhs, node.line)
		break

	case symTypeString:
		if node.op != PLUS {
			doPanic("string type only allow add, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
		}
		break

	default:
		doPanic("error binop on type: %s, lhs: %s, rhs: %s, line: %d", first.tp, lhs, rhs, node.line)
	}

	return lhs
}

func (se *semanticAnalyzer) visitUnaryOP(node *AstUnaryOP) interface{} {
	var rhs AstType
	switch node.dst.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef:
		rhs = se.visitAst(node.dst).(AstType)
		break

	default:
		doPanic("error in unaryop dst, unknown ast type: %s, line: %d", node.dst, node.line)
	}

	return rhs
}

func (se *semanticAnalyzer) visitIntConst(node *AstIntConst) interface{} {
	return &AstPrimType{name: symTypeInt}
}

func (se *semanticAnalyzer) visitStringConst(node *AstStringConst) interface{} {
	return &AstPrimType{name: symTypeString}
}

func (se *semanticAnalyzer) visitIndexedRef(node *AstIndexedRef) interface{} {
	idxTp := se.visitAst(node.index)
	primTp, ok := idxTp.(*AstPrimType)
	if !ok || primTp.name != symTypeInt {
		doPanic("error in indexedRef: %s, index should be int, actual: %s",
			node.host, primTp)
		return nil
	}

	//check host
	hostTp := se.visitAst(node.host)
	arrTp, ok := hostTp.(*AstArrayType)
	if !ok {
		doPanic("error in indexedRef: %s, host should be array, actual: %s",
			node.host, arrTp)
		return nil
	}

	return arrTp.elemType
}

func (se *semanticAnalyzer) visitDotRef(node *AstDotRef) interface{} {
	hType := se.visitAst(node.host)
	strctTp, ok := hType.(*AstStructType)
	if !ok {
		doPanic("error in dotRef: %s, host should be struct, actual: %s",
			node.host, hType)
		return nil
	}

	//check host
	for _, field := range strctTp.fields {
		if field.name == node.name {
			return realType(field.type_)
		}
	}

	doPanic("visit dotRef error, struct %s has no field: %s", strctTp.name, node.name)
	return nil
}

func (se *semanticAnalyzer) visitVarRef(node *AstVarNameRef) interface{} {
	sym := se.curSymbolTable.lookup(node.name, true)
	if sym == nil {
		doPanic("error in varRef, symbol not found: %s, line: %d", node.name, node.line)
		return nil
	}

	if varSym, ok := sym.(*varSymbol); ok {
		return realType(varSym.type_)
	} else {
		doPanic("error in varRef, name: %s, line: %d, %T", node.name, node.line, sym)
		return nil
	}
}

func (se *semanticAnalyzer) visitExternVar(node *AstExternVar) {
	if sym := se.curSymbolTable.lookup(node.name, false); sym != nil {
		doPanic("error var symbol defined in level: %d, name: %s, line: %d, type: %s, already exist: %s",
			se.curSymbolTable.level, node.name, node.line, "AstConstDef", sym.symName())
		return
	}

	tp := node.type_

	//ok
	sym := newVarSymbol(node.name, tp, se.curSymbolTable.level, node.line)
	se.curSymbolTable.insertSymbol(sym, se.debug)
}

func (se *semanticAnalyzer) visitBindDef(node *AstBindDef) {
	if len(node.msgName) == 0 {
		return
	}

	if sym := se.curSymbolTable.lookup(node.msgName, false); sym == nil {
		doPanic("error bind, msg name: %s not found at line: %d", node.msgName, node.line)
		return
	}

	//find dst mid
	id, ok := se.midMap[node.msgId]
	if !ok {
		doPanic("msg bind error at line: %d, msg: %s, mid: %s, mid not defined yet",
			node.line, node.msgName, node.msgId)
		return
	}

	if !id.isMsgId {
		doPanic("msg bind error at line: %d, msg: %s, mid: %s, mid is not msg id",
			node.line, node.msgName, node.msgId)
		return
	}

	if len(id.bindMsg) != 0 {
		doPanic("msg bind error at line: %d, msg: %s, mid: %s already bind to: %s",
			node.line, node.msgName, node.msgId, id.bindMsg)
		return
	}
	id.bindMsg = node.msgName
}

func (se *semanticAnalyzer) resolveTypes(pro *AstProgram) {
	//add user defined type to type table

	for {
		fixArr := []*AstUndefType{}
		for k, v := range pro.tpMap {
			switch node := v.(type) {
			case *AstPrimType, *AstArrayType, *AstStructType:
				//fmt.Printf("skip resolve type: %s\n", node)
				break

			case *AstUndefType:
				if node.resolved != nil {
					continue
				}

				if rs, ok := pro.tpMap[node.name]; ok {
					if rs.astType() != AST_TP_UndefType {
						node.resolved = rs
						//fmt.Printf("-->resolve type: %s -> %s, ref by: %s\n", node.name, rs, k)
						fixArr = append(fixArr, node)
					} else {
						if undefTp := rs.(*AstUndefType); undefTp.resolved != nil {
							//fmt.Printf("==>resolve type: %s -> %s, ref by: %s\n", node.name, rs, k)
							node.resolved = undefTp.resolved
							fixArr = append(fixArr, node)
						}
					}
				} else {
					doPanic("undefined type, name: %s", node.name)
				}
				break

			default:
				doPanic("unsupported ast type in type map: %T, name: %s", node, k)
			}
		}

		if len(fixArr) == 0 {
			break
		}
	}

	for k, v := range pro.tpMap {
		switch node := v.(type) {
		case *AstUndefType:
			if node.resolved == nil {
				doPanic("unresolved type, name: %s, ref by: %s", node.name, k)
			}

			//fmt.Printf("indirect collected type: %s -> %s\n", k, node.resolved)
			break

		default:
			//fmt.Printf("collected type: [%s] %s\n", k, node)
		}
	}
}

func (se *semanticAnalyzer) DoAnalyze(root AstNode) (result error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			desc := r.(error).Error() + "\n" + stack
			result = errors.New(desc)
		}
	}()

	switch node := root.(type) {
	case *AstProgram:
		se.visitProgram(node)
		//second pass
		se.visitProgram(node)
		break

	default:
		return errors.Errorf("root ast type should be program, actual recv: %T", root)
	}

	return nil
}

func (se *semanticAnalyzer) visitAst(ast AstNode) interface{} {
	switch statement := ast.(type) {
	case *AstVarDecl:
		se.visitVarDecl(statement)
		break

	case *AstBinOP:
		return se.visitBinOP(statement)

	case *AstUnaryOP:
		return se.visitUnaryOP(statement)

	case *AstStringConst:
		return se.visitStringConst(statement)

	case *AstIntConst:
		return se.visitIntConst(statement)

	case *AstVarNameRef:
		return se.visitVarRef(statement)

	case *AstIndexedRef:
		return se.visitIndexedRef(statement)

	case *AstDotRef:
		return se.visitDotRef(statement)

	default:
		doPanic("unknown ast type: %T", ast)
		break
	}

	return nil
}

func NewSemanticAnalyzer() *semanticAnalyzer {
	se := &semanticAnalyzer{}

	symTb := newSymTable(0, nil)
	se.symbolStack = []*symbolTable{symTb}
	se.stackSize = 1
	se.curSymbolTable = se.symbolStack[0]
	se.brkStack = []bool{}
	se.midMap = make(map[string]*idItem)
	se.firstPass = true
	return se
}
