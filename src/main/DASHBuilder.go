package main

import (
	"os"
	"time"
	"math"
	"errors"
	"parser"
	"runtime"
	"strconv"
	"path/filepath"
	//"runtime/debug"
)

/* Structure to hold manifest info to avoid recomputing */
type ManifestInfos struct {
	bufferDepth float64
	duration float64
	maxChunkDuration float64
	minBufferTime float64
	minVideoBandwidth int
	maxVideoBandwidth int
	minAudioBandwidth int
	maxAudioBandwidth int
	minWidth int
	maxWidth int
	minHeight int
	maxHeight int
	imgWidth int
	imgHeight int
}

type DASHBuilder struct {
	tracks        []*parser.Track
	manifestInfos *ManifestInfos
	demuxer       *parser.Demuxer
	stop          bool
}

/* Structure used to store building specific information */
type DASHConverter struct {
	videoDir  string
	cachedDir string
	builders  map[string]*DASHBuilder
}

/* Initialise a DASHConverter structure */
func (b *DASHConverter) Initialise(videoDir string, cachedDir string) {
	b.videoDir = videoDir
	b.cachedDir = cachedDir
	b.builders = make(map[string]*DASHBuilder)
	parser.InitialiseDemuxers()
}

/* Compute manifest informations */
func (b *DASHBuilder) computeManifestInfos() *ManifestInfos {
	var res ManifestInfos
	res.bufferDepth = math.MaxFloat64
	res.duration = math.MaxFloat64
	res.maxChunkDuration = float64(0)
	res.minBufferTime = float64(0)
	res.minVideoBandwidth = int(^uint(0) >> 1)
	res.maxVideoBandwidth = 0
	res.minAudioBandwidth = int(^uint(0) >> 1)
	res.maxAudioBandwidth = 0
	res.minWidth = int(^uint(0) >> 1)
	res.maxWidth = 0
	res.minHeight = int(^uint(0) >> 1)
	res.maxHeight = 0
	for i := 0; i < len(b.tracks); i++ {
		b.tracks[i].ComputePrivateInfos()
		if b.tracks[i].BufferDepth() < res.bufferDepth {
			res.bufferDepth = b.tracks[i].BufferDepth()
		}
		if b.tracks[i].Duration() < res.duration {
			res.duration = b.tracks[i].Duration()
		}
		if b.tracks[i].MaxChunkDuration() > res.maxChunkDuration {
			res.maxChunkDuration = b.tracks[i].MaxChunkDuration()
		}
		if b.tracks[i].MinBufferTime() > res.minBufferTime {
			res.minBufferTime = b.tracks[i].MinBufferTime()
		}
		if b.tracks[i].IsAudio() {
			if res.minAudioBandwidth > b.tracks[i].Bandwidth() {
				res.minAudioBandwidth = b.tracks[i].Bandwidth()
			}
			if res.maxAudioBandwidth < b.tracks[i].Bandwidth() {
				res.maxAudioBandwidth = b.tracks[i].Bandwidth()
			}
		} else {
			if res.minVideoBandwidth > b.tracks[i].Bandwidth() {
				res.minVideoBandwidth = b.tracks[i].Bandwidth()
			}
			if res.maxVideoBandwidth < b.tracks[i].Bandwidth() {
				res.maxVideoBandwidth = b.tracks[i].Bandwidth()
			}
			if res.minWidth > b.tracks[i].Width() {
				res.minWidth = b.tracks[i].Width()
			}
			if res.maxWidth < b.tracks[i].Width() {
				res.maxWidth = b.tracks[i].Width()
			}
			if res.minHeight > b.tracks[i].Height() {
				res.minHeight = b.tracks[i].Height()
			}
			if res.maxHeight < b.tracks[i].Height() {
				res.maxHeight = b.tracks[i].Height()
			}
		}
	}
	return &res
}

