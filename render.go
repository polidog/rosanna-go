package rosanna

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	tagNamePattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9\-]*$`)
	attributeNamePattern = regexp.MustCompile(`^[A-Za-z_:][A-Za-z0-9:_.\-]*$`)
	eventHandlerPattern  = regexp.MustCompile(`(?i)^on[a-z]`)
)

type renderer struct {
	b                   *strings.Builder
	foreignContentDepth int
	textOnlyAncestors   []string
	// いま開いている select の value。nil なら select の外
	selectValues map[string]bool
}

func newRenderer(b *strings.Builder) *renderer {
	return &renderer{b: b}
}

func (r *renderer) renderNode(n Node) error {
	switch v := n.(type) {
	case nil:
		return nil
	case *Element:
		if v == nil {
			return nil
		}
		return r.renderElement(v)
	case Element:
		return r.renderElement(&v)
	case Renderable:
		return r.renderNode(v.Render())
	case string:
		r.b.WriteString(Escape(v))
		return nil
	case bool:
		return nil
	case fmt.Stringer:
		r.b.WriteString(Escape(v.String()))
		return nil
	}
	if s, ok := numberString(n); ok {
		r.b.WriteString(s)
		return nil
	}
	if items, ok := sliceItems(n); ok {
		for _, item := range items {
			if err := r.renderNode(item); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("rosanna: cannot render a child of type %T. Children must be an *Element, Renderable, string, number, fmt.Stringer, nil, bool, or a slice of those", n)
}

func (r *renderer) renderChildren(children []Node) error {
	for _, child := range children {
		if err := r.renderNode(child); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) renderElement(e *Element) error {
	if e.Type == FragmentType {
		return r.renderChildren(e.Children)
	}

	tag := e.Type
	if !tagNamePattern.MatchString(tag) {
		return fmt.Errorf("rosanna: invalid HTML tag name %q: tag names must match %s", tag, tagNamePattern)
	}
	lowerTag := strings.ToLower(tag)

	outer := r.selectValues
	defer func() { r.selectValues = outer }()

	if lowerTag == "select" {
		values, err := selectValues(e.Attrs.Get("value"))
		if err != nil {
			return err
		}
		r.selectValues = values
		attrs := make(Attrs, 0, len(e.Attrs))
		for _, a := range e.Attrs {
			if !strings.EqualFold(a.Name, "value") {
				attrs = append(attrs, a)
			}
		}
		e = &Element{Type: e.Type, Attrs: attrs, Children: e.Children}
	} else if lowerTag == "option" && len(r.selectValues) > 0 && e.Attrs.Get("selected") == nil {
		marked, err := r.markIfSelected(e)
		if err != nil {
			return err
		}
		e = marked
	}

	for _, a := range e.Attrs {
		if len(a.Name) >= 5 && strings.EqualFold(a.Name[:5], "wire:") {
			return fmt.Errorf("rosanna: %s needs a state layer: event bindings do nothing without it", a.Name)
		}
	}

	attrs, err := renderAttributes(lowerTag, e.Attrs)
	if err != nil {
		return err
	}

	if isVoidTag(lowerTag) {
		r.b.WriteString("<" + tag + attrs + " />")
		return nil
	}

	r.b.WriteString("<" + tag + attrs + ">")
	if err := r.renderContent(lowerTag, e.Children); err != nil {
		return err
	}
	r.b.WriteString("</" + tag + ">")
	return nil
}

func (r *renderer) markIfSelected(option *Element) (*Element, error) {
	value := option.Attrs.Get("value")
	key, ok := scalarString(value)
	if b, isBool := value.(bool); isBool {
		// 値なしの属性 (true) の値は空文字列である
		ok = b
		key = ""
	}
	if !ok {
		// value 属性が出ない option の値はテキスト。調べるときに解決した子を描画にも使い、2 回解決しない
		children := resolveDeep(option.Children)
		option = &Element{Type: option.Type, Attrs: option.Attrs, Children: children}
		key = collapseWhitespace(textOf(children))
	}
	if !r.selectValues[key] {
		return option, nil
	}
	attrs := append(append(Attrs{}, option.Attrs...), Attr{Name: "selected", Value: true})
	return &Element{Type: option.Type, Attrs: attrs, Children: option.Children}, nil
}

func (r *renderer) renderContent(lowerTag string, children []Node) error {
	if isRawTextTag(lowerTag) && r.foreignContentDepth == 0 {
		return r.renderRawText(lowerTag, children)
	}
	if isTextOnlyAncestor(lowerTag) {
		r.textOnlyAncestors = append(r.textOnlyAncestors, lowerTag)
		defer func() { r.textOnlyAncestors = r.textOnlyAncestors[:len(r.textOnlyAncestors)-1] }()
		return r.renderChildren(children)
	}
	if lowerTag == "svg" || lowerTag == "math" {
		// svg と math の中ではパーサーが script と style の中身をマークアップとして読む
		r.foreignContentDepth++
		defer func() { r.foreignContentDepth-- }()
	}
	return r.renderChildren(children)
}

// パーサーがデコードしないのでエスケープせずに連結する。子に分割しても通らないよう、連結後に終了タグを検査する。
func (r *renderer) renderRawText(lowerTag string, children []Node) error {
	var b strings.Builder
	if err := joinRawText(&b, children); err != nil {
		return err
	}
	text := b.String()
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "</"+lowerTag) {
		return fmt.Errorf(`rosanna: the content of <%[1]s> must not contain "</%[1]s": it would close the element early`, lowerTag)
	}
	for _, ancestor := range r.textOnlyAncestors {
		if strings.Contains(lowerText, "</"+ancestor) {
			return fmt.Errorf(`rosanna: the content of <%[1]s> inside <%[2]s> must not contain "</%[2]s": it would close the ancestor early`, lowerTag, ancestor)
		}
	}
	r.b.WriteString(text)
	return nil
}

func joinRawText(b *strings.Builder, children []Node) error {
	for _, child := range children {
		switch v := child.(type) {
		case nil, bool:
			continue
		case string:
			b.WriteString(v)
			continue
		case fmt.Stringer:
			if _, isRenderable := child.(Renderable); !isRenderable {
				b.WriteString(v.String())
				continue
			}
		}
		if s, ok := numberString(child); ok {
			b.WriteString(s)
			continue
		}
		if items, ok := sliceItems(child); ok {
			if err := joinRawText(b, items); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("rosanna: cannot render a child of type %T inside <script> or <style>: only text is allowed", child)
	}
	return nil
}

func renderAttributes(lowerTag string, attrs Attrs) (string, error) {
	var b strings.Builder
	animatesHref := (lowerTag == "animate" || lowerTag == "set") && targetsHref(attrs)

	for _, a := range attrs {
		name := AttributeName(a.Name)
		if err := assertValidAttributeName(name); err != nil {
			return "", err
		}
		lowerName := strings.ToLower(name)

		if v, ok := a.Value.(bool); ok {
			if isBooleanStringAttribute(lowerName) {
				b.WriteString(" " + name + `="` + strconv.FormatBool(v) + `"`)
			} else if v {
				b.WriteString(" " + name)
			}
			continue
		}
		if a.Value == nil {
			continue
		}

		value, ok := scalarString(a.Value)
		if !ok {
			return "", fmt.Errorf("rosanna: value for attribute %q must be a string, number or bool, got %T", name, a.Value)
		}

		// 中身が同じオリジンの HTML になるので、スキームの検査では安全にならない
		if lowerName == "srcdoc" && value != "" {
			return "", fmt.Errorf("rosanna: the srcdoc attribute is not supported because its value is rendered as HTML in the same origin. Use src with a separate URL instead")
		}
		if isURLAttribute(lowerName) && hasDangerousScheme(value) {
			continue
		}
		if animatesHref && isAnimationValueAttribute(lowerName) && hasDangerousAnimationValue(value) {
			continue
		}

		b.WriteString(" " + name + `="` + Escape(value) + `"`)
	}
	return b.String(), nil
}

func assertValidAttributeName(name string) error {
	if !attributeNamePattern.MatchString(name) {
		return fmt.Errorf("rosanna: invalid HTML attribute name %q: attribute names must match %s", name, attributeNamePattern)
	}
	if eventHandlerPattern.MatchString(name) {
		return fmt.Errorf("rosanna: inline event handler attribute %q is not allowed", name)
	}
	return nil
}

// Chromium は <set attributeName="href" to="javascript:..."> を実行する
func targetsHref(attrs Attrs) bool {
	for _, a := range attrs {
		if strings.ToLower(AttributeName(a.Name)) != "attributename" {
			continue
		}
		s, ok := scalarString(a.Value)
		if !ok {
			continue
		}
		target := strings.ToLower(strings.TrimFunc(s, func(r rune) bool { return r <= 0x20 }))
		if i := strings.LastIndex(target, ":"); i >= 0 {
			target = target[i+1:]
		}
		if target == "href" {
			return true
		}
	}
	return false
}

func selectValues(value any) (map[string]bool, error) {
	values := map[string]bool{}
	items, ok := sliceItems(value)
	if !ok {
		items = []any{value}
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if _, isBool := item.(bool); isBool {
			continue
		}
		s, ok := scalarString(item)
		if !ok {
			return nil, fmt.Errorf(`rosanna: value for attribute "value" must be a string, number or bool, got %T`, item)
		}
		values[s] = true
	}
	return values, nil
}

// resolveDeep は Renderable を解決し、Element の子孫も同じく解決する。
func resolveDeep(children []Node) []Node {
	resolved := make([]Node, 0, len(children))
	var walk func(n Node)
	walk = func(n Node) {
		switch v := n.(type) {
		case *Element:
			if v != nil {
				resolved = append(resolved, &Element{Type: v.Type, Attrs: v.Attrs, Children: resolveDeep(v.Children)})
			}
			return
		case Element:
			resolved = append(resolved, &Element{Type: v.Type, Attrs: v.Attrs, Children: resolveDeep(v.Children)})
			return
		case Renderable:
			walk(v.Render())
			return
		case string:
			resolved = append(resolved, v)
			return
		}
		if items, ok := sliceItems(n); ok {
			for _, item := range items {
				walk(item)
			}
			return
		}
		resolved = append(resolved, n)
	}
	for _, child := range children {
		walk(child)
	}
	return resolved
}

func textOf(children []Node) string {
	var b strings.Builder
	for _, child := range children {
		switch v := child.(type) {
		case *Element:
			if v != nil {
				b.WriteString(textOf(v.Children))
			}
			continue
		case Element:
			b.WriteString(textOf(v.Children))
			continue
		case string:
			b.WriteString(v)
			continue
		case nil, bool:
			continue
		case fmt.Stringer:
			b.WriteString(v.String())
			continue
		}
		if s, ok := numberString(child); ok {
			b.WriteString(s)
		} else if items, ok := sliceItems(child); ok {
			b.WriteString(textOf(items))
		}
	}
	return b.String()
}

var asciiWhitespace = regexp.MustCompile(`[\t\n\f\r ]+`)

// HTML の option の value と同じく、ASCII の空白を詰めて前後を除く
func collapseWhitespace(s string) string {
	return strings.Trim(asciiWhitespace.ReplaceAllString(s, " "), " ")
}

// Escape は PHP の htmlspecialchars(ENT_QUOTES | ENT_SUBSTITUTE) と同じくエスケープします。
// 不正な UTF-8 のバイトは U+FFFD にします。
func Escape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteString("�")
			i++
			continue
		}
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#039;")
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	return b.String()
}

// scalarString は string、数値、bool を属性の値の文字列にする。true は "1"、false は ""。
func scalarString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case bool:
		if x {
			return "1", true
		}
		return "", true
	}
	return numberString(v)
}

func numberString(v any) (string, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(rv.Uint(), 10), true
	case reflect.Float32, reflect.Float64:
		return formatFloat(rv.Float(), rv.Type().Bits()), true
	}
	return "", false
}

// PHP の (string) と同じ形にする。1.0 は "1"、1e20 は "1.0E+20"
func formatFloat(f float64, bits int) string {
	switch {
	case math.IsNaN(f):
		return "NAN"
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	}
	abs := math.Abs(f)
	if abs != 0 && (abs >= 1e15 || abs < 1e-4) {
		s := strconv.FormatFloat(f, 'E', -1, bits)
		mantissa, exp, _ := strings.Cut(s, "E")
		if !strings.Contains(mantissa, ".") {
			mantissa += ".0"
		}
		return mantissa + "E" + exp
	}
	return strconv.FormatFloat(f, 'f', -1, bits)
}

// sliceItems は string 以外のスライスと配列を []any にする。
func sliceItems(v any) ([]any, bool) {
	switch x := v.(type) {
	case []Node:
		return x, true
	case []*Element:
		items := make([]any, len(x))
		for i, e := range x {
			items[i] = e
		}
		return items, true
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		return items, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	if rv.Type().Elem().Kind() == reflect.Uint8 {
		// []byte は文字列の代わりに使われることが多いが、黙って数値の列にしない
		return nil, false
	}
	items := make([]any, rv.Len())
	for i := range items {
		items[i] = rv.Index(i).Interface()
	}
	return items, true
}
