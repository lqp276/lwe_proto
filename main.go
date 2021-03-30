package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	protoc "lwe_proto/protoc"
	"os"
)

func main() {
	fname := flag.String("f", "", "the protocol file to use")
	mode := flag.String("m", "go", "the mode to use, modes: \"go\": golang")

	flag.Parse()

	if len(*fname) == 0 {
		fmt.Printf("error: protocol file not specified, see: %s -h\n", os.Args[0])
		return
	}

	body, err := ioutil.ReadFile(*fname)
	if err != nil {
		fmt.Printf("read protocol file failed, file: %s\n", *fname)
		return
	}

	program := string(body)
	//fmt.Printf("%s", program)
	p := protoc.NewParser(program)
	analyzer := protoc.NewSemanticAnalyzer()

	pro := p.Program()
	err = analyzer.DoAnalyze(pro)
	//fmt.Printf("analyze result: %v\n", err)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("semantic analyze error: %v\n", err))
		os.Exit(-1)
		return
	}

	interp := protoc.NewInterpreter()
	interp.SrcFile = *fname
	switch *mode {
	case "go":
		interp.Mode = protoc.INTERP_MODE_GO

	default:
		os.Stderr.WriteString(fmt.Sprintf("unknown mode: %s\n", *mode))
		os.Exit(-1)
		return
	}

	err = interp.DoInterpret(pro)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("interpret error: %v\n", err))
		os.Exit(-1)
		return
	}
}
