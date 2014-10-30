package main

import (
	"os"
	"io"
	"utils"
	"errors"
	"parsers"
	"path/filepath"
)

type Parser interface {
	Initialise()
	Probe(reader io.ReadSeeker, isDir bool) int
	Parse(reader io.ReadSeeker, tracks *[]parsers.Track, isDir bool) error
}

type DASHBuilder struct {
	videoDir  string
	cachedDir string
	parsers	  []Parser
	tracks    []parsers.Track
}

func (b *DASHBuilder) addParser(p Parser) {
	p.Initialise()
	b.parsers = append(b.parsers, p)
}

func (b *DASHBuilder) Initialise(videoDir string, cachedDir string) {
	b.videoDir = videoDir
	b.cachedDir = cachedDir
	b.tracks = nil
	b.addParser(parsers.MP4Parser{})
}

func (b *DASHBuilder) GetPathFromFilename(filename string) (string, bool) {
	var i int
	/* Open directory with videos */
	dir, err := os.Open(b.videoDir)
	if err != nil { return "", false }
	defer dir.Close()
	/* Read directory */
	fileInfos, err := dir.Readdir(-1)
	if err != nil { return "", false }
	/* Try to find a file corresponding to filename */
	for i = 0; i < len(fileInfos); i++ {
		if filename + filepath.Ext(fileInfos[i].Name()) == fileInfos[i].Name() {
			break
		}
	}
	/* If we did not find one, return empty string */
	if i == len(fileInfos) { return "", false }
	/* Compute and return path to file */
	res := filepath.Join(b.videoDir, fileInfos[i].Name())
	return res, utils.IsDirectory(res)
}

func (b *DASHBuilder) Build(filename string) error {
	var parser Parser
	var i      int
	var score  int
	/* Clean up if necessary */
	if len(b.tracks) > 0 {
		b.tracks = nil
	}
	/* Get path to file */
	path, isDir := b.GetPathFromFilename(filename)
	if path == "" { return errors.New("Can't find file for building !") }
	/* Open file */
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	/* Find best parser */
	currentScore := 0
	for i = 0; i < len(b.parsers); i++ {
		score = b.parsers[i].Probe(f, isDir)
		f.Seek(0, 0)
		if score > 50 && score > currentScore {
			currentScore = score;
			parser = b.parsers[i]
		}
	}
	/* If we don't have a good parser, return error */
	if currentScore < 50 { return errors.New("Can't find suitable parser for building !") }
	/* Parse file and recover tracks */
	err = parser.Parse(f, &(b.tracks), isDir)
	if err != nil { return err }
	for _, track := range b.tracks {
		track.BuildChunks(50, filepath.Join(b.cachedDir, filename))
	}
	/*
           TODO : Build manifest
         */
	return nil
}
