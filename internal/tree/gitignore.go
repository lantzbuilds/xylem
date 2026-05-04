package tree

import (
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

type GitIgnore struct {
	matcher *ignore.GitIgnore
}

func NewGitIgnore(root string) (*GitIgnore, error) {
	path := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	matcher, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return nil, err
	}

	return &GitIgnore{matcher: matcher}, nil
}

func (g *GitIgnore) MatchesPath(path string) bool {
	return g.matcher.MatchesPath(path)
}
