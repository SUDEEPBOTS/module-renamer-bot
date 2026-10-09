package renamer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var commonIgnoredFolderNames = map[string]bool{
	".git": true, ".github": true, ".idea": true, ".vscode": true,
	"venv": true, ".venv": true, "env": true, "node_modules": true,
	"__pycache__": true, "tests": true, "test": true, "docs": true,
	"doc": true, "assets": true, "dist": true, "build": true,
	"bin": true, "static": true, "templates": true, "scripts": true,
	"internal": true, "cmd": true, "pkg": true,
}

func DetectModuleName(targetDir string, repoURL string) string {
	if name := detectFromGoMod(targetDir); name != "" {
		return cleanDetectedName(name)
	}

	if name := detectFromPackageFiles(targetDir); name != "" {
		return cleanDetectedName(name)
	}

	if name := detectFromDirectories(targetDir); name != "" {
		return cleanDetectedName(name)
	}

	if repoURL != "" {
		if name := detectFromRepoURL(repoURL); name != "" {
			return cleanDetectedName(name)
		}
	}

	return "Yukki"
}

func detectFromGoMod(targetDir string) string {
	goModPath := filepath.Join(targetDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				mod := parts[1]
				segs := strings.Split(mod, "/")
				last := segs[len(segs)-1]
				if last != "main" && last != "." && last != "" {
					return last
				}
			}
		}
	}
	return ""
}

func detectFromPackageFiles(targetDir string) string {
	appJson := filepath.Join(targetDir, "app.json")
	if data, err := os.ReadFile(appJson); err == nil {
		re := regexp.MustCompile(`(?i)"name"\s*:\s*["']([^"']+)["']`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}

	pyproject := filepath.Join(targetDir, "pyproject.toml")
	if data, err := os.ReadFile(pyproject); err == nil {
		re := regexp.MustCompile(`(?i)name\s*=\s*["']([^"']+)["']`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}

	setupPy := filepath.Join(targetDir, "setup.py")
	if data, err := os.ReadFile(setupPy); err == nil {
		re := regexp.MustCompile(`(?i)name\s*=\s*["']([^"']+)["']`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}

	pkgJson := filepath.Join(targetDir, "package.json")
	if data, err := os.ReadFile(pkgJson); err == nil {
		re := regexp.MustCompile(`(?i)"name"\s*:\s*["']([^"']+)["']`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}

	return ""
}

func detectFromDirectories(targetDir string) string {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if commonIgnoredFolderNames[name] || strings.HasPrefix(name, ".") {
			continue
		}

		subEntries, err := os.ReadDir(filepath.Join(targetDir, name))
		if err == nil && len(subEntries) > 0 {
			for _, se := range subEntries {
				if se.Name() == "__init__.py" || strings.HasSuffix(se.Name(), ".py") || strings.HasSuffix(se.Name(), ".go") {
					return name
				}
			}
		}
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !commonIgnoredFolderNames[name] && !strings.HasPrefix(name, ".") {
			return name
		}
	}

	return ""
}

func detectFromRepoURL(repoURL string) string {
	clean := strings.TrimRight(repoURL, "/")
	clean = strings.TrimSuffix(clean, ".git")
	segs := strings.Split(clean, "/")
	if len(segs) > 0 {
		return segs[len(segs)-1]
	}
	return ""
}

func cleanDetectedName(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "-Go")
	raw = strings.TrimSuffix(raw, "-go")
	raw = strings.TrimSuffix(raw, "-bot")
	raw = strings.TrimSuffix(raw, "-Bot")
	raw = strings.TrimSuffix(raw, "_bot")
	raw = strings.TrimSuffix(raw, "-master")
	raw = strings.TrimSuffix(raw, "-main")

	if strings.HasSuffix(raw, "Music") && len(raw) > 5 {
		return strings.TrimSuffix(raw, "Music")
	}
	if strings.HasSuffix(raw, "MUSIC") && len(raw) > 5 {
		return strings.TrimSuffix(raw, "MUSIC")
	}

	return raw
}
