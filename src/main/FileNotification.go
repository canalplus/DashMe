package main

import (
	"os"
	"utils"
	"path/filepath"
)

func inotifyRoutine(watcher *utils.Watcher, errChan chan error, cache *CacheManager) {
	/* Poll watcher channels */
	for {
		select {
		case ev := <-watcher.Event:
			/* received Inotify event */
			if ev.Mask & utils.IN_CREATE > 0 &&
				ev.Mask & utils.IN_ISDIR == 0 {
				/* Added file in video directory */
				err := cache.AddFile(ev.Name)
				if err != nil {
					/* Error while adding file, forward it */
					errChan <- err
				} else {
					/* Add watch on file for future updates */
					watcher.AddWatch(ev.Name, utils.IN_CLOSE_WRITE)
				}
			} else if ev.Mask & utils.IN_DELETE > 0 &&
				ev.Mask & utils.IN_ISDIR == 0 {
				/* Delted file in video directory */
				err := cache.RemoveFile(ev.Name)
				if err != nil {
					/* Error while removing file, forward it */
					errChan <- err
				} else {
					/* Remove watch on the file */
					watcher.RemoveWatch(ev.Name)
				}
			} else if ev.Mask & utils.IN_CLOSE_WRITE > 0 {
				cache.UpdateFile(ev.Name)
			}
		case err := <-watcher.Error:
			/* Forward error to main thread */
			errChan <- err
		}
	}
}

func StartInotify(cache *CacheManager, path string) (chan error, error) {
	/* Create watcher */
	watcher, err := utils.NewWatcher()
	if err != nil { return nil, err }
	/* Add watch on video directory */
	err = watcher.AddWatch(path, utils.IN_CREATE|utils.IN_DELETE)
	if err != nil { return nil, err }
	/* Add watch on every file in video directory */
	dir, err := os.Open(path)
	if err != nil { return nil, err }
	defer dir.Close()
	fileInfos, err := dir.Readdir(-1)
	if err != nil { return nil, err }
	for _, fi := range fileInfos {
		watcher.AddWatch(filepath.Join(path, fi.Name()), utils.IN_CLOSE_WRITE)
	}
	/* Create error channel for main thread */
	errChan := make(chan error)
	/* Start routine on other thread */
	go inotifyRoutine(watcher, errChan, cache)
	return errChan, nil
}
