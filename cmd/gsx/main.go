// gsx は .gsx を Go に変換するコマンドです。
//
//	gsx generate [--check] [--watch] [paths...]
//
// パスを省略するとカレントディレクトリを対象にします。go:generate からは次のように呼びます。
//
//	//go:generate go run github.com/polidog/rosanna-go/cmd/gsx generate
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/polidog/rosanna-go/gsx"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gsx generate [--check] [--watch] [paths...]")
}

func run(args []string) int {
	if len(args) == 0 || args[0] != "generate" {
		usage()
		return 2
	}
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	check := fs.Bool("check", false, "write nothing and exit 1 if any generated file is stale (for CI)")
	watch := fs.Bool("watch", false, "regenerate when a .gsx file changes")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *check && *watch {
		fmt.Fprintln(os.Stderr, "gsx: --check and --watch cannot be used together")
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	if *watch {
		return watchLoop(paths)
	}
	return generateOnce(paths, *check)
}

func generateOnce(paths []string, check bool) int {
	res, err := gsx.Generate(paths, check)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if check {
		for _, f := range res.Written {
			fmt.Fprintf(os.Stderr, "stale: %s\n", f)
		}
		for _, f := range res.Removed {
			fmt.Fprintf(os.Stderr, "orphan: %s\n", f)
		}
		if len(res.Written)+len(res.Removed) > 0 {
			return 1
		}
		return 0
	}
	for _, f := range res.Written {
		fmt.Printf("wrote %s\n", f)
	}
	for _, f := range res.Removed {
		fmt.Printf("removed %s\n", f)
	}
	return 0
}

// watchLoop は 500ms ごとに .gsx の更新時刻と一覧を確かめる。失敗しても終了せず、次の変更まで待つ
func watchLoop(paths []string) int {
	last := ""
	for {
		state, err := snapshot(paths)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if state != last {
			last = state
			generateOnce(paths, false)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func snapshot(paths []string) (string, error) {
	var b strings.Builder
	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".gsx") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			fmt.Fprintf(&b, "%s %d %d\n", path, info.ModTime().UnixNano(), info.Size())
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return b.String(), nil
}
