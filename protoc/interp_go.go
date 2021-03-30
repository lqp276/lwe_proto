package protoc

import (
	"fmt"
	"strings"
	"unicode"
)

func (interp *interpreter) visitIdGroupDefine_Go(node *AstIdGroupDef) {
	interp.addLine("const (")
	interp.pushStackFrame()

	var notes []*AstSrcComment
	if len(node.notes) > 0 {
		notes = node.notes[:]
	}

	for idx, id := range node.items {
		if id.base && idx > 0 {
			interp.addNewLine()
		}

		for len(notes) > 0 && id.line > notes[0].line {
			interp.addLine("//" + notes[0].value)
			notes = notes[1:]
		}

		interp.addLine("%s = %d //hex: 0x%x", nameForMsg(id.name), id.idVal, id.idVal)
	}
	interp.popStackFrame()
	interp.addLine(")")

	interp.idGroupName_Go(node)
}

func (interp *interpreter) idGroupName_Go(node *AstIdGroupDef) {
	interp.addNewLine()
	interp.addLine("func %s_name(id uint16)(string, bool) {", node.name)
	interp.pushStackFrame()

	interp.addLine("switch id {")
	for idx, id := range node.items {
		if idx != 0 {
			interp.addNewLine()
		}
		interp.addLine("case %s:", id.name)
		interp.pushStackFrame()
		interp.addLine("return \"%s\", true", id.name)
		interp.popStackFrame()
	}

	interp.addLine("}")
	interp.addLine("return \"\", false")
	interp.popStackFrame()
	interp.addLine("}")
}

func (interp *interpreter) intConstComment_go(val int, node AstNode) string {
	if tp, ok := node.(*AstBinOP); ok {
		if tp.op == LSHIFT {
			return fmt.Sprint("(", tp.lval, "<<", tp.rval, ")")
		} else if tp.op == RSHIFT {
			return fmt.Sprint("(", tp.lval, ">>", tp.rval, ")")
		}
	}

	return fmt.Sprintf("0x%x", val)
}

func (interp *interpreter) visitConstDef_Go(node *AstConstDef) {
	tp := interp.visitAst(node.val)
	switch val := tp.(type) {
	case int:
		interp.addLine("const %s = %d //%s", node.name, val,
			interp.intConstComment_go(val, node.val))

	case string:
		interp.addLine("const %s = \"%s\"", node.name, val)

	default:
		doPanic("unknown val type: %s:%T", val, val)
	}
}

func nameForMsg(name string) string {
	if true {
		return name
	}

	nextUp := true

	carr := []rune(name)
	tarr := []rune{}
	for _, ch := range carr {
		if nextUp {
			tarr = append(tarr, unicode.ToUpper(ch))
			nextUp = false
		} else if ch == '_' {
			nextUp = true
		} else {
			tarr = append(tarr, ch)
		}
	}

	return string(tarr)
}

func typeName4Go(tp AstType) string {
	switch ft := tp.(type) {
	case *AstPrimType:
		switch ft.name {
		case symTypeU1, symTypeU2, symTypeU3,
			symTypeU4, symTypeU5, symTypeU6, symTypeU7, symTypeU8, symTypeChar:
			return "uint8"

		case symTypeU16:
			return "uint16"

		case symTypeU32, symTypeV32:
			return "uint32"

		case symTypeU64, symTypeV64:
			return "uint64"
		}

	case *AstStructType:
		return fmt.Sprintf("%s", ft.name)

	case *AstUndefType:
		return fmt.Sprintf("%s", ft.name)
	}

	doPanic("unsupported type: %s", tp)
	return ""
}