/* Build manifest as a string */
func (b *DASHBuilder) buildManifest(isLive bool, thumbnailWidth int, thumbnailHeight int) (string, error) {
	if b.manifestInfos == nil {
		b.manifestInfos = b.computeManifestInfos()
	}
	manifest := `<?xml version="1.0" encoding="utf-8"?>
<MPD
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns="urn:mpeg:dash:schema:mpd:2011"
  xsi:schemaLocation="urn:mpeg:dash:schema:mpd:2011 http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd"`
	if isLive {
		manifest += `
  type="dynamic"
  minimumUpdatePeriod="PT2S"
  timeShiftBufferDepth="PT` + strconv.FormatFloat(b.manifestInfos.bufferDepth, 'f', -1, 64) + `S"
  maxSegmentDuration="PT` + strconv.FormatFloat(b.manifestInfos.maxChunkDuration, 'f', -1, 64) + `S"
  minBufferTime="PT` + strconv.FormatFloat(b.manifestInfos.minBufferTime, 'f', -1, 64) + `S"
  profiles="urn:mpeg:dash:profile:isoff-live:2011,urn:com:dashif:dash264,urn:hbbtv:dash:profile:isoff-live:2012">`
	} else {
		manifest += `
  type="static"
  mediaPresentationDuration="PT` + strconv.FormatFloat(b.manifestInfos.duration, 'f', -1, 64) + `S"
  maxSegmentDuration="PT` + strconv.FormatFloat(b.manifestInfos.maxChunkDuration, 'f', -1, 64) + `S"
  profiles="urn:com:dashif:dash264">`
	}
	manifest += `
  <Period>
    <AdaptationSet
      group="1"
      mimeType="video/mp4"
      par="16:9"`
	if (b.manifestInfos.minVideoBandwidth != b.manifestInfos.maxVideoBandwidth) {
		manifest += `
      minBandwidth="` + strconv.Itoa(b.manifestInfos.minVideoBandwidth) + `"
      maxBandwidth="` + strconv.Itoa(b.manifestInfos.maxVideoBandwidth) + `"`
	} else {
		manifest += `
      bandwidth="` + strconv.Itoa(b.manifestInfos.minVideoBandwidth) + `"`
	}
	manifest += `
      minWidth="` + strconv.Itoa(b.manifestInfos.minWidth) + `"
      maxWidth="` + strconv.Itoa(b.manifestInfos.maxWidth) + `"
      minHeight="` + strconv.Itoa(b.manifestInfos.minHeight) + `"
      maxHeight="` + strconv.Itoa(b.manifestInfos.maxHeight) + `"
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
	if (b.manifestInfos.minAudioBandwidth != b.manifestInfos.maxAudioBandwidth) {
		manifest += `
      minBandwidth="` + strconv.Itoa(b.manifestInfos.minAudioBandwidth) + `"
      maxBandwidth="` + strconv.Itoa(b.manifestInfos.maxAudioBandwidth) + `"`
	} else {
		manifest += `
      bandwidth="` + strconv.Itoa(b.manifestInfos.minAudioBandwidth) + `"`
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
    </AdaptationSet>
    <AdaptationSet
      group="3"
      mimeType="image/jpeg"
      segmentAlignment="true">
      <Representation width="` + strconv.Itoa(thumbnailWidth) + `" height="` + strconv.Itoa(thumbnailHeight) + `" codecs="jpeg">
        <SegmentTemplate media="thumbnails.bsi" duration="` + strconv.FormatFloat(b.manifestInfos.duration, 'f', -1, 64) + `" />
      </Representation>
    </AdaptationSet>`
	manifest += `
  </Period>
</MPD>`
	return manifest, nil
}

