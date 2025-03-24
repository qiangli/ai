package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// https://github.com/fsnotify/fsnotify/blob/main/cmd/fsnotify/file.go
// Watch one or more files, but instead of watching the file directly it watches
// the parent directory. This solves various issues where files are frequently
// renamed, such as editors saving them.
func WatchFile(files ...string) error {
	if len(files) < 1 {
		return fmt.Errorf("must specify at least one file to watch")
	}

	// Create a new watcher.
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating a new watcher: %s", err)
	}
	defer w.Close()

	// Start listening for events.
	go fileLoop(w, files)

	// Add all files from the commandline.
	for _, p := range files {
		st, err := os.Lstat(p)
		if err != nil {
			return fmt.Errorf("%s", err)
		}

		if st.IsDir() {
			return fmt.Errorf("%q is a directory, not a file", p)
		}

		// Watch the directory, not the file itself.
		err = w.Add(filepath.Dir(p))
		if err != nil {
			return fmt.Errorf("%q: %s", p, err)
		}
	}

	printTime("ready; press ^C to exit")
	<-make(chan struct{}) // Block forever
	return nil
}

func fileLoop(w *fsnotify.Watcher, files []string) {
	i := 0
	for {
		select {
		// Read from Errors.
		case err, ok := <-w.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}
			printTime("ERROR: %s", err)
		// Read from Events.
		case e, ok := <-w.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}

			// Ignore files we're not interested in. Can use a
			// map[string]struct{} if you have a lot of files, but for just a
			// few files simply looping over a slice is faster.
			var found bool
			for _, f := range files {
				if f == e.Name {
					found = true
				}
			}
			if !found {
				continue
			}

			// Just print the event nicely aligned, and keep track how many
			// events we've seen.
			i++
			printTime("%3d %s", i, e)
		}
	}
}

// Print line prefixed with the time (a bit shorter than log.Print; we don't
// really need the date and ms is useful here).
func printTime(s string, args ...interface{}) {
	fmt.Printf(time.Now().Format("15:04:05.0000")+" "+s+"\n", args...)
}
