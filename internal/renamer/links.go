package renamer

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type DiscoveredLink struct {
	Index int
	URL   string
	Count int
}

type LinkScanReport struct {
	TotalFound int
	Links      []DiscoveredLink
}

var linkExtractionRegex = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/(?:\+|joinchat/)?[a-zA-Z0-9_/-]+|https?://[a-zA-Z0-9_\-\./:%?=+#&~]+`)

var ignoredLinkPrefixes = []string{
	"http://www.w3.org",
	"https://www.w3.org",
	"http://json-schema.org",
	"https://json-schema.org",
	"https://api.telegram.org",
	"http://schemas.openxmlformats.org",
}

func (e *Engine) ScanLinks(targetDir string) (*LinkScanReport, error) {
	counts := make(map[string]int)

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
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
		matches := linkExtractionRegex.FindAllString(content, -1)
		for _, m := range matches {
			cleaned := cleanExtractedURL(m)
			if cleaned != "" && !isIgnoredURL(cleaned) {
				counts[cleaned]++
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	report := &LinkScanReport{}
	for u, c := range counts {
		report.Links = append(report.Links, DiscoveredLink{
			URL:   u,
			Count: c,
		})
	}

	sort.Slice(report.Links, func(i, j int) bool {
		isTeleI := strings.Contains(report.Links[i].URL, "t.me") || strings.Contains(report.Links[i].URL, "telegram.me")
		isTeleJ := strings.Contains(report.Links[j].URL, "t.me") || strings.Contains(report.Links[j].URL, "telegram.me")
		if isTeleI && !isTeleJ {
			return true
		}
		if !isTeleI && isTeleJ {
			return false
		}
		return report.Links[i].Count > report.Links[j].Count
	})

	for i := range report.Links {
		report.Links[i].Index = i
		report.TotalFound += report.Links[i].Count
	}

	return report, nil
}

func (e *Engine) ReplaceLink(targetDir, oldLink, newLink string) (int, int, error) {
	filesModified := 0
	totalReplacements := 0

	pairs := generateLinkReplacementPairs(oldLink, newLink)

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
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

		original := string(data)
		modified := original
		fileReplacements := 0

		for _, p := range pairs {
			if strings.Contains(modified, p.Old) {
				c := strings.Count(modified, p.Old)
				modified = strings.ReplaceAll(modified, p.Old, p.New)
				fileReplacements += c
			}
		}

		if modified != original {
			err = os.WriteFile(path, []byte(modified), info.Mode())
			if err == nil {
				filesModified++
				totalReplacements += fileReplacements
			}
		}

		return nil
	})

	return filesModified, totalReplacements, err
}

func cleanExtractedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, ".,;)\"'>]`}")
	raw = strings.TrimLeft(raw, "\"'(<[{")
	if len(raw) < 7 {
		return ""
	}
	return raw
}

func isIgnoredURL(u string) bool {
	for _, ign := range ignoredLinkPrefixes {
		if strings.HasPrefix(u, ign) {
			return true
		}
	}
	return false
}

func generateLinkReplacementPairs(oldLink, newLink string) []ReplacementPair {
	var pairs []ReplacementPair

	cleanOld := strings.TrimSpace(oldLink)
	cleanNew := strings.TrimSpace(newLink)

	pairs = append(pairs, ReplacementPair{Old: cleanOld, New: cleanNew})

	oldWithoutProto := strings.TrimPrefix(strings.TrimPrefix(cleanOld, "https://"), "http://")
	newWithoutProto := strings.TrimPrefix(strings.TrimPrefix(cleanNew, "https://"), "http://")

	if oldWithoutProto != cleanOld || newWithoutProto != cleanNew {
		pairs = append(pairs, ReplacementPair{Old: oldWithoutProto, New: newWithoutProto})
		pairs = append(pairs, ReplacementPair{Old: "https://" + oldWithoutProto, New: "https://" + newWithoutProto})
		pairs = append(pairs, ReplacementPair{Old: "http://" + oldWithoutProto, New: "http://" + newWithoutProto})
	}

	return pairs
}
