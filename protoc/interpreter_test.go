package protoc

import (
	"fmt"
	"io/ioutil"
	"path"
	"testing"
)

func TestInterp(t *testing.T) {
	file := "../data/test.proto"
	body, _ := ioutil.ReadFile(file)
	program := string(body)

	//fmt.Printf("%s", program)
	p := NewParser(program)
	analyzer := NewSemanticAnalyzer()

	pro := p.Program()
	err := analyzer.DoAnalyze(pro)
	fmt.Printf("analyze result: %v\n", err)
	if err != nil {
		t.Errorf("analyze error: %v\n", err)
		return
	}

	interp := NewInterpreter()
	interp.Mode = INTERP_MODE_GO
	interp.SrcFile = path.Base(file)
	err = interp.DoInterpret(pro)

	if err != nil {
		t.Errorf("interpret error: %v\n", err)
	}

	// for _, va := range interp.curFrame.table {
	// 	fmt.Printf("%s %s: %v\n", va.name, va.type_, va.val)
	// }
}
