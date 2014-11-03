package main

import (
	"os"
	"io"
	"utils"
	"errors"
	"parser"
	"path/filepath"
)

type Parser interface {
	Initialise()
	Probe(reader io.ReadSeeker, isDir bool) int
	Parse(reader io.ReadSeeker, tracks *[]parser.Track, isDir bool) error
}

type DASHBuilder struct {
	videoDir  string
	cachedDir string
	tracks    []parser.Track
}

func (b *DASHBuilder) Initialise(videoDir string, cachedDir string) {
	b.videoDir = videoDir
	b.cachedDir = cachedDir
	b.tracks = nil
	parser.Initialise()
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
	var demuxer *parser.Demuxer
	var err error
	/* Clean up if necessary */
	if len(b.tracks) > 0 {
		b.tracks = nil
	}
	/* Get path to file */
	path, isDir := b.GetPathFromFilename(filename)
	if path == "" { return errors.New("Can't find file for building !") }
	/* Get demuxer */
	if (!isDir) {
		demuxer, err = parser.OpenDemuxer(path)
	} else {
		return errors.New("Can't parse multiple file format yet !")
	}
	/* Recover track from demuxer */
	err = demuxer.GetTracks(&b.tracks)
	if err != nil { return err }
	defer demuxer.CleanTracks(b.tracks)
	for _, track := range b.tracks {
		track.BuildChunks(50, filepath.Join(b.cachedDir, filename))
	}
	/*
           TODO : Build manifest
         */
	return nil
}
