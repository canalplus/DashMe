package main

import (
	"os"
	"utils"
	"errors"
	"parser"
	"runtime"
	"strconv"
	"path/filepath"
	"runtime/debug"
)

/* Structure used to store building specific information */
type DASHBuilder struct {
	videoDir  string
	cachedDir string
	tracks    []*parser.Track
}

/* Initialise a DASHBuilder structure */
func (b *DASHBuilder) Initialise(videoDir string, cachedDir string) {
	b.videoDir = videoDir
	b.cachedDir = cachedDir
	b.tracks = nil
	parser.Initialise()
}

/* Retrieve path to file according to stored filename */
func (b *DASHBuilder) GetPathFromFilename(filename string) string {
	var i int
	/* Open directory with videos */
	dir, err := os.Open(b.videoDir)
	if err != nil { return "" }
	defer dir.Close()
	/* Read directory */
	fileInfos, err := dir.Readdir(-1)
	if err != nil { return "" }
	/* Try to find a file corresponding to filename */
	for i = 0; i < len(fileInfos); i++ {
		if filename + filepath.Ext(fileInfos[i].Name()) == fileInfos[i].Name() {
			break
		}
	}
	/* If we did not find one, return empty string */
	if i == len(fileInfos) { return "" }
	/* Compute and return path to file */
	res := filepath.Join(b.videoDir, fileInfos[i].Name())
	if utils.IsDirectory(res) { return "" }
	return res
}

func (b *DASHBuilder) buildManifest() (string, error) {
	duration := float64(0)
	maxChunkDuration := float64(0)
	minBufferTime := float64(0)
	if len(b.tracks) > 0 {
		duration = b.tracks[0].Duration()
		maxChunkDuration = b.tracks[0].MaxChunkDuration()
	}
	for i := 1; i < len(b.tracks); i++ {
		if b.tracks[i].Duration() < duration {
			duration = b.tracks[i].Duration()
		}
		if b.tracks[i].MaxChunkDuration() > maxChunkDuration {
			maxChunkDuration = b.tracks[i].MaxChunkDuration()
		}
		if b.tracks[i].MinBufferTime() > minBufferTime {
			minBufferTime = b.tracks[i].MinBufferTime()
		}
	}
	manifest := `<?xml version="1.0" encoding="utf-8"?>
<MPD
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns="urn:mpeg:dash:schema:mpd:2011"
  xsi:schemaLocation="urn:mpeg:dash:schema:mpd:2011 http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd"
  type="static"
  mediaPresentationDuration="PT` + strconv.FormatFloat(duration, 'f', -1, 64) + `S"
  maxSegmentDuration="PT` + strconv.FormatFloat(maxChunkDuration, 'f', -1, 64) + `S"
  profiles="urn:mpeg:dash:profile:isoff-live:2011,urn:com:dashif:dash264,urn:hbbtv:dash:profile:isoff-live:2012">
  <Period>`
	for i := 0; i < len(b.tracks); i++ {
		manifest += b.tracks[i].BuildAdaptationSet()
	}
	manifest += `
  </Period>
</MPD>
`
	return manifest, nil
}

func (b *DASHBuilder) cleanTracks() {
	for i := 0; i < len(b.tracks); i++ {
		b.tracks[i].Clean()
		b.tracks[i] = nil
	}
	b.tracks = b.tracks[:0]
}

func (b *DASHBuilder) buildChunks(outPath string) {
	for i := 0; i < len(b.tracks); i++ {
		b.tracks[i].BuildChunk(outPath)
		b.tracks[i].Clean()
	}
	runtime.GC()
	debug.FreeOSMemory()
}

/* Build a DASH version of a file (manifest and chunks) */
func (b *DASHBuilder) Build(filename string) error {
	var demuxer *parser.Demuxer
	var err error
	var manifest string
	/* Clean up if necessary */
	if len(b.tracks) > 0 {
		b.tracks = nil
	}
	/* Get path to file */
	inPath := b.GetPathFromFilename(filename)
	if inPath == "" { return errors.New("Can't find file for building !") }
	/* Get demuxer */
	demuxer, err = parser.OpenDemuxer(inPath)
	/* Recover track from demuxer */
	err = demuxer.GetTracks(&b.tracks)
	if err != nil { return err }
	defer demuxer.Close()
	defer b.cleanTracks()
	if len(b.tracks) <= 0 { return errors.New("No tracks found !") }
	outPath := filepath.Join(b.cachedDir, filename)
	/* Initialise build for each track and build init chunk */
	for i := 0; i < len(b.tracks); i++ {
		b.tracks[i].InitialiseBuild(outPath)
		b.tracks[i].BuildInit(outPath)
	}
	/* While we have sample build chunks for each tracks */
	eof := false
	for !eof {
		eof = !demuxer.ExtractChunk(&b.tracks)
		b.buildChunks(outPath)
	}
	/* If there is samples left in tracks */
	b.buildChunks(outPath)
	/* Build manifest */
	manifest, err = b.buildManifest()
	if err != nil { return err }
	/* Write it to file */
	f, err := os.OpenFile(filepath.Join(b.cachedDir, filename, "manifest.mpd"), os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	/* Write generated manifest */
	_, err = f.WriteString(manifest)
	//utils.DisplayMemStats()
	debug.FreeOSMemory()
	return err
}
