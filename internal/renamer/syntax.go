package renamer

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type SyntaxVerificationResult struct {
	TotalChecked int
	Passed       int
	Warnings     []string
}

func VerifyDirectorySyntax(dir string) SyntaxVerificationResult {
	res := SyntaxVerificationResult{}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if DefaultBinaryExtensions[ext] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || isBinary(data) {
			return nil
		}
		res.TotalChecked++

		rel, _ := filepath.Rel(dir, path)

		if !utf8.Valid(data) {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: non-valid UTF-8 sequence detected", rel))
			return nil
		}

		if ext == ".json" {
			var js interface{}
			if err := json.Unmarshal(data, &js); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: invalid JSON: %v", rel, err))
				return nil
			}
		}

		if ext == ".go" {
			fset := token.NewFileSet()
			if _, err := parser.ParseFile(fset, path, data, parser.AllErrors); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: Go parser error: %v", rel, err))
				return nil
			}
		}

		res.Passed++
		return nil
	})
	return res
}
