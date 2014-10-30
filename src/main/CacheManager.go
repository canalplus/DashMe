package main

import (
	"os"
	"utils"
	"errors"
	"path/filepath"
)

/*
  $CACHED_DIR/$FILENAME/manifest.mpd
  $CACHED_DIR/$FILENAME/chunk1.mp4
*/

type CacheManager struct {
	videoDir   string
	cachedDir  string
	availables []string
	cached     []string
	converter  DASHBuilder
}

func (c *CacheManager) BuildAvailables() {
	/* Iterate over videoDir (1st level only) and extract filenames */
	dir, err := os.Open(c.videoDir)
	if err != nil { return }
	defer dir.Close()
	fileInfos, err := dir.Readdir(-1)
	if err != nil { return }
	for _, fi := range fileInfos {
		c.availables = append(c.availables, utils.RemoveExtension(fi.Name()))
	}
}

func (c *CacheManager) BuildCached() {
	/* Iterate over cachedDir (1st level only) and retrieve filenames */
	dir, err := os.Open(c.cachedDir)
	if err != nil { return }
	defer dir.Close()
	fileInfos, err := dir.Readdir(-1)
	if err != nil { return }
	for _, fi := range fileInfos {
		c.cached = append(c.cached, utils.RemoveExtension(fi.Name()))
	}
}

func (c *CacheManager) Initialise(videoDir string, cachedDir string) {
	c.videoDir = videoDir
	c.BuildAvailables()
	c.cachedDir = cachedDir
	if (utils.FileExist(cachedDir)) {
		c.BuildCached()
	} else {
		os.MkdirAll(cachedDir, os.ModeDir|os.ModePerm)
	}
	c.converter.Initialise(videoDir, cachedDir)
}

func (c *CacheManager) GetAvailables() []string {
	return c.availables
}

func (c *CacheManager) GetManifest(filename string) (string, error) {
	var i int
	var err error
	err = nil
	/* check that filename has a match in availables */
	for i = 0; i < len(c.availables); i++ {
		if c.availables[i] == filename {
			break
		}
	}
	if i == len(c.availables) {
		return "", errors.New("File '" + filename + "' does not exist")
	}
	/* Test if filename has a match in cache */
	for i = 0; i < len(c.cached); i++ {
		if c.cached[i] == filename {
			break
		}
	}
	/* Try to build file if none is found in cache */
	if i == len(c.cached) {
		err = c.converter.Build(filename)
	}
	/* Return path only when one exist or build was successful */
	if i != len(c.cached) || err == nil {
		return filepath.Join(c.cachedDir, filename, "manifest.mpd"), nil
	}
	return "", err
}

func (c *CacheManager) GetChunk(filename string, chunk string) (string, error) {
	var i int
	var err error
	err = nil
	/* check that filename has a match in availables */
	for i = 0; i < len(c.availables); i++ {
		if c.availables[i] == filename {
			break
		}
	}
	if i == len(c.availables) {
		return "", errors.New("File '" + filename + "' does not exist")
	}
	/* Test if filename has a match in cached */
	for i := 0; i < len(c.cached); i++ {
		if c.cached[i] == filename {
			break
		}
	}
	/* Try to build file if none is found in cache */
	if i == len(c.cached) {
		err = c.converter.Build(filename)
	}
	/* Return path only when one exist or build was successful */
	if i != len(c.cached) || err == nil {
		return filepath.Join(c.cachedDir, filename, chunk), nil
	}
	return "", err
}
