package basic

import (
	"runtime"
	"strings"
	"testing"

	"github.com/polidog/rosanna-go"
)

func TestPage(t *testing.T) {
	got := rosanna.MustRender(Page("Toto IV", []string{"Rosanna", "Africa"}))
	want := `<div class="card"><section class="songs"><h2>Toto IV</h2><ul><li>Rosanna</li><li>Africa</li></ul></section></div>` +
		`<label for="plan">Plan</label>` +
		`<select id="plan" name="plan"><option value="a">A</option><option value="b" selected>B</option></select>` +
		`<a>x</a>` +
		"<style>\n\t\t\t\ta > b { color: red; }\n\t\t\t</style>" +
		`<input type="checkbox" checked aria-hidden="true" />`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestEmptySongList(t *testing.T) {
	got := rosanna.MustRender(SongList(SongListProps{Title: "B"}))
	if !strings.Contains(got, "<p>曲がありません &amp; 空です</p>") {
		t.Errorf("got %s", got)
	}
}

// panic の位置は .gsx の行になる
func TestPanicPointsAtGsxLine(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
		buf := make([]byte, 4096)
		trace := string(buf[:runtime.Stack(buf, false)])
		if !strings.Contains(trace, "songs.gsx:60") {
			t.Errorf("stack trace does not point at songs.gsx:60\n%s", trace)
		}
	}()
	Broken()
}