func (interp *interpreter) visitMsgEncode_Go(node *AstStructType) {
	node.name = nameForMsg(node.name)
	interp.addLine("func encode_%s(buf io.Writer, m *%s) int {", node.name, typeName4Go(node))
	interp.pushStackFrame()

	var notes []*AstSrcComment
	if len(node.notes) > 0 {
		notes = node.notes[:]
	}

	hasTmp := false
	bitAggr := false
	bits := 0
	var xorVar *AstVarNameRef
	for _, f := range node.fields {
		for len(notes) > 0 && f.line > notes[0].line {
			interp.addLine("//" + notes[0].value)
			notes = notes[1:]
		}

		switch ft := f.type_.(type) {
		case *AstPrimType:
			ok, bn := isIntType(ft)
			mask := 1<<bn - 1
			if bitAggr {
				if !ok {
					doPanic("msg encode in bit assemble mode error, hasTmp: %v, bitAggr: %v, bits: %d, bn: %d",
						hasTmp, bitAggr, bits, bn)
				}

				//go on assemble to byte
				bits += bn
				if bits <= 8 {
					shift := 8 - bits
					if shift == 0 {
						interp.addLine("tmp |= m.%s & 0x%x", f.name, mask)
					} else {
						interp.addLine("tmp |= (m.%s & 0x%x) << %d", f.name, mask, shift)
					}

					if bits == 8 {
						if xorVar != nil {
							tpAst := &AstPrimType{name: symTypeU8}
							interp.addLine("tmp ^= %s(%s)", typeName4Go(tpAst), xorVar.name)
						}

						interp.addLine("if binary.Write(buf, binary.BigEndian, tmp) != nil { return -1 }")
						interp.addNewLine()
						bitAggr = false
						xorVar = nil
					}
				} else {
					doPanic("msg encode in bit assemble mode error, bits aggregate num: %d", bits)
				}
			} else if ok, bn := isIntType(ft); ok {
				if in := bn / 8; bn%8 == 0 {
					switch in {
					case 1, 2, 4, 8:
						interp.wrapExist_Go(f, func() {
							if isVarInt(ft) {
								doPanic("var int encode is not supported in golang default style")
								return
							}

							if f.max != nil {
								interp.addNewLine()
								interp.addLine("if m.%s > %s { m.%s = %s} ", f.name, f.max.name, f.name, f.max.name)
							}
							//interp.addLine("byte_buf_put_u%d(buf,  m->%s);", in*8, f.name)
							if f.xor == nil {
								interp.addLine("if binary.Write(buf, binary.BigEndian, m.%s) != nil { return -1 }", f.name)
							} else {
								interp.addLine("if binary.Write(buf, binary.BigEndian, m.%s^%s(%s)) != nil { return -1 }", f.name, typeName4Go(ft), f.xor.name)
							}
						})
					default:
						doPanic("msg encode not support int, bytes: %d", in)
					}
				} else {
					if !hasTmp {
						hasTmp = true
						interp.addLine("tmp := uint8(0)")
					} else {
						interp.addLine("tmp = 0")
					}

					xorVar = f.xor
					bits = bn
					interp.addLine("tmp |= (m.%s & 0x%x) << %d", f.name, mask, 8-bits)
					bitAggr = true
				}

			} else {
				doPanic("msg encode not support non int types")
			}

		case *AstStructType:
			interp.wrapExist_Go(f, func() {
				interp.addLine("if encode_%s(buf, &m.%s) < 0 { return -1 }", ft.name, f.name)
			})

		case *AstArrayType:
			interp.wrapExist_Go(f, func() {
				if ut, ok := ft.elemType.(*AstPrimType); ok {
					if ok, bn := isIntType(ut); ok && bn == 8 {
						interp.addNewLine()
						if getLimitFieldMax(node, f.limit.name) != nil {
							interp.addLine("if binary.Write(buf, binary.BigEndian, m.%s[0:m.%s]) != nil { return -1 }", f.name, f.limit.name)
						} else {
							interp.addLine("if binary.Write(buf, binary.BigEndian, m.%s[0:%s]) != nil { return -1 }", f.name, f.limit.name)
						}
						return
					}
				}

				//gen for loop
				if getLimitFieldMax(node, f.limit.name) != nil {
					interp.addLine("for i := 0; i < int(m.%s); i++ {", f.limit.name)
				} else {
					interp.addLine("for i := 0; i < int(%s); i++ {", f.limit.name)
				}
				interp.pushStackFrame()
				switch et := ft.elemType.(type) {
				case *AstPrimType:
					ok, bn := isIntType(et)
					if !ok || bn%8 != 0 {
						doPanic("msg encode not support non int type or type int of bits not div by 8")
					} else {
						interp.addLine("if binary.Write(buf, binary.BigEndian, m.%s[i]) != nil { return -1 }", f.name)
					}

				case *AstStructType:
					interp.addLine("if encode_%s(buf, &m.%s[i]) < 0 { return -1 }", et.name, f.name)

				default:
					doPanic("unsupported array elem type encode: %s %s", f.name, ft)
				}
				interp.popStackFrame()
				interp.addLine("}")
				interp.addNewLine()
			})

		case *AstUndefType:
			interp.wrapExist_Go(f, func() {
				interp.addLine("if encode_%s(buf, &m.%s) < 0 { return -1 }", ft.name, f.name)
			})

		default:
			doPanic("encode unsupported type: %s %s", f.name, ft)
		}
	}

	interp.addLine("return 0")
	interp.popStackFrame()
	interp.addLine("}")
}

