// Package gsx は、Go のファイルの中に HTML のようにタグを書ける GSX を Go に変換します。
//
// Rosanna と GSX の関係は React と JSX の関係と同じです。タグは rosanna.El、rosanna.Frag、rosanna.C の呼び出しになります。
package gsx

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// goCode は Go のコードの断片と、その中に書かれたタグの並びです。
type goCode struct {
	start int
	parts []any // string か *element
	// トークンがあったか。{/* コメント */} のように空なら false
	hasTokens bool
	// 最後のトークンが行コメントか
	endsWithLineComment bool
}

type element struct {
	pos      int // '<' の位置
	name     string
	fragment bool
	attrs    []*attribute
	children []*child
	closePos int // 閉じの '>' の位置
}

type attrKind int

const (
	attrString attrKind = iota
	attrExpr
	attrTrue
)

type attribute struct {
	pos  int
	name string
	kind attrKind
	str  string
	expr *goCode
}

type childKind int

const (
	childText childKind = iota
	childExpr
	childSpread
	childElement
)

type child struct {
	pos  int
	kind childKind
	text string
	expr *goCode
	el   *element
}

// Error は .gsx の位置を持つエラーです。
type Error struct {
	File   string
	Line   int
	Column int
	Msg    string
	Source string // 該当行
	Phase  string // "parse" か "compile"
}

func (e *Error) Error() string {
	prefix := "GSX " + e.Phase + " error"
	if e.File != "" {
		prefix += " in " + e.File
	}
	s := fmt.Sprintf("%s at line %d, column %d: %s", prefix, e.Line, e.Column, e.Msg)
	if e.Source != "" {
		s += "\n    " + e.Source + "\n    " + strings.Repeat(" ", caretWidth(e.Source, e.Column-1)) + "^"
	}
	return s
}

// caretWidth は列までの表示幅を数える。全角は幅 2
func caretWidth(line string, runes int) int {
	w := 0
	for i, r := range []rune(line) {
		if i >= runes {
			break
		}
		if r == '\t' {
			w++
		} else if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	return r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) || (r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe30 && r <= 0xfe6f) || (r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) || (r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd))
}

type parser struct {
	src  string
	file string
}

func (p *parser) errorAt(pos int, phase, format string, args ...any) *Error {
	line, col, text := p.position(pos)
	return &Error{File: p.file, Line: line, Column: col, Msg: fmt.Sprintf(format, args...), Source: text, Phase: phase}
}

func (p *parser) position(pos int) (line, col int, text string) {
	if pos > len(p.src) {
		pos = len(p.src)
	}
	line = strings.Count(p.src[:pos], "\n") + 1
	lineStart := strings.LastIndex(p.src[:pos], "\n") + 1
	lineEnd := strings.IndexByte(p.src[pos:], '\n')
	if lineEnd < 0 {
		lineEnd = len(p.src)
	} else {
		lineEnd += pos
	}
	col = utf8.RuneCountInString(p.src[lineStart:pos]) + 1
	return line, col, strings.TrimRight(p.src[lineStart:lineEnd], "\r")
}

func (p *parser) line(pos int) int {
	return strings.Count(p.src[:pos], "\n") + 1
}

// 直前のトークンの種類。タグを書ける位置かと、改行で ; が入るかを決める
const (
	tokNone    = "" // 式の先頭
	tokOperand = "operand"
	tokSemi    = ";"
)

