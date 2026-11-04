package rosanna

import (
	"strings"
	"testing"
)

type song string

func (s song) String() string { return string(s) }

type row struct{ title string }

func (r row) Render() Node { return El("li", nil, r.title) }

func TestRender(t *testing.T) {
	tests := []struct {
		name string
		node Node
		want string
	}{
		{"text is escaped", El("p", nil, `<a href="x">'&'</a>`), `<p>&lt;a href=&quot;x&quot;&gt;&#039;&amp;&#039;&lt;/a&gt;</p>`},
		{"invalid utf-8 is substituted", El("p", nil, "a\xffb"), "<p>a�b</p>"},
		{"nil and bool render nothing", El("p", nil, nil, true, false, "x"), "<p>x</p>"},
		{"numbers", El("p", nil, 1, " ", 1.5, " ", 2.0, " ", uint8(3), " ", 1e20), "<p>1 1.5 2 3 1.0E+20</p>"},
		{"slices are flattened", El("ul", nil, []Node{El("li", nil, "a"), []string{"b", "c"}}), "<ul><li>a</li>bc</ul>"},
		{"typed element slices", El("ul", nil, []*Element{El("li", nil, "a")}), "<ul><li>a</li></ul>"},
		{"stringer", El("p", nil, song("<b>")), "<p>&lt;b&gt;</p>"},
		{"renderable", El("ul", nil, row{"Rosanna"}, row{"Africa"}), "<ul><li>Rosanna</li><li>Africa</li></ul>"},
		{"fragment", Frag(El("b", nil), "x"), "<b></b>x"},
		{"void", El("br", A("className", "x")), `<br class="x" />`},
		{"if", Frag(If(true, "a"), If(false, "b")), "a"},
		{"map", El("ul", nil, Map([]string{"a", "b"}, func(s string) Node { return El("li", nil, s) })), "<ul><li>a</li><li>b</li></ul>"},

		{"jsx attribute names", El("label", A("htmlFor", "a", "className", "b", "tabIndex", 1, "strokeWidth", 2)), `<label for="a" class="b" tabindex="1" stroke-width="2"></label>`},
		{"boolean attributes", El("input", A("disabled", true, "checked", false, "hidden", nil)), `<input disabled />`},
		{"boolean string attributes", El("div", A("aria-hidden", true, "draggable", false, "spellCheck", true)), `<div aria-hidden="true" draggable="false" spellcheck="true"></div>`},
		{"attribute values are escaped", El("a", A("title", `"'<&>`)), `<a title="&quot;&#039;&lt;&amp;&gt;"></a>`},
		{"dangerous urls are dropped", El("a", A("href", " \tjava\nscript:alert(1)", "title", "x")), `<a title="x"></a>`},
		{"data urls are dropped", El("img", A("src", "DATA:image/png,x")), `<img />`},
		{"safe urls are kept", El("a", A("href", "/x?a=1&b=2")), `<a href="/x?a=1&amp;b=2"></a>`},
		{"animate href is dropped", El("svg", nil, El("set", A("attributeName", "xlink:href", "to", "javascript:alert(1)"))), `<svg><set attributeName="xlink:href"></set></svg>`},
		{"animate values are checked one by one", El("animate", A("attributeName", "href", "values", "/a;javascript:x")), `<animate attributeName="href"></animate>`},

		{"select marks option", El("select", A("value", "b"), El("option", A("value", "a"), "A"), El("option", A("value", "b"), "B")), `<select><option value="a">A</option><option value="b" selected>B</option></select>`},
		{"select compares as strings", El("select", A("value", 2), El("option", A("value", "2"), "2")), `<select><option value="2" selected>2</option></select>`},
		{"select multiple", El("select", A("multiple", true, "value", []string{"a", "c"}), El("option", A("value", "a")), El("option", A("value", "b")), El("option", A("value", "c"))), `<select multiple><option value="a" selected></option><option value="b"></option><option value="c" selected></option></select>`},
		{"option without value matches its text", El("select", A("value", "Two words"), El("option", nil, "  Two\n ", El("b", nil, "words"), " ")), `<select><option selected>  Two
 <b>words</b> </option></select>`},
		{"optgroup", El("select", A("value", "b"), El("optgroup", nil, El("option", A("value", "b")))), `<select><optgroup><option value="b" selected></option></optgroup></select>`},
		{"own selected wins", El("select", A("value", "b"), El("option", A("value", "b", "selected", false))), `<select><option value="b"></option></select>`},
		{"option outside select is untouched", El("option", A("value", "b")), `<option value="b"></option>`},

		{"script is not escaped", El("script", nil, "if (a < b && c) {}"), `<script>if (a < b && c) {}</script>`},
		{"style children are joined", El("style", nil, "a > b", []string{" {", "}"}), `<style>a > b {}</style>`},
		{"script inside svg is escaped", El("svg", nil, El("script", nil, "a < b")), `<svg><script>a &lt; b</script></svg>`},
		{"textarea", El("textarea", A("name", "body"), "<x>"), `<textarea name="body">&lt;x&gt;</textarea>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.node)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestRenderErrors(t *testing.T) {
	tests := []struct {
		name string
		node Node
		want string
	}{
		{"invalid tag", El("di v", nil), "invalid HTML tag name"},
		{"invalid attribute", El("div", A("a b", 1)), "invalid HTML attribute name"},
		{"inline event handler", El("div", A("onClick", "x")), "inline event handler"},
		{"srcdoc", El("iframe", A("srcDoc", "<p>")), "srcdoc"},
		{"non-scalar attribute", El("div", A("title", []string{"a"})), "must be a string, number or bool"},
		{"unknown child", El("div", nil, struct{}{}), "cannot render a child of type struct {}"},
		{"script closing tag", El("script", nil, "</scr", "ipt>"), `must not contain "</script"`},
		{"raw text inside textarea", El("textarea", nil, El("style", nil, "</textarea>")), "close the ancestor early"},
		{"element inside script", El("script", nil, El("b", nil)), "only text is allowed"},
		{"wire attributes", El("button", A("wire:click", "x")), "needs a state layer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Render(tt.node)
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestRenderToWritesNothingOnError(t *testing.T) {
	var b strings.Builder
	if err := RenderTo(&b, El("div", nil, El("p", nil, "ok"), El("x y", nil))); err == nil {
		t.Fatal("expected an error")
	}
	if b.Len() != 0 {
		t.Errorf("wrote %q", b.String())
	}
}
