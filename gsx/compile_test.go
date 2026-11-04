package gsx

import (
	goparser "go/parser"
	"go/token"
	"strings"
	"testing"
)

func wrap(body string) string {
	return "package p\n\nimport \"github.com/polidog/rosanna-go\"\n\nfunc F(props Props, xs []string, ok bool) rosanna.Node {\n" + body + "\n}\n"
}

func compile(t *testing.T, src string) string {
	t.Helper()
	out, err := Compile([]byte(src), "t.gsx")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return string(out)
}

func TestCompile(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"element", `return <div className="a" id={props.ID} disabled>hi</div>`,
			[]string{`gsxrt.El("div", gsxrt.A("className", "a", "id", props.ID, "disabled", true, ), "hi", )`}},
		{"self closing", `return <br />`, []string{`gsxrt.El("br", nil, )`}},
		{"fragment", `return <><a /><b /></>`, []string{`gsxrt.Frag(gsxrt.El("a", nil, ), gsxrt.El("b", nil, ), )`}},
		{"component", `return <Card title="t" n={1} ID="x" />`, []string{`Card(CardProps{Title: "t", N: 1, ID: "x", }, )`}},
		{"component without props", `return <Header />`, []string{`Header(HeaderProps{}, )`}},
		{"component last attribute wins", `return <Card a="1" b="2" A="3" />`, []string{`Card(CardProps{A: "3", B: "2", }, )`}},
		{"package component", `return <ui.Button>x</ui.Button>`, []string{`ui.Button(ui.ButtonProps{Children: "x", }, )`}},
		{"component with one child", "return <Card>\n\t<b />\n</Card>", []string{`Children: gsxrt.El("b", nil, ), `}},
		{"component with children", "return <Card><b />text</Card>", []string{`Children: []gsxrt.Node{gsxrt.El("b", nil, ), "text", }, `}},
		{"for becomes htmlFor", `return <label for="a" />`, []string{`gsxrt.A("htmlFor", "a", )`}},
		{"last attribute wins", `return <a x="1" y="2" x="3" />`, []string{`gsxrt.A("x", "3", "y", "2", )`}},
		{"attribute string escapes and entities", `return <a title="a \"b\" &amp; \\ \n" />`, []string{`gsxrt.A("title", "a \"b\" & \\ \\n", )`}},
		{"single quoted attribute", `return <a title='it\'s' />`, []string{`"it's"`}},
		{"text whitespace", "return <p>\n\t\ta   b\n\t\tc\n\t</p>", []string{`gsxrt.El("p", nil, "a b c", `}},
		{"single line text keeps one space", `return <p>  a  <b>b</b>  </p>`, []string{`" a ", gsxrt.El("b", nil, "b", ), " ", `}},
		{"entities", `return <p>&lt;&amp;&nbsp;&#x41;</p>`, []string{`"<&\u00a0A"`}},
		{"text with quotes and slashes", `return <p>it's // not a comment # } ok</p>`, []string{`"it's // not a comment # } ok"`}},
		{"comment child", `return <p>{/* nothing */}</p>`, []string{`gsxrt.El("p", nil, )`}},
		{"spread", `return <ul>{...xs}</ul>`, []string{`gsxrt.El("ul", nil, xs, )`}},
		{"nested tag in expression", `return <ul>{rosanna.Map(xs, func(x string) rosanna.Node { return <li>{x}</li> })}</ul>`,
			[]string{`rosanna.Map(xs, func(x string) rosanna.Node { return gsxrt.El("li", nil, x, ) })`}},
		{"tag in attribute", `return <Layout header={<h1>T</h1>} />`, []string{`Header: gsxrt.El("h1", nil, "T", )`}},
		{"style is raw", "return <style>a > b { color: red; } &amp;</style>", []string{`"a > b { color: red; } &amp;"`}},
		{"script is raw", "return <script>if (a < b) { x() }</script>", []string{`"if (a < b) { x() }"`}},
		{"comparison stays go", "if len(xs) < 2 { return nil }\nreturn <p />", []string{"len(xs) < 2"}},
		{"channel receive stays go", "ch := make(chan int)\ngo func() { ch <- 1 }()\n_ = <-ch\nreturn nil", []string{"<-ch"}},
		{"strings stay go", "s := \"<div>\" + `<p>`\n_ = s\nreturn nil", []string{"\"<div>\" + `<p>`"}},
		{"assignment", "x := <p />\nreturn x", []string{`x := gsxrt.El("p", nil, )`}},
		{"return parenthesized", "return (\n\t<div>\n\t\t<p />\n\t</div>\n)", []string{"\n))\n"}},
		{"if helper", `return <div>{rosanna.If(ok, <b>yes</b>)}</div>`, []string{`rosanna.If(ok, gsxrt.El("b", nil, "yes", ))`}},
		{"composite literal", "return []rosanna.Node{<a />, <b />}", []string{`[]rosanna.Node{gsxrt.El("a", nil, ), gsxrt.El("b", nil, )}`}},
		{"multiline expression child", "return <p>{\n\tprops.Name\n}</p>", []string{"\n\tprops.Name, \n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := wrap(tt.body)
			out := compile(t, src)
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output does not contain %q\n%s", want, out)
				}
			}
			if _, err := goparser.ParseFile(token.NewFileSet(), "t.go", out, goparser.AllErrors); err != nil {
				t.Errorf("generated Go does not parse: %v\n%s", err, out)
			}
			if got, want := strings.Count(out, "\n")-2, strings.Count(src, "\n"); got != want {
				t.Errorf("generated Go has %d lines, the source has %d\n%s", got, want, out)
			}
		})
	}
}

