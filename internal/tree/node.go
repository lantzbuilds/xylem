package tree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type IgnoreChecker interface {
	MatchesPath(path string) bool
}

type Node struct {
	Path     string
	Name     string
	IsDir    bool
	Expanded bool
	Ignored  bool
	Children []*Node
}

func NewNode(path, name string, isDir bool) *Node {
	return &Node{
		Path:  path,
		Name:  name,
		IsDir: isDir,
	}
}

func (n *Node) LoadChildren(ignore IgnoreChecker) error {
	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return err
	}

	n.Children = nil
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" {
			continue
		}
		fullPath := filepath.Join(n.Path, name)

		child := NewNode(fullPath, name, entry.IsDir())

		if n.Ignored {
			child.Ignored = true
		} else if ignore != nil {
			rel, _ := filepath.Rel(n.Path, fullPath)
			if entry.IsDir() {
				rel += "/"
			}
			if ignore.MatchesPath(rel) {
				child.Ignored = true
			}
		}

		n.Children = append(n.Children, child)
	}

	sort.Slice(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.Ignored != b.Ignored {
			return !a.Ignored
		}
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	return nil
}

func (n *Node) Flatten() []*Node {
	var result []*Node
	result = append(result, n)
	if n.IsDir && n.Expanded {
		for _, child := range n.Children {
			result = append(result, child.Flatten()...)
		}
	}
	return result
}

func (n *Node) Depth(root string) int {
	rel, err := filepath.Rel(root, n.Path)
	if err != nil {
		return 0
	}
	if rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator))
}
