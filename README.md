# rosanna-go

[日本語](README.ja.md)

The Go port of [polidog/rosanna](https://github.com/polidog/rosanna). It lets you write HTML from Go in the shape of React elements.
The Go counterpart of PHP's PSX is **GSX**, which lets you write tags like HTML inside a Go file.
Rosanna relates to GSX the way React relates to JSX: GSX is optional.

Rendering takes values and returns a string. Rosanna has no router and no DI, and it knows nothing about HTTP.

```bash
go get github.com/polidog/rosanna-go
```

## Writing with El

```go
import "github.com/polidog/rosanna-go"

page := rosanna.El("main", nil,
	rosanna.El("h1", nil, "Toto IV"),
	rosanna.El("ul", rosanna.A("className", "songs"),
		rosanna.Map(songs, func(s string) rosanna.Node { return rosanna.El("li", nil, s) }),
	),
)
html, err := rosanna.Render(page)
```

- Children can be `*Element`, `string`, numbers, `fmt.Stringer`, `Renderable`, `nil`, `bool`, and slices of those. `nil` and `bool` output nothing. Anything else is an error.
- Text and attribute values are escaped (the same as PHP's `htmlspecialchars(ENT_QUOTES | ENT_SUBSTITUTE)`).
- Attribute names can also be JSX names (`className`, `htmlFor`, `tabIndex`, `strokeWidth`, ...).
- `on*` attributes and a non-empty `srcdoc` are errors. A `javascript:`, `vbscript:` or `data:` URL in `href`, `src` and similar attributes drops just that attribute.
- A `select`'s `value` is not output. Instead, the matching `option` is marked `selected`. With `multiple`, pass a slice.
- The content of `script` and `style` is not escaped. It is an error if it contains `</script` or the like.

## GSX

A `.gsx` file is a Go file in which you can write tags.

```go
// components/songs.gsx
package components

import "github.com/polidog/rosanna-go"

type SongListProps struct {
	Title string
	Songs []string
}

func SongList(p SongListProps) rosanna.Node {
	return (
		<section className="songs">
			<h2>{p.Title}</h2>
			<ul>
				{rosanna.Map(p.Songs, func(song string) rosanna.Node {
					return <li>{song}</li>
				})}
			</ul>
			{rosanna.If(len(p.Songs) == 0, <p>No songs</p>)}
		</section>
	)
}

func Page(title string, songs []string) rosanna.Node {
	return <Layout title={title}><SongList title={title} songs={songs} /></Layout>
}
```

### Components and props

A component is a function that takes one `<Name>Props` struct and returns a `rosanna.Node`. A tag becomes a call to that function.

```go
<SongList title={title} songs={songs} />   // SongList(SongListProps{Title: title, Songs: songs})
<ui.Button kind="primary">OK</ui.Button>   // ui.Button(ui.ButtonProps{Kind: "primary", Children: "OK"})
<Header />                                 // Header(HeaderProps{})
```

- An attribute name becomes a field name by upper-casing its first letter only (`title` → `Title`). The rest is kept as written, so a field like `ID` is written `<User ID={id} />`. A name that is not a Go identifier, such as `data-id`, is a compile error.
- Children go into the `Children` field: the value itself for a single child, a `[]rosanna.Node` for several. Declare `Children rosanna.Node` to accept both.
- `<Header />` with no attributes still passes `HeaderProps{}`, so a component without props needs an empty `type HeaderProps struct{}`.
- The Go compiler catches misspelled fields and mismatched types, and reports them at the `.gsx` line.

```
components/songs.gsx:68: unknown field Colour in struct literal of type SongListProps
```

### Compiling

```bash
go run github.com/polidog/rosanna-go/cmd/gsx generate [--check] [--watch] [paths...]
```

It writes `foo_gsx.go` next to `foo.gsx` (`foo_test.gsx` becomes `foo_gsx_test.go`). Running it from `go:generate` is the easiest way.

```go
//go:generate go run github.com/polidog/rosanna-go/cmd/gsx generate .
```

| Option | Meaning |
|---|---|
| `--check` | Write nothing, and exit 1 if a generated file is stale or has no source. For CI |
| `--watch` | Check `.gsx` files every 500ms and regenerate. Keeps running after a failure |

- Without paths, the current directory is used. Directories are walked recursively, skipping directories that start with `.` or `_`, `vendor` and `testdata`.
- A generated file whose `.gsx` is gone is removed (only files that start with the gsx header).
- The lines of the generated code match the lines of the `.gsx`, and a `//line` directive is added. **Go compile errors and panic stack traces point at the `.gsx` lines.**

### Differences from PSX

| | PSX (PHP) | GSX (Go) |
|---|---|---|
| Component | A closure the file `return`s | A `func(XxxProps) rosanna.Node`. A file can hold any number of them |
| Props | `array $props` | An `XxxProps` struct. The Go compiler checks fields and types |
| When it is called | At render time | When the tag is evaluated (a plain function call) |
| Name resolution | The compiler's manifest | The Go compiler. There is no manifest |
| Across packages | Name it with `use` | Write `<ui.Button />` |
| `{$a && <b/>}` | Rewritten by the compiler | Not supported. Use `{rosanna.If(a, <b/>)}` |
| `array_map` | `{array_map(fn ($x) => <li/>, $xs)}` | `{rosanna.Map(xs, func(x T) rosanna.Node { return <li/> })}` |
| Output location | `var/cache/psx` | Next to the source (`_gsx.go`) |

### Syntax

The syntax is the same as PSX.

| Form | Meaning |
|---|---|
| `<div>...</div>`, `<my-tag />` | A built-in tag. The name does not start with an upper-case letter |
| `<Counter />`, `<ui.Button />` | A component. The name (after the dot) starts with an upper-case letter |
| `<>...</>` | Fragment |
| `a="..."`, `a='...'` | A string. Only `\` followed by the quote or `\` is unescaped, then entities are decoded |
| `a={expr}` | A single Go expression. It can contain tags |
| `a` | `true` |
| `{expr}` | A child. `{...xs}` also works (slices are flattened at render time) |
| `{/* comment */}` | Makes no child |

- Tags can only be written where Go expects an expression (after `return`, `=`, `:=`, `(`, `,`, `[`, `{`, `:`). `a < b`, `<-ch`, and `<` inside strings and comments stay Go.
- Go inserts `;` at a line break after `return`, so write a multi-line tag after `return (`. The closing `)` may go on the next line.
- Text whitespace, entities, `style` and `script`, `for` → `htmlFor`, and "the last attribute with the same name wins" work as in PSX.
- Attribute spread (`<div {...a} />`) and namespaced attribute names (`xlink:href`) are compile errors.
- A `{}` cannot end with a `//` comment (use `/* */`).

## Not yet available

- State (`fc()`, `useState`, `onClick`) and `<defer>`, the counterparts of PHP's `rosanna-state` and `rosanna-defer`
- `.gsx` support in editors (gopls)
- Generated files are not `gofmt`ed (to keep lines aligned), so they show up in `gofmt -l`

## License

MIT
