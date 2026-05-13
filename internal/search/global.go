package search

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

type GlobalResult struct {
	File    string
	Matches []Match
}

var rgPath string
var rgChecked bool

func HasRipgrep() bool {
	if !rgChecked {
		rgChecked = true
		path, err := exec.LookPath("rg")
		if err == nil {
			rgPath = path
		}
	}
	return rgPath != ""
}

func SearchGlobal(root, query string) []GlobalResult {
	if query == "" {
		return nil
	}

	if HasRipgrep() {
		return searchWithRg(root, query)
	}
	return searchWithGo(root, query)
}

func searchWithRg(root, query string) []GlobalResult {
	cmd := exec.Command(rgPath, "-n", "--no-heading", "--color=never", "-i",
		"--max-filesize", "1M", "--max-count", "50", query, root)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return nil
	}

	return parseRgOutput(root, query, out)
}

func parseRgOutput(root, query string, data []byte) []GlobalResult {
	resultMap := make(map[string]*GlobalResult)
	var order []string

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()

		firstColon := strings.Index(line, ":")
		if firstColon < 0 {
			continue
		}
		rest := line[firstColon+1:]
		secondColon := strings.Index(rest, ":")
		if secondColon < 0 {
			continue
		}

		file := line[:firstColon]
		lineNum, err := strconv.Atoi(rest[:secondColon])
		if err != nil {
			continue
		}
		text := rest[secondColon+1:]

		rel, err := filepath.Rel(root, file)
		if err != nil {
			rel = file
		}

		lower := strings.ToLower(text)
		col := strings.Index(lower, strings.ToLower(query))

		match := Match{
			Line: lineNum - 1,
			Col:  col,
			Text: text,
			File: rel,
		}

		if _, exists := resultMap[rel]; !exists {
			resultMap[rel] = &GlobalResult{File: rel}
			order = append(order, rel)
		}
		resultMap[rel].Matches = append(resultMap[rel].Matches, match)
	}

	results := make([]GlobalResult, 0, len(order))
	for _, f := range order {
		results = append(results, *resultMap[f])
	}
	return results
}

func searchWithGo(root, query string) []GlobalResult {
	lower := strings.ToLower(query)
	var results []GlobalResult

	gi, _ := loadGitIgnore(root)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := info.Name()
		if info.IsDir() {
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" {
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

		if isBinaryData(data) {
			return nil
		}

		content := string(data)
		lines := strings.Split(content, "\n")
		var matches []Match
		for i, line := range lines {
			col := strings.Index(strings.ToLower(line), lower)
			if col >= 0 {
				matches = append(matches, Match{
					Line: i,
					Col:  col,
					Text: line,
					File: rel,
				})
				if len(matches) >= 50 {
					break
				}
			}
		}

		if len(matches) > 0 {
			results = append(results, GlobalResult{File: rel, Matches: matches})
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

func isBinaryData(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	contentType := http.DetectContentType(sample)
	return !strings.HasPrefix(contentType, "text/")
}

func SearchEngine() string {
	if HasRipgrep() {
		return fmt.Sprintf("rg (%s)", rgPath)
	}
	return "go (native)"
}
