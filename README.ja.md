# rosanna-go

[English](README.md)

[polidog/rosanna](https://github.com/polidog/rosanna) の Go 版です。HTML を React の要素の形で Go から書きます。
PHP 版の PSX にあたるものは **GSX** で、Go のファイルの中に HTML のようにタグを書けます。
Rosanna と GSX の関係は React と JSX の関係と同じで、GSX は省いても構いません。

描画は値を受け取って文字列を返すだけです。ルーターも DI も持たず、HTTP のことも知りません。

```bash
go get github.com/polidog/rosanna-go
```

## El で書く

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

- 子には `*Element`、`string`、数値、`fmt.Stringer`、`Renderable`、`nil`、`bool` と、それらのスライスを渡せます。`nil` と `bool` は何も出力しません。それ以外はエラーです。
- テキストと属性の値はエスケープします (PHP の `htmlspecialchars(ENT_QUOTES | ENT_SUBSTITUTE)` と同じ)。
- 属性名は JSX の名前 (`className`、`htmlFor`、`tabIndex`、`strokeWidth` …) も使えます。
- `on*` 属性、値のある `srcdoc` はエラーです。`href` や `src` の `javascript:`、`vbscript:`、`data:` は属性だけを落とします。
- `select` の `value` は出力せず、一致する `option` に `selected` を付けます。`multiple` ならスライスを渡します。
- `script` と `style` の中身はエスケープしません。中に `</script` などがあるとエラーです。

## GSX

`.gsx` は、タグを書ける Go のファイルです。

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
			{rosanna.If(len(p.Songs) == 0, <p>曲がありません</p>)}
		</section>
	)
}

func Page(title string, songs []string) rosanna.Node {
	return <Layout title={title}><SongList title={title} songs={songs} /></Layout>
}
```

### コンポーネントと props

コンポーネントは、`<名前>Props` の struct を 1 つ受け取って `rosanna.Node` を返す関数です。タグはその関数の呼び出しになります。

```go
<SongList title={title} songs={songs} />   // SongList(SongListProps{Title: title, Songs: songs})
<ui.Button kind="primary">OK</ui.Button>   // ui.Button(ui.ButtonProps{Kind: "primary", Children: "OK"})
<Header />                                 // Header(HeaderProps{})
```

- 属性名は先頭の 1 文字だけを大文字にしてフィールド名にします (`title` → `Title`)。残りは書いたとおりなので、`ID` のようなフィールドは `<User ID={id} />` と書きます。`data-id` のように Go の識別子にならない名前はコンパイルエラーです。
- 子は `Children` フィールドに入ります。子が 1 つならその値、複数なら `[]rosanna.Node` です。`Children rosanna.Node` と宣言しておけばどちらも受けられます。
- 属性のない `<Header />` も `HeaderProps{}` を渡すので、props のないコンポーネントにも空の `type HeaderProps struct{}` を用意します。
- フィールド名の誤りや型の不一致は Go のコンパイラが見つけ、`.gsx` の行で報告します。

```
components/songs.gsx:68: unknown field Colour in struct literal of type SongListProps
```

### コンパイル

```bash
go run github.com/polidog/rosanna-go/cmd/gsx generate [--check] [--watch] [paths...]
```

`foo.gsx` の隣に `foo_gsx.go` を書きます (`foo_test.gsx` は `foo_gsx_test.go`)。`go:generate` から呼ぶのが手軽です。

```go
//go:generate go run github.com/polidog/rosanna-go/cmd/gsx generate .
```

| オプション | 意味 |
|---|---|
| `--check` | 何も書かず、古い生成ファイルか、ソースのない生成ファイルがあれば exit 1。CI 向け |
| `--watch` | 500ms ごとに `.gsx` を確かめて再生成する。失敗しても終了しない |

- パスを省略するとカレントディレクトリを対象にします。ディレクトリは再帰的に走査し、`.`・`_` で始まるディレクトリ、`vendor`、`testdata` は飛ばします。
- `.gsx` がなくなった生成ファイルは消します (先頭が gsx のヘッダーのものだけ)。
- 生成コードの行は `.gsx` の行と一致し、`//line` 指令を付けます。**Go のコンパイルエラー、panic のスタックトレースは `.gsx` の行を指します。**

### PSX との違い

| | PSX (PHP) | GSX (Go) |
|---|---|---|
| コンポーネント | ファイルが `return` するクロージャ | `func(XxxProps) rosanna.Node` の関数。ファイルにいくつ書いてもよい |
| props | `array $props` | `XxxProps` の struct。フィールドと型は Go のコンパイラが検査する |
| 描画のタイミング | 描画のときに呼ぶ | タグを評価したときに呼ぶ (普通の関数呼び出し) |
| 名前の解決 | コンパイラの manifest | Go のコンパイラ。manifest はない |
| パッケージをまたぐ | `use` で名前を付ける | `<ui.Button />` と書ける |
| `{$a && <b/>}` | コンパイラが書き換える | 書けない。`{rosanna.If(a, <b/>)}` |
| `array_map` | `{array_map(fn ($x) => <li/>, $xs)}` | `{rosanna.Map(xs, func(x T) rosanna.Node { return <li/> })}` |
| 生成物の場所 | `var/cache/psx` | ソースの隣 (`_gsx.go`) |

### 構文

構文は PSX と同じです。

| 書き方 | 意味 |
|---|---|
| `<div>...</div>`、`<my-tag />` | 組み込みタグ。先頭が大文字でないもの |
| `<Counter />`、`<ui.Button />` | コンポーネント。名前 (ドットの後) の先頭が大文字のもの |
| `<>...</>` | Fragment |
| `a="..."`、`a='...'` | 文字列。`\` に続く引用符と `\` だけを解除し、その後でエンティティをデコードする |
| `a={式}` | Go の式 1 つ。中にタグも書ける |
| `a` | `true` |
| `{式}` | 子。`{...xs}` も書ける (スライスは描画のときに平らにする) |
| `{/* コメント */}` | 子を作らない |

- タグを書けるのは、Go が式を期待している位置だけです (`return`、`=`、`:=`、`(`、`,`、`[`、`{`、`:` の後)。`a < b`、`<-ch`、文字列とコメントの中の `<` は Go のままです。
- `return` の後で改行すると Go が `;` を入れるので、複数行のタグは `return (` の後に書きます。閉じの `)` は次の行に書いて構いません。
- テキストの空白、エンティティ、`style` と `script`、`for` → `htmlFor`、同名の属性は後が勝つ、は PSX と同じです。
- 属性のスプレッド (`<div {...a} />`) と名前空間付きの属性名 (`xlink:href`) はコンパイルエラーです。
- `{}` の末尾に `//` コメントは書けません (`/* */` を使います)。

## まだないもの

- 状態 (`fc()`、`useState`、`onClick`) と `<defer>`。PHP 版の `rosanna-state`、`rosanna-defer` にあたるもの
- エディタ (gopls) の `.gsx` 対応
- 生成ファイルは `gofmt` していません (行を揃えるため)。`gofmt -l` に出ます

## ライセンス

MIT
