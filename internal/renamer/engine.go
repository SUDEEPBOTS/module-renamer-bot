package renamer

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RenameOptions struct {
	TargetDir      string
	OldName        string
	NewName        string
	IncludeFonts   bool
	FileExtensions []string
	IgnoredDirs    []string
}

type RenameReport struct {
	FilesScanned       int
	FilesModified      int
	ReplacementsCount  int
	DirectoriesRenamed int
	FilesRenamed       int
}

type FindOptions struct {
	TargetDir    string
	SearchTerm   string
	IncludeFonts bool
}

type FileHit struct {
	Path  string
	Count int
}

type FindReport struct {
	Query      string
	TotalHits  int
	FilesCount int
	Hits       []FileHit
}

var DefaultIgnoredDirs = []string{
	".git",
	"__pycache__",
	".venv",
	"venv",
	"node_modules",
	".idea",
	".vscode",
	"dist",
	"build",
	".pytest_cache",
}

var DefaultBinaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".ico": true, ".svg": false, ".exe": true, ".dll": true, ".so": true,
	".dylib": true, ".bin": true, ".zip": true, ".tar": true, ".gz": true,
	".7z": true, ".rar": true, ".mp3": true, ".mp4": true, ".mkv": true,
	".ogg": true, ".wav": true, ".pyc": true, ".o": true, ".a": true,
	".pdf": true, ".ttf": true, ".woff": true, ".woff2": true, ".class": true,
}

type Engine struct {
	ignoredDirs map[string]bool
}

func NewEngine() *Engine {
	ignored := make(map[string]bool)
	for _, d := range DefaultIgnoredDirs {
		ignored[d] = true
	}
	return &Engine{ignoredDirs: ignored}
}

func (e *Engine) Execute(opts RenameOptions) (*RenameReport, error) {
	report := &RenameReport{}

	if opts.OldName == "" || opts.NewName == "" {
		return report, nil
	}

	err := e.processFileContents(opts, report)
	if err != nil {
		return report, err
	}

	err = e.renamePathsBottomUp(opts, report)
	if err != nil {
		return report, err
	}

	return report, nil
}

func (e *Engine) FindOccurrences(opts FindOptions) (*FindReport, error) {
	report := &FindReport{
		Query: opts.SearchTerm,
	}

	normQuery := NormalizeToASCII(opts.SearchTerm)
	lowerQuery := strings.ToLower(normQuery)
	if lowerQuery == "" {
		return report, nil
	}

	hitMap := make(map[string]int)

	err := filepath.Walk(opts.TargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if e.ignoredDirs[info.Name()] {
				return filepath.SkipDir
			}
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

		content := string(data)
		normContent := strings.ToLower(NormalizeToASCII(content))

		count := strings.Count(normContent, lowerQuery)
		if count > 0 {
			rel, _ := filepath.Rel(opts.TargetDir, path)
			hitMap[rel] = count
			report.TotalHits += count
		}
		return nil
	})

	if err != nil {
		return report, err
	}

	report.FilesCount = len(hitMap)

	for p, c := range hitMap {
		report.Hits = append(report.Hits, FileHit{Path: p, Count: c})
	}

	sort.Slice(report.Hits, func(i, j int) bool {
		return report.Hits[i].Count > report.Hits[j].Count
	})

	return report, nil
}

func (e *Engine) processFileContents(opts RenameOptions, report *RenameReport) error {
	pairs := generateReplacementPairs(opts.OldName, opts.NewName)

	return filepath.Walk(opts.TargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if e.ignoredDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if DefaultBinaryExtensions[ext] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		report.FilesScanned++

		if isBinary(data) {
			return nil
		}

		originalStr := string(data)
		content := originalStr
		replacedInFile := 0

		for _, pair := range pairs {
			if strings.Contains(content, pair.Old) {
				count := strings.Count(content, pair.Old)
				content = strings.ReplaceAll(content, pair.Old, pair.New)
				replacedInFile += count
			}
		}

		if opts.IncludeFonts {
			content, replacedInFile = replaceStylizedFonts(content, opts.OldName, opts.NewName, replacedInFile)
		}

		if content != originalStr {
			err = os.WriteFile(path, []byte(content), info.Mode())
			if err == nil {
				report.FilesModified++
				report.ReplacementsCount += replacedInFile
			}
		}

		return nil
	})
}