/* Routine launched for live streams */
func liveWorker(demuxer *parser.Demuxer, b *DASHBuilder, outPath string, filename string, cachedDir string, thumbnailWidth int, thumbnailHeight int, selected int) {
	for !b.stop {
		/* Extract and build chunk for each track */
		(*demuxer).ExtractChunk(&b.tracks, true)
		duration := b.buildChunks(outPath, selected)
		/* If we succeeded, update manifest */
		if duration > 0 && duration < math.MaxFloat64 {
			for i := 0; i < len(b.tracks); i++ {
				b.tracks[i].CleanForLive()
				b.tracks[i].CleanDirectory(filepath.Join(cachedDir, filename))
			}
			/* Build manifest */
			manifest, _ := b.buildManifest(true, thumbnailWidth, thumbnailHeight)
			/* Write it to file */
			f, _ := os.OpenFile(filepath.Join(cachedDir, filename, "manifest.mpd"), os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			/* Write generated manifest */
			f.WriteString(manifest)
			f.Close()
			/* Sleep until next chunk */
			time.Sleep(time.Duration(int64(duration * 1000000)) * time.Microsecond)
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
	(*demuxer).Close()
	b.cleanTracks()
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
func (b *DASHBuilder) buildChunks(outPath string, selected int) float64 {
	duration := math.MaxFloat64
	/* Call each track generation function */
	for i := 0; i < len(b.tracks); i++ {
		tmp, _ := b.tracks[i].BuildChunk(outPath)
		if selected == i && !b.tracks[i].IsEncrypted() {
			b.tracks[i].BuildImageChunk(filepath.Join(outPath, "thumbnails.bsi"))
		}
		if duration > tmp {
			duration = tmp
		}
		b.tracks[i].Clean()
	}
	/* Force GC to pass */
	runtime.GC()
	/*
           Release memory to OS. It calls GC again but because there is a finalizer for
	   samples, their GO part has not be freed by GC.
        */
	runtime.GC()
	//debug.FreeOSMemory()
	return duration
}


/* Build a DASH version of a file (manifest and chunks) */
func (c *DASHConverter) Build(inPath string, filename string, isLive bool) error {
	var demuxer parser.Demuxer
	var builder DASHBuilder
	var err error
	var manifest string
	if _, exists := c.builders[filename]; exists {
		return errors.New("File '" + filename + "' is already building !")
	}
	/* Get demuxer */
	demuxer, err = parser.OpenDemuxer(inPath)
	if err != nil { return err }
	/* Recover track from demuxer */
	err = demuxer.GetTracks(&builder.tracks)
	if err != nil { return err }
	/* Defer demuxer close and track clean up if anything goes wrong */
	if !isLive {
		defer demuxer.Close()
		defer builder.cleanTracks()
	}
	/* If we did not find any track, there is a problem */
	if len(builder.tracks) <= 0 {
		demuxer.Close()
		return errors.New("No tracks found !")
	}
	outPath := filepath.Join(c.cachedDir, filename)
	/* Initialise build for each track and build init chunk */
	/* Also select track for thumbnails */
	selected := 0
	currentWidth := 0
	currentHeight := 0
	for i := 0; i < len(builder.tracks); i++ {
		builder.tracks[i].InitialiseBuild(outPath)
		builder.tracks[i].BuildInit(outPath)
		if builder.tracks[i].IsAudio() {
			continue
		}
		if (currentWidth == 0 && currentHeight == 0) ||
		   (builder.tracks[i].Width() < currentWidth && builder.tracks[i].Width() >=  parser.THUMBNAIL_WIDTH) ||
		   (builder.tracks[i].Height() < currentHeight && builder.tracks[i].Height() >=  parser.THUMBNAIL_HEIGHT) ||
		   (builder.tracks[i].Width() > currentWidth && currentWidth < parser.THUMBNAIL_WIDTH) ||
		   (builder.tracks[i].Height() > currentHeight && currentHeight < parser.THUMBNAIL_HEIGHT) {
			selected = i
			currentWidth = builder.tracks[i].Width()
			currentHeight = builder.tracks[i].Height()
		}
	}
	if currentWidth > parser.THUMBNAIL_WIDTH {
		currentWidth =  parser.THUMBNAIL_WIDTH
	}
	if currentHeight > parser.THUMBNAIL_HEIGHT {
		currentHeight =  parser.THUMBNAIL_HEIGHT
	}
	/* While we have sample build chunks for each tracks */
	eof := false
	for !eof {
		eof = !demuxer.ExtractChunk(&builder.tracks, false)
		builder.buildChunks(outPath, selected)
	}
	/* If there is samples left in tracks */
	builder.buildChunks(outPath, selected)
	/* Build manifest */
	manifest, err = builder.buildManifest(isLive, currentWidth, currentHeight)
	if err != nil { return err }
	/* Write it to file */
	f, err := os.OpenFile(filepath.Join(c.cachedDir, filename, "manifest.mpd"), os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	/* Write generated manifest */
	_, err = f.WriteString(manifest)
	f.Close()
	if err == nil && isLive {
		go liveWorker(&demuxer, &builder, outPath, filename, c.cachedDir, currentWidth, currentHeight, selected)
		builder.demuxer = &demuxer
		c.builders[filename] = &builder
	}
	/* Force GC pass and memory release */
	runtime.GC()
	//debug.FreeOSMemory()
	return err
}

/* Stop a live generation thread */
func (c *DASHConverter) Stop(filename string) error {
	builder, exists := c.builders[filename]
	if !exists {
		return errors.New("File '" + filename + "' is not building !")
	}
	builder.stop = true
	delete(c.builders, filename)
	return nil
}
