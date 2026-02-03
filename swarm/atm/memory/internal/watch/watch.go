package watch

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	mi       *index.MemoryIndex
	debounce time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	watcher  *fsnotify.Watcher
}

func New(mi *index.MemoryIndex, debounce time.Duration) *Watcher {
	return &Watcher{mi: mi, debounce: debounce}
}

func (w *Watcher) Start(watchPaths ...string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	w.watcher = watcher
	for _, root := range watchPaths {
		w.addRecursive(root)
	}
	go w.loop()
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		return w.watcher.Add(path)
	})
}

func (w *Watcher) loop() {
	for {
		select {
		case event := <-w.watcher.Events:
			if event.Op&fsnotify.Write != fsnotify.Write && event.Op&fsnotify.Create != fsnotify.Create {
				continue
			}
			if filepath.Ext(event.Name) != ".md" {
				continue
			}
			w.mu.Lock()
			if w.timer != nil {
				w.timer.Stop()
			}
			w.timer = time.AfterFunc(w.debounce, func() {
				w.mi.IndexFile(event.Name)
			})
			w.mu.Unlock()
		case err := <-w.watcher.Errors:
			log.Println("watch error:", err)
		}
	}
}

func (w *Watcher) Stop() error {
	if w.timer != nil {
		w.timer.Stop()
	}
	return w.watcher.Close()
}