func (interp *interpreter) visitMsgDecode_Go(node *AstStructType) {
	interp.addLine("func decode_%s(buf io.Reader, m *%s) int {", node.name, typeName4Go(node))
	interp.pushStackFrame()

	hasTmp := false
	bitAggr := false
	bits := 0
	for _, f := range node.fields {
		switch ft := f.type_.(type) {
		case *AstPrimType:
			ok, bn := isIntType(ft)
			if bitAggr {
				mask := 1<<bn - 1
				if !ok {
					doPanic("msg decode in bit assemble mode error, hasTmp: %v, bitAggr: %v, bits: %d, bn: %d",
						hasTmp, bitAggr, bits, bn)
				}

				//go on assemble to byte
				bits += bn
				if bits <= 8 {
					shift := 8 - bits
					if shift > 0 {
						interp.addLine("m.%s = (tmp >> %d) & 0x%x;", f.name, 8-bits, mask)
					} else {
						interp.addLine("m.%s = tmp & 0x%x;", f.name, mask)
					}

					if f.equ != nil {
						interp.addLine("if m.%s != %s return -1; ", f.name, f.equ.name)
					}

					if bits == 8 {
						bitAggr = false
						bits = 0
					}
				} else {
					doPanic("msg decode in bit assemble mode error, bits aggregate num: %d", bits)
				}
			} else if ok, bn := isIntType(ft); ok {
				if in := bn / 8; bn%8 == 0 {
					switch in {
					case 1, 2, 4, 8:
						interp.wrapExist_Go(f, func() {
							if isVarInt(ft) {
								doPanic("var int encode is not supported in golang default style")
								return
							}

							interp.addLine("if binary.Read(buf, binary.BigEndian, &m.%s) != nil { return -1 }", f.name)
							if f.xor != nil {
								interp.addLine("m.%s ^= %s(%s)", f.name, typeName4Go(f.type_), f.xor.name)
							}

							if f.max != nil {
								interp.addLine("if m.%s > %s { return -1 }", f.name, f.max.name)
							} else if f.equ != nil {
								interp.addLine("if m.%s != %s { return -1 }", f.name, f.equ.name)
							}
						})

					default:
						doPanic("msg decode not support int, bytes: %d", in)
					}
				} else {
					if !hasTmp {
						hasTmp = true
						interp.addLine("tmp := uint8(0)")
					} else {
						interp.addNewLine()
						interp.addLine("tmp = 0")
					}

					bits = bn
					mask := (1 << bn) - 1
					interp.addLine("if binary.Read(buf, binary.BigEndian, &tmp) != nil { return -1 }")
					if f.xor != nil {
						tpAst := &AstPrimType{name: symTypeU8}
						interp.addLine("tmp ^= %s(%s)", typeName4Go(tpAst), f.xor.name)
					}

					interp.addLine("m.%s = (tmp >> %d) & 0x%x", f.name, 8-bits, mask)
					if f.equ != nil {
						interp.addLine("if m.%s != %s { return -1 }", f.name, f.equ.name)
					}
					bitAggr = true
				}

			} else {
				doPanic("msg decode not support non int types")
			}

		case *AstStructType:
			interp.wrapExist_Go(f, func() {
				interp.addLine("if decode_%s(buf, &m.%s) < 0 { return -1 }", ft.name, f.name)
			})

		case *AstArrayType:
			interp.wrapExist_Go(f, func() {
				if ut, ok := ft.elemType.(*AstPrimType); ok {
					if ok, bn := isIntType(ut); ok && bn == 8 {
						if getLimitFieldMax(node, f.limit.name) != nil {
							interp.addLine("if binary.Read(buf, binary.BigEndian, m.%s[:m.%s]) != nil { return -1 }", f.name, f.limit.name)
						} else {
							interp.addLine("if binary.Read(buf, binary.BigEndian, m.%s[:]) != nil { return -1 }", f.name)
						}
						return
					}
				}

				//gen for loop
				if getLimitFieldMax(node, f.limit.name) != nil {
					interp.addLine("for i := 0; i < int(m.%s); i++ {", f.limit.name)
				} else {
					interp.addLine("for i := 0; i < int(%s); i++ {", f.limit.name)
				}

				interp.pushStackFrame()
				switch et := ft.elemType.(type) {
				case *AstPrimType:
					ok, bn := isIntType(et)
					if !ok || bn%8 != 0 {
						doPanic("msg decode not support non int type or type int of bits not div by 8")
					} else {
						interp.addLine("if binary.Read(buf, binary.BigEndian, &m.%s[i]) != nil { return -1 }", f.name)
					}

				case *AstStructType:
					interp.addLine("if decode_%s(buf, &m.%s[i]) < 0 { return -1 }", et.name, f.name)

				default:
					doPanic("unsupported array elem type decode: %s %s", f.name, ft)
				}
				interp.popStackFrame()
				interp.addLine("}")
				interp.addNewLine()
			})

		case *AstUndefType:
			interp.wrapExist_Go(f, func() {
				interp.addLine("if decode_%s(buf, &m.%s) < 0 { return -1 }", ft.name, f.name)
			})

		default:
			doPanic("decode unsupported type: %s %s", f.name, ft)
		}
	}

	interp.addLine("return 0")
	interp.popStackFrame()
	interp.addLine("}")
}

