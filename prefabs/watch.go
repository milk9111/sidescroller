package prefabs

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher *fsnotify.Watcher
	Events  chan string
	Errors  chan error
	closeCh chan struct{}
	once    sync.Once
}

func NewWatcher(dirs ...string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if err := w.Add(dir); err != nil {
			_ = w.Close()
			return nil, err
		}
	}

	watcher := &Watcher{
		watcher: w,
		Events:  make(chan string, 16),
		Errors:  make(chan error, 1),
		closeCh: make(chan struct{}),
	}
	go watcher.run()
	return watcher, nil
}

func (w *Watcher) Close() error {
	var err error
	w.once.Do(func() {
		close(w.closeCh)
		err = w.watcher.Close()
		close(w.Events)
		close(w.Errors)
	})
	return err
}

func (w *Watcher) run() {
	last := make(map[string]time.Time)
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			if !isSpecFile(event.Name) && !isScriptFile(event.Name) {
				continue
			}
			now := time.Now()
			if t, ok := last[event.Name]; ok && now.Sub(t) < 100*time.Millisecond {
				continue
			}
			last[event.Name] = now
			w.Events <- event.Name
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.Errors <- err
		case <-w.closeCh:
			return
		}
	}
}

func isSpecFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func isScriptFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tengo" || ext == ".lua"
}
