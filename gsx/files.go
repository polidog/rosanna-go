package gsx

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// OutputPath は .gsx に対応する生成ファイルのパスです。foo.gsx は foo_gsx.go、foo_test.gsx は foo_gsx_test.go になります。
func OutputPath(src string) string {
	base := strings.TrimSuffix(src, ".gsx")
	if strings.HasSuffix(base, "_test") {
		return strings.TrimSuffix(base, "_test") + "_gsx_test.go"
	}
	return base + "_gsx.go"
}

// Result は Generate の結果です。
type Result struct {
	Written []string // 書いた (check では古い) 生成ファイル
	Removed []string // ソースがなくなって消した (check では残っている) 生成ファイル
}

// Generate は paths の下の .gsx をコンパイルし、隣に生成ファイルを書きます。ディレクトリは再帰的に走査し、
// シンボリックリンクと、"." や "_" で始まるディレクトリ、vendor、testdata は飛ばします。
// check なら何も書かず、古いものを Result に入れて返します。エラーは最初の 1 つで止まります。
func Generate(paths []string, check bool) (*Result, error) {
	res := &Result{}
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return res, err
		}
		if !info.IsDir() {
			if err := generateFile(root, check, res); err != nil {
				return res, err
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() {
				if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" || name == "testdata") {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return nil
			}
			switch {
			case strings.HasSuffix(name, ".gsx"):
				return generateFile(path, check, res)
			case strings.HasSuffix(name, "_gsx.go") || strings.HasSuffix(name, "_gsx_test.go"):
				return removeOrphan(path, check, res)
			}
			return nil
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

func generateFile(src string, check bool, res *Result) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	output, err := Compile(input, src)
	if err != nil {
		return err
	}
	dst := OutputPath(src)
	current, err := os.ReadFile(dst)
	if err == nil && bytes.Equal(current, output) {
		return nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	res.Written = append(res.Written, dst)
	if check {
		return nil
	}
	// 一時ファイルと rename で置き換え、書きかけのファイルを読ませない
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".gsx-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(output); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

// removeOrphan はソースの .gsx がなくなった生成ファイルを消す。gsx が書いたファイルだけを対象にする
func removeOrphan(path string, check bool, res *Result) error {
	var src string
	if strings.HasSuffix(path, "_gsx_test.go") {
		src = strings.TrimSuffix(path, "_gsx_test.go") + "_test.gsx"
	} else {
		src = strings.TrimSuffix(path, "_gsx.go") + ".gsx"
	}
	if _, err := os.Stat(src); err == nil {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(content, []byte(Header+"\n")) {
		return nil
	}
	res.Removed = append(res.Removed, path)
	if check {
		return nil
	}
	return os.Remove(path)
}
