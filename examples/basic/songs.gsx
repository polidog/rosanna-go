package basic

import (
	"github.com/polidog/rosanna-go"
)

type SongListProps struct {
	Title string
	Songs []string
}

// SongList は曲の一覧を出すコンポーネントです。
func SongList(p SongListProps) rosanna.Node {
	return (
		<section className="songs">
			<h2>{p.Title}</h2>
			<ul>
				{rosanna.Map(p.Songs, func(song string) rosanna.Node {
					return <li>{song}</li>
				})}
			</ul>
			{rosanna.If(len(p.Songs) == 0, <p>曲がありません &amp; 空です</p>)}
		</section>
	)
}

type CardProps struct {
	Children rosanna.Node
}

func Card(p CardProps) rosanna.Node {
	return <div className="card">{p.Children}</div>
}

func Page(title string, songs []string) rosanna.Node {
	return (
		<>
			<Card>
				<SongList title={title} songs={songs} />
			</Card>
			<label for="plan">Plan</label>
			<select id="plan" name="plan" value="b">
				<option value="a">A</option>
				<option value="b">B</option>
			</select>
			<a href="javascript:alert(1)">x</a>
			<style>
				a > b { color: red; }
			</style>
			<input type="checkbox" checked disabled={false} aria-hidden={true} />
		</>
	)
}

// Broken は panic の行が .gsx の行になることを確かめるためのものです。
func Broken() rosanna.Node {
	var names []string
	return (
		<p>
			{names[3]}
		</p>
	)
}