func (interp *interpreter) visitMsgCodec_Go(node *AstStructType) {
	interp.visitMsgEncode_Go(node)
	interp.addLine("")
	interp.visitMsgDecode_Go(node)
}

func (interp *interpreter) visitMsgDefine_Go(node *AstStructType) {

	interp.addLine("")
	interp.addLine("type %s struct {", node.name)
	interp.pushStackFrame()
	for _, f := range node.fields {
		switch ft := f.type_.(type) {
		case *AstPrimType:
			if f.comment != nil {
				interp.addLine("%s %s //%s %s", f.name, typeName4Go(ft), ft.name, f.comment.value)
			} else {
				interp.addLine("%s %s //%s", f.name, typeName4Go(ft), ft.name)
			}
		case *AstStructType, *AstUndefType:
			if f.comment != nil {
				interp.addLine("%s %s //%s", f.name, typeName4Go(ft), f.comment.value)
			} else {
				interp.addLine("%s %s", f.name, typeName4Go(ft))
			}

		case *AstArrayType:
			lm := getLimitFieldMax(node, f.limit.name)
			if lm != nil {
				interp.addLine("%s [%s]%s", f.name, lm.name, typeName4Go(ft.elemType))
			} else {
				interp.addLine("%s [%s]%s", f.name, f.limit.name, typeName4Go(ft.elemType))
			}

		default:
			doPanic("unsupported type: %s %s", f.name, ft)
		}
	}
	interp.popStackFrame()
	interp.addLine("}")
	interp.addLine("")

	interp.visitMsgCodec_Go(node)
}

func (interp *interpreter) visitTypeDef_Go(node *AstTypeDef) {
	interp.addLine("type %s %s;", node.name, typeName4Go(node.impl))

	if node.impl.astType() == AST_TP_Struct {
		impl := node.impl.(*AstStructType)
		oname := impl.name
		impl.name = node.name
		interp.visitMsgCodec_Go(impl)
		node.name = oname
	}
}

func visitVarRef_Go(ref *AstVarNameRef) string {
	if ref.this {
		return fmt.Sprintf("m.%s", ref.name)
	}

	return fmt.Sprintf("%s", ref.name)
}

func visitBinOP_Go(op *AstBinOP) string {
	switch op.op {
	case BIT_AND:
		return "&"

	case BIT_OR:
		return "|"

	case AND:
		return "&&"

	case OR:
		return "||"

	case EQU:
		return "=="

	case NEQ:
		return "!="

	case LT:
		return "<"

	case LTE:
		return "<="

	case GT:
		return ">"

	case GTE:
		return ">="
	}

	return "??"
}

func (interp *interpreter) visitBindEncode_Go(binds []*AstBindDef) {
	interp.addNewLine()
	mspace := interp.program.mspace
	interp.addLine(fmt.Sprint("func encode", strings.ToUpper(mspace[:1]), mspace[1:], "MsgById(buf io.Writer, mid uint16, msg interface{}) int {"))
	interp.pushStackFrame()

	interp.addLine("switch mid {")
	for idx, bind := range binds {
		if idx != 0 {
			interp.addNewLine()
		}
		interp.addLine("case %s:", bind.msgId)
		interp.pushStackFrame()
		if len(bind.msgName) != 0 {
			interp.addLine("return encode_%s(buf, msg.(*%s))", bind.msgName, bind.msgName)
		} else {
			interp.addLine("return 0")
		}
		interp.popStackFrame()
	}
	interp.addLine("}")

	interp.addNewLine()
	interp.addLine("return -1")
	interp.popStackFrame()
	interp.addLine("}")
	interp.addNewLine()
}

func (interp *interpreter) visitBindDecode_Go(binds []*AstBindDef) {
	interp.addNewLine()
	mspace := interp.program.mspace
	interp.addLine(fmt.Sprint("func decode", strings.ToUpper(mspace[:1]), mspace[1:], "MsgById(buf io.Reader, mid uint16, msg interface{}) int {"))
	interp.pushStackFrame()

	interp.addLine("switch mid {")
	for idx, bind := range binds {
		if idx != 0 {
			interp.addNewLine()
		}
		interp.addLine("case %s:", bind.msgId)
		interp.pushStackFrame()
		if len(bind.msgName) != 0 {
			interp.addLine("return decode_%s(buf, msg.(*%s))", bind.msgName, bind.msgName)
		} else {
			interp.addLine("return 0")
		}
		interp.popStackFrame()
	}
	interp.addLine("}")

	interp.addNewLine()
	interp.addLine("return -1")
	interp.popStackFrame()
	interp.addLine("}")
	interp.addNewLine()
}

func (interp *interpreter) visitBinds_Go(binds []*AstBindDef) {
	interp.visitBindEncode_Go(binds)
	interp.visitBindDecode_Go(binds)
}
