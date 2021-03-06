// Copyright 2015 CANAL+ Group
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"utils"
	"parser"
	"errors"
	"path/filepath"
)

/*
  $CACHED_DIR/$FILENAME/manifest.mpd
  $CACHED_DIR/$FILENAME/chunk1.mp4
*/

type Available struct {
	Proto     string
	Path      string
	Name      string
	IsLive    bool
	Generated bool
	State     string
}

func (a Available) checkProto() bool {
	authorized := parser.GetAuthorizedProtocols()
	for _, proto := range authorized {
		if proto == a.Proto {
			return true
		}
	}
	return false
}

/* Structure used to store cache specific information */
type CacheManager struct {
	videoDir   string
	cachedDir  string
	availables []Available
	cached     []string
	converter  DASHConverter
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
		c.availables = append(c.availables, Available{
			Proto : "file",
			Name : utils.RemoveExtension(fi.Name()),
			Path : filepath.Join(c.videoDir, fi.Name()),
		})
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
		filename := utils.RemoveExtension(fi.Name())
		c.cached = append(c.cached, filename)
		for i := 0; i < len(c.availables); i++ {
			if c.availables[i].Name == filename {
				c.availables[i].Generated = true
			}
		}
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
func (c *CacheManager) GetAvailables() []Available {
	for i := 0; i < len(c.availables); i++ {
		if c.availables[i].Generated {
			c.availables[i].State = "generated"
		} else if c.converting[c.availables[i].Name] {
			c.availables[i].State = "generation"
		} else {
			c.availables[i].State = "not generated"
		}

	}
	return c.availables
}

/* Retrieve path to file according to stored filename */
func (c *CacheManager) getPathFromFilename(filename string) string {
	var i int
	/* Retrieve corresponding available */
	for i = 0; i < len(c.availables) && c.availables[i].Name != filename; i++ {}
	if i == len(c.availables) {
		return ""
	}
	return c.availables[i].Proto + "://" + c.availables[i].Path
}

/* Build DASH version of file if necessary */
func (c *CacheManager) buildIfNeeded(filename string) error {
	var i int
	var err error
	if c.converting[filename] {
		return errors.New("File '" + filename + "' is being generated")
	}
	/* Test if filename has a match in cache */
	for i = 0; i < len(c.cached); i++ {
		if c.cached[i] == filename {
			break
		}
	}
	/* We have a cached version, so we don't need a build */
	if i < len(c.cached) {
		return nil
	}
	/* check that filename has a match in availables */
	for i = 0; i < len(c.availables); i++ {
		if c.availables[i].Name == filename {
			break
		}
	}
	if i == len(c.availables) {
		return errors.New("File '" + filename + "' does not exist")
	}
	/* Try to build file */
	c.converting[filename] = true
	/* Get path to file */
	inPath := c.getPathFromFilename(filename)
	if inPath == "" { return errors.New("Can't find file for building !") }
	err = c.converter.Build(inPath, filename, c.availables[i].IsLive)
	delete(c.converting, filename)
	if err != nil { return err }
	c.availables[i].Generated = true
	c.cached = append(c.cached, filename)
	return nil
}

/* Build an element if it does not exist */
func (c *CacheManager) Build(filename string) error {
	return c.buildIfNeeded(filename)
}

/* Stop a demuxer for a live stream */
func (c *CacheManager) Stop(filename string) error {
	err := c.converter.Stop(filename)
	/* Remove directory */
	os.RemoveAll(filepath.Join(c.cachedDir, filename))
	/* Update available */
	for i := 0; i < len(c.cached); i++ {
		if c.cached[i] == filename {
			c.cached = append(c.cached[:i], c.cached[i + 1:]...)
			break
		}
	}
	for i := 0; i < len(c.availables); i++ {
		if c.availables[i].Name == filename {
			c.availables[i].Generated = false
			break
		}
	}
	return err
}

/* Return element for a file */
func (c *CacheManager) GetElement(filename string, element string) (string, error) {
	return filepath.Join(c.cachedDir, filename, element), nil
}

/* Add an available to the list for building */
func (c *CacheManager) AddAvailable(av Available) error {
	if !(av.checkProto()) {
		return errors.New("Incorrect protocol '" + av.Proto + "' !")
	}
	c.availables = append(c.availables, av)
	return nil
}

/* Add a file to the list of available file for building */
func (c *CacheManager) AddFile(path string) error {
	c.availables = append(c.availables, Available{
		Proto : "file",
		Name : utils.RemoveExtension(filepath.Base(path)),
		Path : path,
		IsLive : false,
	})
	return nil
}

/* Remove file from cache (if it has been generated) and from availables */
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
		if c.availables[i].Name == filename {
			c.availables = append(c.availables[:i], c.availables[i + 1:]...)
			break
		}
	}
	return nil
}

/* Signal that a file on disk has been updated and the generated cache is out of date */
func (c *CacheManager) UpdateFile(path string) error {
	/* Just remove directory, this will force a generation next time */
	os.Remove(path)
	return nil
}
