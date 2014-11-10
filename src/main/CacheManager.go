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

/* Structure used to store cache specific information */
type CacheManager struct {
	videoDir   string
	cachedDir  string
	availables []string
	cached     []string
	converter  DASHBuilder
	converting map[string]bool
}

/* Create internal buffer of files that can be converted */
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

/* Create internal buffer of files that are already converted */
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

/* Initialise a CacheManager structure */
func (c *CacheManager) Initialise(videoDir string, cachedDir string) {
	c.videoDir = videoDir
	c.BuildAvailables()
	c.cachedDir = cachedDir
	c.converting = make(map[string]bool)
	if (utils.FileExist(cachedDir)) {
		c.BuildCached()
	} else {
		os.MkdirAll(cachedDir, os.ModeDir|os.ModePerm)
	}
	c.converter.Initialise(videoDir, cachedDir)
}

/* Return list of files that can be converted */
func (c *CacheManager) GetAvailables() []string {
	return c.availables
}
func (c *CacheManager) buildIfNeeded(filename string) error {
	var i int
	var err error
	if c.converting[filename] {
		return errors.New("File '" + filename + "' is being generated")
	}
	/* check that filename has a match in availables */
	for i = 0; i < len(c.availables); i++ {
		if c.availables[i] == filename {
			break
		}
	}
	if i == len(c.availables) {
		return errors.New("File '" + filename + "' does not exist")
	}
	/* Test if filename has a match in cache */
	for i = 0; i < len(c.cached); i++ {
		if c.cached[i] == filename {
			break
		}
	}
	/* Try to build file if none is found in cache */
	if i == len(c.cached) {
		c.converting[filename] = true
		err = c.converter.Build(filename)
		delete(c.converting, filename)
	}
	if err != nil { return err }
	c.cached = append(c.cached, filename)
	return nil
}

/* Return manifest for a file, build it if it does not exist */
func (c *CacheManager) GetManifest(filename string) (string, error) {
	if err := c.buildIfNeeded(filename); err != nil {
		return "", err
	}
	return filepath.Join(c.cachedDir, filename, "manifest.mpd"), nil
}

/* Return a chunk from a file, build all if it does not exist */
func (c *CacheManager) GetChunk(filename string, chunk string) (string, error) {
	if err := c.buildIfNeeded(filename); err != nil {
		return "", err
	}
	return filepath.Join(c.cachedDir, filename, chunk), nil
}

func (c *CacheManager) AddFile(path string) error {
	c.availables = append(c.availables, utils.RemoveExtension(filepath.Base(path)))
	return nil
}

func (c *CacheManager) RemoveFile(path string) error {
	filename := utils.RemoveExtension(filepath.Base(path))
	/* If filename in cached remove directory and remove from list */
	for i := 0; i < len(c.cached); i++ {
		if c.cached[i] == filename {
			c.cached = append(c.cached[:i], c.cached[i + 1:]...)
			os.Remove(path)
			break
		}
	}
	/* Remove from availables list */
	for i := 0; i < len(c.availables); i++ {
		if c.availables[i] == filename {
			c.availables = append(c.availables[:i], c.availables[i + 1:]...)
		}
	}
	return nil
}

func (c *CacheManager) UpdateFile(path string) error {
	/* Just remove directory, this will force a generation next time */
	os.Remove(path)
	return nil
}
