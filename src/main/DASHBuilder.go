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
	parser.InitialiseDemuxers()
}

/* Build manifest file */
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

/* Clean builder private structures for GC */
func (b *DASHBuilder) cleanTracks() {
	for i := 0; i < len(b.tracks); i++ {
		b.tracks[i].Clean()
		b.tracks[i] = nil
	}
	b.tracks = b.tracks[:0]
}

/* Build one chunk for each track in the builder */
func (b *DASHBuilder) buildChunks(outPath string) {
	/* Call each track generation function */
	for i := 0; i < len(b.tracks); i++ {
		b.tracks[i].BuildChunk(outPath)
		b.tracks[i].Clean()
	}
	/* Force GC to pass */
	runtime.GC()
	/*
           Release memory to OS. It calls GC again but because there is a finalizer for
	   samples, there GO part has not be freed by GC.
        */
	debug.FreeOSMemory()
}

/* Build a DASH version of a file (manifest and chunks) */
func (b *DASHBuilder) Build(inPath string) error {
	var demuxer parser.Demuxer
	var err error
	var manifest string
	filename := utils.RemoveExtension(filepath.Base(inPath))
	/* Clean up if necessary */
	if len(b.tracks) > 0 {
		b.tracks = nil
	}
	/* Get demuxer */
	demuxer, err = parser.OpenDemuxer(inPath)
	if err != nil { return err }
	/* Recover track from demuxer */
	err = demuxer.GetTracks(&b.tracks)
	if err != nil { return err }
	/* Defer demuxer close and track clean up if anything goes wrong */
	defer demuxer.Close()
	defer b.cleanTracks()
	/* If we did not find any track, there is a problem */
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
	/* Force GC pass and memory release */
	debug.FreeOSMemory()
	return err
}