// parseGo は pos から Go のコードを読む。inExpr なら対応する '}' の位置で止まり、その位置を返す。
func (p *parser) parseGo(pos int, inExpr bool) (*goCode, int, error) {
	code := &goCode{start: pos}
	chunkStart := pos
	last := tokNone
	if !inExpr {
		last = tokSemi
	}
	depth := 0
	src := p.src

	flush := func(end int) {
		if end > chunkStart {
			code.parts = append(code.parts, src[chunkStart:end])
		}
	}
	token := func(kind string) {
		last = kind
		code.hasTokens = true
		code.endsWithLineComment = false
	}

	for pos < len(src) {
		c := src[pos]
		switch {
		case c == '\n':
			if last == tokOperand || last == "return" || last == "break" || last == "continue" || last == "fallthrough" {
				last = tokSemi
			}
			pos++
		case c == ' ' || c == '\t' || c == '\r':
			pos++
		case strings.HasPrefix(src[pos:], "//"):
			end := strings.IndexByte(src[pos:], '\n')
			if end < 0 {
				end = len(src) - pos
			}
			pos += end
			code.endsWithLineComment = true
		case strings.HasPrefix(src[pos:], "/*"):
			end := strings.Index(src[pos+2:], "*/")
			if end < 0 {
				return nil, 0, p.errorAt(pos, "parse", "unterminated comment")
			}
			pos += end + 4
		case c == '"' || c == '\'':
			end, err := p.scanQuoted(pos, c)
			if err != nil {
				return nil, 0, err
			}
			pos = end
			token(tokOperand)
		case c == '`':
			end := strings.IndexByte(src[pos+1:], '`')
			if end < 0 {
				return nil, 0, p.errorAt(pos, "parse", "unterminated raw string literal")
			}
			pos += end + 2
			token(tokOperand)
		case isIdentStart(src, pos):
			start := pos
			for pos < len(src) && isIdentPart(src, pos) {
				_, size := utf8.DecodeRuneInString(src[pos:])
				pos += size
			}
			word := src[start:pos]
			switch word {
			case "return", "case", "go", "defer", "else", "if", "for", "switch", "select", "range", "chan", "map", "func",
				"struct", "interface", "type", "var", "const", "package", "import", "default", "goto":
				token(word)
			default:
				token(tokOperand)
			}
		case c >= '0' && c <= '9' || c == '.' && pos+1 < len(src) && src[pos+1] >= '0' && src[pos+1] <= '9':
			pos++
			for pos < len(src) {
				d := src[pos]
				if d == '_' || d == '.' || d >= '0' && d <= '9' || d >= 'a' && d <= 'z' || d >= 'A' && d <= 'Z' {
					pos++
				} else if (d == '+' || d == '-') && strings.ContainsRune("eEpP", rune(src[pos-1])) {
					pos++
				} else {
					break
				}
			}
			token(tokOperand)
		case c == '<' && tagAllowedAfter(last) && pos+1 < len(src) && (isASCIILetter(src[pos+1]) || src[pos+1] == '>'):
			flush(pos)
			el, end, err := p.parseElement(pos)
			if err != nil {
				return nil, 0, err
			}
			code.parts = append(code.parts, el)
			pos = end
			chunkStart = pos
			token(tokOperand)
		case c == '(' || c == '[' || c == '{':
			depth++
			pos++
			token(string(c))
		case c == ')' || c == ']' || c == '}':
			if c == '}' && inExpr && depth == 0 {
				flush(pos)
				return code, pos, nil
			}
			depth--
			pos++
			token(tokOperand)
		default:
			op := scanOperator(src[pos:])
			pos += len(op)
			if op == "++" || op == "--" {
				token(tokOperand)
			} else {
				token(op)
			}
		}
	}
	if inExpr {
		return nil, 0, p.errorAt(code.start-1, "parse", "unclosed {: expected }")
	}
	flush(pos)
	return code, pos, nil
}

func tagAllowedAfter(last string) bool {
	switch last {
	case tokNone, "return", "(", "[", "{", ",", "=", ":=", ":", "&&", "||", "!", "<-", "case", "else":
		return true
	}
	return false
}

var operators = []string{
	"<<=", ">>=", "&^=", "...", "&&", "||", "<-", "++", "--", "==", "!=", "<=", ">=", ":=", "+=", "-=", "*=", "/=",
	"%=", "&=", "|=", "^=", "<<", ">>", "&^",
}

func scanOperator(s string) string {
	for _, op := range operators {
		if strings.HasPrefix(s, op) {
			return op
		}
	}
	_, size := utf8.DecodeRuneInString(s)
	return s[:size]
}

