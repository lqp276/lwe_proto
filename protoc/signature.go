package protoc

var sigCharMap = map[rune]string{'*': symTypeAny,
	'I': symTypeInt, 'S': symTypeString,
	'[': symTypeArray, 'V': symTypeArray,
	's': symTypeStruct,
}

type sigParser struct {
	text    []rune
	posMax  int
	pos     int
	curChar rune
}

type sigElem struct {
	value string
	tp    string
}

func (sp *sigParser) getStructSig() *sigElem {
	sp.advance()

	arr := []rune{}
	for sp.curChar != ';' && sp.curChar != 0 {
		arr = append(arr, sp.curChar)
		sp.advance()
	}

	sp.advance()

	elem := &sigElem{tp: symTypeStruct, value: string(arr)}
	return elem
}

func (sp *sigParser) advance() {
	sp.pos++

	if sp.pos < sp.posMax {
		sp.curChar = sp.text[sp.pos]
	} else {
		sp.curChar = 0
	}
}

func (s *sigParser) getNextElem() *sigElem {
	if s.curChar == 0 {
		return nil
	}

	switch s.curChar {
	case '*', 'I', 'S', '[', 'V':
		return &sigElem{tp: sigCharMap[s.curChar], value: string(s.curChar)}

	case 's':
		return s.getStructSig()

	default:
		doPanic("")
		return nil
	}
}

func newSigParser(text string) *sigParser {
	sp := &sigParser{}
	sp.text = []rune(text)
	sp.posMax = len(sp.text)
	sp.pos = 0

	if sp.posMax > 0 {
		sp.curChar = sp.text[0]
	}

	return sp
}
