package definition

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

type Result struct {
	File string
	Line int
	Text string
}

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:^|[\s])func\s+(?:\([^)]*\)\s+)?%s\s*[\[(]`),
	regexp.MustCompile(`(?:^|[\s])type\s+%s\s`),
	regexp.MustCompile(`(?:^|[\s])(?:var|const)\s+%s[\s=]`),

	regexp.MustCompile(`(?:^|[\s])def\s+%s\s*[\(:]`),
	regexp.MustCompile(`(?:^|[\s])class\s+%s[\s(:]`),

	regexp.MustCompile(`(?:^|[\s])(?:export\s+)?function\s+%s\s*[\(<]`),
	regexp.MustCompile(`(?:^|[\s])(?:export\s+)?(?:const|let|var)\s+%s[\s=:]`),
	regexp.MustCompile(`(?:^|[\s])(?:export\s+)?interface\s+%s[\s{<]`),
	regexp.MustCompile(`(?:^|[\s])(?:export\s+)?(?:abstract\s+)?class\s+%s[\s{<(]`),
	regexp.MustCompile(`(?:^|[\s])(?:export\s+)?type\s+%s[\s=<]`),
	regexp.MustCompile(`(?:^|[\s])(?:export\s+)?enum\s+%s[\s{]`),

	regexp.MustCompile(`(?:^|[\s])fn\s+%s\s*[\(<]`),
	regexp.MustCompile(`(?:^|[\s])(?:pub\s+)?struct\s+%s[\s{<]`),
	regexp.MustCompile(`(?:^|[\s])(?:pub\s+)?enum\s+%s[\s{<]`),
	regexp.MustCompile(`(?:^|[\s])(?:pub\s+)?trait\s+%s[\s{<:]`),
}

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"__pycache__": true, ".venv": true, "target": true, "build": true,
}

func Search(root, symbol string) []Result {
	if symbol == "" {
		return nil
	}

	escaped := regexp.QuoteMeta(symbol)
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		src := fmt.Sprintf(p.String(), escaped)
		compiled = append(compiled, regexp.MustCompile(src))
	}

	gi, _ := loadGitIgnore(root)
	var results []Result

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() > 1024*1024 {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		if gi != nil && gi.MatchesPath(rel) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if isBinary(data) {
			return nil
		}

		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			for _, re := range compiled {
				if re.MatchString(line) {
					results = append(results, Result{
						File: rel,
						Line: lineNum,
						Text: strings.TrimSpace(line),
					})
					break
				}
			}
			lineNum++
		}

		if len(results) >= 50 {
			return filepath.SkipAll
		}
		return nil
	})

	return results
}

func loadGitIgnore(root string) (*ignore.GitIgnore, error) {
	path := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	return ignore.CompileIgnoreFile(path)
}

func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	return !strings.HasPrefix(http.DetectContentType(sample), "text/")
}