func (e *Engine) renamePathsBottomUp(opts RenameOptions, report *RenameReport) error {
	pairs := generateReplacementPairs(opts.OldName, opts.NewName)

	var paths []string

	_ = filepath.Walk(opts.TargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path == opts.TargetDir {
			return nil
		}
		if info.IsDir() && e.ignoredDirs[info.Name()] {
			return filepath.SkipDir
		}
		paths = append(paths, path)
		return nil
	})

	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) > len(paths[j])
	})

	for _, currentPath := range paths {
		dir := filepath.Dir(currentPath)
		base := filepath.Base(currentPath)
		newBase := base

		for _, pair := range pairs {
			if strings.Contains(newBase, pair.Old) {
				newBase = strings.ReplaceAll(newBase, pair.Old, pair.New)
			}
		}

		if newBase != base {
			newPath := filepath.Join(dir, newBase)
			if err := os.Rename(currentPath, newPath); err == nil {
				info, err := os.Stat(newPath)
				if err == nil && info.IsDir() {
					report.DirectoriesRenamed++
				} else {
					report.FilesRenamed++
				}
			}
		}
	}

	return nil
}

type ReplacementPair struct {
	Old string
	New string
}

func generateReplacementPairs(oldName, newName string) []ReplacementPair {
	var pairs []ReplacementPair

	oldUpper := strings.ToUpper(oldName)
	newUpper := strings.ToUpper(newName)

	oldLower := strings.ToLower(oldName)
	newLower := strings.ToLower(newName)

	oldTitle := toTitleCase(oldName)
	newTitle := toTitleCase(newName)

	pairs = append(pairs, ReplacementPair{Old: oldUpper + "MUSIC", New: newUpper + "MUSIC"})
	pairs = append(pairs, ReplacementPair{Old: oldLower + "music", New: newLower + "music"})
	pairs = append(pairs, ReplacementPair{Old: oldTitle + "Music", New: newTitle + "Music"})

	pairs = append(pairs, ReplacementPair{Old: oldUpper, New: newUpper})
	pairs = append(pairs, ReplacementPair{Old: oldName, New: newName})
	pairs = append(pairs, ReplacementPair{Old: oldTitle, New: newTitle})
	pairs = append(pairs, ReplacementPair{Old: oldLower, New: newLower})

	return pairs
}

func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + strings.ToLower(s[size:])
}

func isBinary(data []byte) bool {
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}
	return bytes.IndexByte(data[:checkLen], 0) != -1
}

func replaceStylizedFonts(content, oldName, newName string, currentReplacements int) (string, int) {
	normOld := NormalizeToASCII(oldName)
	normNew := NormalizeToASCII(newName)
	if len(normOld) == 0 {
		return content, currentReplacements
	}

	targets := []struct {
		oldStr string
		newStr string
	}{
		{ToAestheticFancy(normOld), ToAestheticFancy(normNew)},
		{ToBoldSerif(normOld), ToBoldSerif(normNew)},
		{ToBoldSerif(strings.ToUpper(normOld)), ToBoldSerif(strings.ToUpper(normNew))},
		{ToBoldSerif(strings.ToLower(normOld)), ToBoldSerif(strings.ToLower(normNew))},
		{ToBoldSerif(toTitleCase(normOld)), ToBoldSerif(toTitleCase(normNew))},
	}

	if oldName != normOld {
		targets = append(targets, struct {
			oldStr string
			newStr string
		}{oldName, newName})
	}

	for _, t := range targets {
		if t.oldStr != "" && t.oldStr != t.newStr && strings.Contains(content, t.oldStr) {
			count := strings.Count(content, t.oldStr)
			content = strings.ReplaceAll(content, t.oldStr, t.newStr)
			currentReplacements += count
		}
	}

	return content, currentReplacements
}
