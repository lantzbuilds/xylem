package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
}

type Watcher struct {
	fsw    *fsnotify.Watcher
	Events chan struct{}
	done   chan struct{}
}

func New(root string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsw:    fsw,
		Events: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if skipDirs[info.Name()] {
			return filepath.SkipDir
		}
		fsw.Add(path)
		return nil
	})

	go w.loop(root)
	return w, nil
}

func (w *Watcher) loop(root string) {
	var timer *time.Timer
	defer close(w.done)

	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if shouldIgnore(root, event.Name) {
				continue
			}
			if event.Has(fsnotify.Create) {
				w.addIfDir(event.Name)
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(200*time.Millisecond, func() {
				select {
				case w.Events <- struct{}{}:
				default:
				}
			})
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) addIfDir(path string) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	if skipDirs[info.Name()] {
		return
	}
	w.fsw.Add(path)
}

func (w *Watcher) Close() {
	w.fsw.Close()
	<-w.done
}

func shouldIgnore(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, p := range parts {
		if skipDirs[p] {
			return true
		}
	}
	return false
}
