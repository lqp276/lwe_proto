package protoc

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestSemantic(t *testing.T) {
	body, _ := ioutil.ReadFile("../data/test.proto")
	program := string(body)

	p := NewParser(program)
	analyzer := NewSemanticAnalyzer()

	pro := p.declarations()
	fmt.Printf("%T\n", pro)
	err := analyzer.DoAnalyze(pro)
	fmt.Printf("analyze result: %v\n", err)
	if err != nil {
		t.Errorf("analyze error: %v\n", err)
	}
}
