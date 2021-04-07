package main

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

func watchFiles(files []string, callback func(path string)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, path := range files {
		err = watcher.Add(path)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					callback(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Error(err)
				}
			}
		}
	}()
	return nil
}