func TestCompileWithoutTagsAddsNoImport(t *testing.T) {
	out := compile(t, "package p\n\nfunc F(a, b int) bool { return a < b }\n")
	if strings.Contains(out, "gsxrt") {
		t.Errorf("unexpected runtime import:\n%s", out)
	}
}

func TestCompileImportStaysOnThePackageLine(t *testing.T) {
	out := compile(t, "// package doc mentions package x\npackage p // c\n\nvar X = <p />\n")
	if !strings.Contains(out, "\npackage p; import gsxrt \"github.com/polidog/rosanna-go\" // c\n") {
		t.Errorf("import not injected after the package clause:\n%s", out)
	}
}

// //line 指令で、生成コードのエラーの位置は .gsx の行になる
func TestLineDirectiveMapsToSource(t *testing.T) {
	src := wrap("return (\n\t<div>\n\t\t<p>{props[}</p>\n\t</div>\n)")
	_, err := Compile([]byte(src), "dir/view.gsx")
	if err == nil {
		t.Fatal("expected a compile error for an invalid expression")
	}
	if !strings.Contains(err.Error(), "line 8") {
		t.Errorf("error does not point at line 8: %v", err)
	}

	src = wrap("x := <div>\n\t<p>{props}</p>\n</div>\nreturn x +")
	out := compile(t, src)
	_, err = goparser.ParseFile(token.NewFileSet(), "view_gsx.go", out, 0)
	if err == nil || !strings.Contains(err.Error(), "t.gsx:10:") {
		t.Errorf("syntax error is not reported at t.gsx:10: %v", err)
	}
}

func TestCompileErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"mismatched closing tag", "return <p>\n\t</q>", "line 7, column 2: Mismatched closing tag: expected </p>, got </q>"},
		{"unclosed tag", "return <p>", "unclosed tag <p>"},
		{"attribute spread", "return <div {...a} />", "attribute spread is not supported"},
		{"namespaced attribute", `return <svg xlink:href="a" />`, "namespaced attribute names are not supported"},
		{"empty attribute expression", `return <a x={} />`, "empty {} in attribute x"},
		{"two expressions", `return <p>{1, 2}</p>`, "must hold a single Go expression"},
		{"line comment at the end", "return <p>{1 // c\n}</p>", "a // comment cannot end an expression"},
		{"whitespace after <", "return <p>< b</p>", "whitespace after < is not allowed"},
		{"dotted lowercase", "return <a.b />", "a dotted tag name must name a component"},
		{"uppercase script close", "return <script>x</SCRIPT>", "unclosed tag <script>"},
		{"children twice", `return <Card children="a">b</Card>`, "both a children attribute and child nodes"},
		{"component attribute with hyphen", `return <Card data-id="a" />`, "must be a Go identifier to name a field of CardProps"},
		{"wide characters", "return <p>日本語\n</q>", "Mismatched closing tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile([]byte(wrap(tt.body)), "t.gsx")
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestErrorCaretCountsWideCharactersAsTwo(t *testing.T) {
	_, err := Compile([]byte(wrap("return <p>日本</q>")), "t.gsx")
	if err == nil {
		t.Fatal("expected an error")
	}
	lines := strings.Split(err.Error(), "\n")
	if got := lines[len(lines)-1]; got != "    "+strings.Repeat(" ", len("return <p>")+4)+"^" {
		t.Errorf("caret line %q\n%s", got, err)
	}
}

func TestOutputPath(t *testing.T) {
	for src, want := range map[string]string{
		"a/page.gsx":      "a/page_gsx.go",
		"a/page_test.gsx": "a/page_gsx_test.go",
	} {
		if got := OutputPath(src); got != want {
			t.Errorf("OutputPath(%q) = %q, want %q", src, got, want)
		}
	}
}
