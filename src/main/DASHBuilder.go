package main

import (
	"os"
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
	minVideoBandwidth := int(^uint(0) >> 1)
	maxVideoBandwidth := 0
	minAudioBandwidth := int(^uint(0) >> 1)
	maxAudioBandwidth := 0
	minWidth := int(^uint(0) >> 1)
	maxWidth := 0
	minHeight := int(^uint(0) >> 1)
	maxHeight := 0
	if len(b.tracks) > 0 {
		b.tracks[0].ComputePrivateInfos()
		duration = b.tracks[0].Duration()
		maxChunkDuration = b.tracks[0].MaxChunkDuration()
		if b.tracks[0].IsAudio() {
			minAudioBandwidth = b.tracks[0].Bandwidth()
			maxAudioBandwidth = b.tracks[0].Bandwidth()
		} else {
			minVideoBandwidth = b.tracks[0].Bandwidth()
			maxVideoBandwidth = b.tracks[0].Bandwidth()
			minWidth = b.tracks[0].Width()
			maxWidth = b.tracks[0].Width()
			minHeight = b.tracks[0].Height()
			maxHeight = b.tracks[0].Height()
		}
	}
	for i := 1; i < len(b.tracks); i++ {
		b.tracks[i].ComputePrivateInfos()
		if b.tracks[i].Duration() < duration {
			duration = b.tracks[i].Duration()
		}
		if b.tracks[i].MaxChunkDuration() > maxChunkDuration {
			maxChunkDuration = b.tracks[i].MaxChunkDuration()
		}
		if b.tracks[i].MinBufferTime() > minBufferTime {
			minBufferTime = b.tracks[i].MinBufferTime()
		}
		if b.tracks[i].IsAudio() {
			if minAudioBandwidth > b.tracks[i].Bandwidth() {
				minAudioBandwidth = b.tracks[i].Bandwidth()
			}
			if maxAudioBandwidth < b.tracks[i].Bandwidth() {
				maxAudioBandwidth = b.tracks[i].Bandwidth()
			}
		} else {
			if minVideoBandwidth > b.tracks[i].Bandwidth() {
				minVideoBandwidth = b.tracks[i].Bandwidth()
			}
			if maxVideoBandwidth < b.tracks[i].Bandwidth() {
				maxVideoBandwidth = b.tracks[i].Bandwidth()
			}
			if minWidth > b.tracks[i].Width() {
				minWidth = b.tracks[i].Width()
			}
			if maxWidth < b.tracks[i].Width() {
				maxWidth = b.tracks[i].Width()
			}
			if minHeight > b.tracks[i].Height() {
				minHeight = b.tracks[i].Height()
			}
			if maxHeight < b.tracks[i].Height() {
				maxHeight = b.tracks[i].Height()
			}
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
  profiles="urn:com:dashif:dash264">
  <Period>
    <AdaptationSet
      group="1"
      mimeType="video/mp4"
      par="16:9"`
	if (minVideoBandwidth != maxVideoBandwidth) {
		manifest += `
      minBandwidth="` + strconv.Itoa(minVideoBandwidth) + `"
      maxBandwidth="` + strconv.Itoa(maxVideoBandwidth) + `"`
	} else {
		manifest += `
      bandwidth="` + strconv.Itoa(minVideoBandwidth) + `"`
	}
	manifest += `
      minWidth="` + strconv.Itoa(minWidth) + `"
      maxWidth="` + strconv.Itoa(maxWidth) + `"
      minHeight="` + strconv.Itoa(minHeight) + `"
      maxHeight="` + strconv.Itoa(maxHeight) + `"
      segmentAlignment="true"
      startWithSAP="1">`
	adaptationDone := false
	for i := 0; i < len(b.tracks); i++ {
		if !b.tracks[i].IsAudio(){
			if !adaptationDone {
				manifest += b.tracks[i].BuildAdaptationSet()
				adaptationDone = true
			}
			manifest += b.tracks[i].BuildRepresentation()
		}
	}
	manifest += `
    </AdaptationSet>
    <AdaptationSet
      group="2"
      mimeType="audio/mp4"`
	if (minAudioBandwidth != maxAudioBandwidth) {
		manifest += `
      minBandwidth="` + strconv.Itoa(minAudioBandwidth) + `"
      maxBandwidth="` + strconv.Itoa(maxAudioBandwidth) + `"`
	} else {
		manifest += `
      bandwidth="` + strconv.Itoa(minAudioBandwidth) + `"`
	}
	manifest += `
      segmentAlignment="true">`
	adaptationDone = false
	for i := 0; i < len(b.tracks); i++ {
		if b.tracks[i].IsAudio() {
			if !adaptationDone {
				manifest += b.tracks[i].BuildAdaptationSet()
				adaptationDone = true
			}
			manifest += b.tracks[i].BuildRepresentation()
		}
	}
	manifest += `
    </AdaptationSet>`
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
func (b *DASHBuilder) Build(inPath string, filename string) error {
	var demuxer parser.Demuxer
	var err error
	var manifest string
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
