// Package rosanna は、HTML を React の要素の形で Go から書くためのライブラリです。
//
// 要素は El や Frag で組み立てるか、.gsx に書いたタグを gsx コンパイラで Go に変換して作ります。
// Rosanna と GSX の関係は React と JSX の関係と同じで、GSX は省いても構いません。
//
// 描画は値を受け取って文字列を返すだけです。セッションを始めず、JavaScript も要らず、HTML に何も足しません。
package rosanna

import (
	"io"
	"strings"
)

// Node は描画できる値です。*Element、Element、string、数値、bool、nil、fmt.Stringer、Renderable と、
// それらのスライスを受け付けます。nil と bool は何も出力しません。
type Node = any

// FragmentType は囲みのタグを出さない要素の Type です。
const FragmentType = "Fragment"

// Attr は属性 1 つです。Name には JSX の名前 (className、htmlFor など) も書けます。
type Attr struct {
	Name  string
	Value any
}

// Attrs は属性の並びです。ソースの順に出力します。
type Attrs []Attr

// A は名前と値を交互に並べて Attrs を作ります。A("className", "card", "disabled", true)
func A(pairs ...any) Attrs {
	if len(pairs)%2 != 0 {
		panic("rosanna.A: pairs must be name/value pairs")
	}
	attrs := make(Attrs, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		name, ok := pairs[i].(string)
		if !ok {
			panic("rosanna.A: attribute names must be strings")
		}
		attrs = append(attrs, Attr{Name: name, Value: pairs[i+1]})
	}
	return attrs
}

// Get は大文字小文字を区別せずに属性を探し、nil でない最初の値を返します。
func (a Attrs) Get(name string) any {
	for _, attr := range a {
		if attr.Value != nil && strings.EqualFold(attr.Name, name) {
			return attr.Value
		}
	}
	return nil
}

// Element は描画前の木です。
type Element struct {
	Type     string
	Attrs    Attrs
	Children []Node
}

// El は要素を作ります。El("div", A("className", "card"), "text", El("b", nil, "bold"))
func El(tag string, attrs Attrs, children ...Node) *Element {
	return &Element{Type: tag, Attrs: attrs, Children: children}
}

// Frag は囲みのタグを出さない要素を作ります。
func Frag(children ...Node) *Element {
	return &Element{Type: FragmentType, Children: children}
}

// Renderable は自分を Node にする値です。children に置くと描画のときに Render が呼ばれます。
type Renderable interface {
	Render() Node
}

// If は cond が真なら n を、偽なら nil を返します。GSX では {rosanna.If(show, <b>x</b>)} と書きます。
func If(cond bool, n Node) Node {
	if cond {
		return n
	}
	return nil
}

// Map は xs のそれぞれを f で Node にします。
func Map[T any](xs []T, f func(T) Node) []Node {
	nodes := make([]Node, len(xs))
	for i, x := range xs {
		nodes[i] = f(x)
	}
	return nodes
}

// Render は n を HTML にします。
func Render(n Node) (string, error) {
	var b strings.Builder
	if err := newRenderer(&b).renderNode(n); err != nil {
		return "", err
	}
	return b.String(), nil
}

// MustRender は Render と同じですが、失敗すると panic します。
func MustRender(n Node) string {
	html, err := Render(n)
	if err != nil {
		panic(err)
	}
	return html
}

// RenderTo は n を HTML にして w に書きます。途中で失敗したときは何も書きません。
func RenderTo(w io.Writer, n Node) error {
	html, err := Render(n)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, html)
	return err
}