func (p *parser) scanQuoted(pos int, quote byte) (int, error) {
	i := pos + 1
	for i < len(p.src) {
		switch p.src[i] {
		case '\\':
			i += 2
			continue
		case quote:
			return i + 1, nil
		case '\n':
			return 0, p.errorAt(pos, "parse", "unterminated literal")
		}
		i++
	}
	return 0, p.errorAt(pos, "parse", "unterminated literal")
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentStart(s string, pos int) bool {
	r, _ := utf8.DecodeRuneInString(s[pos:])
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(s string, pos int) bool {
	r, _ := utf8.DecodeRuneInString(s[pos:])
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isTagNameChar(c byte) bool {
	return isASCIILetter(c) || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.'
}

func isAttrNameChar(c byte) bool {
	return isASCIILetter(c) || c >= '0' && c <= '9' || c == '_' || c == '-'
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func (p *parser) skipSpace(pos int) int {
	for pos < len(p.src) && isSpace(p.src[pos]) {
		pos++
	}
	return pos
}

// parseElement は pos の '<' から要素を読み、閉じの '>' の次の位置を返す。
func (p *parser) parseElement(pos int) (*element, int, error) {
	src := p.src
	el := &element{pos: pos}
	i := pos + 1

	if src[i] == '>' {
		el.fragment = true
		i++
	} else {
		start := i
		for i < len(src) && isTagNameChar(src[i]) {
			i++
		}
		el.name = src[start:i]
		if strings.HasSuffix(el.name, ".") || strings.Contains(el.name, "..") {
			return nil, 0, p.errorAt(start, "parse", "invalid tag name %q", el.name)
		}
		if strings.Contains(el.name, ".") && !isComponentName(el.name) {
			return nil, 0, p.errorAt(start, "parse", "a dotted tag name must name a component: %q", el.name)
		}

		selfClosing, end, err := p.parseAttributes(el, i)
		if err != nil {
			return nil, 0, err
		}
		i = end
		if selfClosing {
			el.closePos = i - 1
			return el, i, nil
		}
	}

	if el.name == "style" || el.name == "script" {
		return p.parseRawText(el, i)
	}

	for {
		if i >= len(src) {
			return nil, 0, p.errorAt(pos, "parse", "unclosed tag <%s>", el.name)
		}
		switch src[i] {
		case '<':
			if strings.HasPrefix(src[i:], "</") {
				return p.parseClosingTag(el, i)
			}
			if i+1 < len(src) && isSpace(src[i+1]) {
				return nil, 0, p.errorAt(i, "parse", "whitespace after < is not allowed")
			}
			if i+1 >= len(src) || !(isASCIILetter(src[i+1]) || src[i+1] == '>') {
				return nil, 0, p.errorAt(i, "parse", "unexpected <: write &lt; for a literal <")
			}
			childEl, end, err := p.parseElement(i)
			if err != nil {
				return nil, 0, err
			}
			el.children = append(el.children, &child{pos: i, kind: childElement, el: childEl})
			i = end
		case '{':
			c, end, err := p.parseExprChild(i)
			if err != nil {
				return nil, 0, err
			}
			if c != nil {
				el.children = append(el.children, c)
			}
			i = end
		default:
			start := i
			for i < len(src) && src[i] != '<' && src[i] != '{' {
				i++
			}
			if text := decodeEntities(normalizeText(src[start:i])); text != "" {
				el.children = append(el.children, &child{pos: start, kind: childText, text: text})
			}
		}
	}
}

func isComponentName(name string) bool {
	last := name
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		last = name[i+1:]
	}
	return last != "" && last[0] >= 'A' && last[0] <= 'Z'
}

func (p *parser) parseAttributes(el *element, i int) (selfClosing bool, end int, err error) {
	src := p.src
	for {
		i = p.skipSpace(i)
		if i >= len(src) {
			return false, 0, p.errorAt(el.pos, "parse", "unclosed tag <%s>", el.name)
		}
		switch {
		case strings.HasPrefix(src[i:], "/>"):
			return true, i + 2, nil
		case src[i] == '>':
			return false, i + 1, nil
		case src[i] == '{':
			return false, 0, p.errorAt(i, "parse", "attribute spread is not supported: build the element with rosanna.El inside {}")
		}

		start := i
		for i < len(src) && isAttrNameChar(src[i]) {
			i++
		}
		if start == i {
			return false, 0, p.errorAt(i, "parse", "unexpected %q in tag <%s>", src[i], el.name)
		}
		attr := &attribute{pos: start, name: src[start:i]}
		if i < len(src) && src[i] == ':' {
			return false, 0, p.errorAt(start, "parse", "namespaced attribute names are not supported: %s", attr.name+":")
		}

		j := p.skipSpace(i)
		if j < len(src) && src[j] == '=' {
			j = p.skipSpace(j + 1)
			if j >= len(src) {
				return false, 0, p.errorAt(j, "parse", "expected attribute value")
			}
			switch src[j] {
			case '"', '\'':
				value, end, err := p.scanAttrString(j)
				if err != nil {
					return false, 0, err
				}
				attr.kind = attrString
				attr.str = value
				i = end
			case '{':
				code, end, err := p.parseGo(j+1, true)
				if err != nil {
					return false, 0, err
				}
				if !code.hasTokens {
					return false, 0, p.errorAt(j, "parse", "empty {} in attribute %s", attr.name)
				}
				attr.kind = attrExpr
				attr.expr = code
				i = end + 1
			default:
				return false, 0, p.errorAt(j, "parse", "expected \" or ' or { after %s=", attr.name)
			}
		} else {
			attr.kind = attrTrue
		}
		el.attrs = append(el.attrs, attr)
	}
}

// \ に続く引用符と \ だけを解除し、その後でエンティティをデコードする
func (p *parser) scanAttrString(pos int) (string, int, error) {
	quote := p.src[pos]
	var b strings.Builder
	for i := pos + 1; i < len(p.src); i++ {
		c := p.src[i]
		if c == '\\' && i+1 < len(p.src) && (p.src[i+1] == quote || p.src[i+1] == '\\') {
			b.WriteByte(p.src[i+1])
			i++
			continue
		}
		if c == quote {
			return decodeEntities(b.String()), i + 1, nil
		}
		b.WriteByte(c)
	}
	return "", 0, p.errorAt(pos, "parse", "unterminated attribute value")
}

// parseExprChild は '{' から子の式を読む。{/* コメント */} のように空なら nil を返す。
func (p *parser) parseExprChild(pos int) (*child, int, error) {
	i := pos + 1
	kind := childExpr
	j := p.skipSpace(i)
	if strings.HasPrefix(p.src[j:], "...") {
		kind = childSpread
		i = j + 3
	}
	code, end, err := p.parseGo(i, true)
	if err != nil {
		return nil, 0, err
	}
	if !code.hasTokens {
		if kind == childSpread {
			return nil, 0, p.errorAt(pos, "parse", "{...} needs an expression")
		}
		return nil, end + 1, nil
	}
	return &child{pos: pos, kind: kind, expr: code}, end + 1, nil
}

func (p *parser) parseClosingTag(el *element, pos int) (*element, int, error) {
	i := pos + 2
	start := i
	for i < len(p.src) && isTagNameChar(p.src[i]) {
		i++
	}
	name := p.src[start:i]
	i = p.skipSpace(i)
	if i >= len(p.src) || p.src[i] != '>' {
		return nil, 0, p.errorAt(i, "parse", "expected > in closing tag")
	}
	if el.fragment {
		if name != "" {
			return nil, 0, p.errorAt(pos, "parse", "Mismatched closing tag: expected </>, got </%s>", name)
		}
	} else if name != el.name {
		return nil, 0, p.errorAt(pos, "parse", "Mismatched closing tag: expected </%s>, got </%s>", el.name, name)
	}
	el.closePos = i
	return el, i + 1, nil
}

// style と script の中身は 1 つの文字列。{} を展開せず、エンティティをデコードせず、空白も変えない
func (p *parser) parseRawText(el *element, pos int) (*element, int, error) {
	closing := "</" + el.name
	for i := pos; ; {
		k := strings.Index(p.src[i:], closing)
		if k < 0 {
			return nil, 0, p.errorAt(el.pos, "parse", "unclosed tag <%s> (write the closing tag in lowercase)", el.name)
		}
		k += i
		j := p.skipSpace(k + len(closing))
		if j < len(p.src) && p.src[j] == '>' {
			if text := p.src[pos:k]; text != "" {
				el.children = append(el.children, &child{pos: pos, kind: childText, text: text})
			}
			el.closePos = j
			return el, j + 1, nil
		}
		i = k + len(closing)
	}
}
