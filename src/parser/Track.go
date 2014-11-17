package parser

import (
	"os"
	"fmt"
	"time"
	"utils"
	"errors"
	"strings"
	"strconv"
	"path/filepath"
	"encoding/hex"
)

/* Structure used to build chunks */
type Builder struct {
	builders map[string]AtomBuilder
}

/* Structure representing a track inside an input file */
type Track struct {
	index            int
	isAudio	         bool
	creationTime     int
	duration         int
	modificationTime int
	timescale        int
	globalTimescale  int
	width            int
	height           int
	sampleRate       int
	bitsPerSample    int
	colorTableId     int
	bandwidth        int
	codec            string
	currentDuration  int
	extradata        []byte
	samples          []*Sample
	chunksDuration   []int
	chunksSize       []int
	builder          Builder
}

/* Print track on stdout */
func (t *Track) Print() {
	fmt.Println("Track :")
	fmt.Println("\tindex : ", t.index)
	fmt.Println("\tisAudio : ", t.isAudio)
	fmt.Println("\tcreationTime : ", t.creationTime)
	fmt.Println("\tmodificationTime : ", t.modificationTime)
	fmt.Println("\tduration : ", t.duration)
	fmt.Println("\ttimescale : ", t.timescale)
	fmt.Println("\tgolbalTimescale : ", t.globalTimescale)
	fmt.Println("\twidth : ", t.width)
	fmt.Println("\theight : ", t.height)
	fmt.Println("\tsampleRate : ", t.sampleRate)
	fmt.Println("\tbitsPerSample : ", t.bitsPerSample)
	fmt.Println("\tcolorTableId : ", t.colorTableId)
	fmt.Println("\tbandwidth: ", t.bandwidth)
	fmt.Println("\tcodec: ", t.codec)
	fmt.Println("\tsamples count : ", len(t.samples))
}

/* Builder structure methods */

/* Initialise builder building function map */
func (b *Builder) Initialise() {
	b.builders = make(map[string]AtomBuilder)
	b.builders["ftyp"] = buildFTYP /**/
	b.builders["free"] = buildFREE /**/
	b.builders["moov"] = buildMOOV /**/
	b.builders["mvhd"] = buildMVHD /**/
	b.builders["mvex"] = buildMVEX /**/
	b.builders["trex"] = buildTREX /**/
	b.builders["trak"] = buildTRAK /**/
	b.builders["tkhd"] = buildTKHD /**/
	b.builders["mdia"] = buildMDIA /**/
	b.builders["mdhd"] = buildMDHD /**/
	b.builders["hdlr"] = buildHDLR /**/
	b.builders["minf"] = buildMINF /**/
	b.builders["dinf"] = buildDINF /**/
	b.builders["dref"] = buildDREF /**/
	b.builders["stbl"] = buildSTBL /**/
	b.builders["vmhd"] = buildVMHD /**/
	b.builders["smhd"] = buildSMHD /**/
	b.builders["stsd"] = buildSTSD /**/
	b.builders["stts"] = buildSTTS /**/
	b.builders["stsc"] = buildSTSC /**/
	b.builders["stco"] = buildSTCO /**/
	b.builders["stsz"] = buildSTSZ /**/
	b.builders["stss"] = buildSTSS /**/
	b.builders["styp"] = buildSTYP /**/
	b.builders["sidx"] = buildSIDX /**/
	b.builders["moof"] = buildMOOF /**/
	b.builders["mfhd"] = buildMFHD /**/
	b.builders["traf"] = buildTRAF /**/
	b.builders["tfhd"] = buildTFHD /**/
	b.builders["tfdt"] = buildTFDT /**/
	b.builders["trun"] = buildTRUN /**/
	b.builders["mdat"] = buildMDAT /**/
	b.builders["mp4a"] = buildMP4A /**/
	b.builders["esds"] = buildESDS /**/
	b.builders["avcC"] = buildAVCC /**/
	b.builders["avc1"] = buildAVC1 /**/
}

/* Build atoms from their tag passed as string */
func (b Builder) build(t Track, atoms ...string) ([]byte, error) {
	var buf []byte
	var tmp []byte
	var err error
	for i := 0; i < len(atoms); i++ {
		tmp, err = b.builders[atoms[i]](t)
		if err != nil { return nil, err }
		buf = append(buf, tmp...)
	}
	return buf, nil
}

/* Track structure methods */

/* Compute size of to be generated MOOF atom */
func (t *Track) computeMOOFSize() int {
	return 16 + /* MFHD size */
		8 + /* TRAF header size */
		16 + /* TFHD size*/
		16 + /* TFDT size */
		20 + 16 * len(t.samples) + /* TRUN size */
		8 /* MOOF header size */
}

/* Compute duration of to be generated chunk */
func (t *Track) computeChunkDuration() int {
	duration := 0
	for i := 0; i < len(t.samples); i++ {
		duration += t.samples[i].duration
	}
	return duration
}

/* Compute size of to be generated MDAT atom */
func (t *Track) computeMDATSize() int {
	acc := 0
	for i := 0; i < len(t.samples); i++ {
		acc += int(t.samples[i].size)
	}
	return acc + 8
}

/* Build atoms from their tag passed as string */
func (t *Track) buildAtoms(atoms ...string) ([]byte, error) {
	return t.builder.build(*t, atoms...)
}

/* Append a sample to the track sample slice */
func (t *Track) appendSample(sample *Sample) {
	t.samples = append(t.samples, sample)
}

/* Build an init chunk from internal information */
func (t *Track) buildInitChunk(path string) error {
	/* Build init chunk atoms */
	b, err := t.buildAtoms("ftyp", "free", "moov")
	if err != nil {
		return err
	}
	/* Open file */
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	/* Write generated atoms */
	_, err = f.Write(b)
	return err
}

/* Build a chunk with samples from internal information */
func (t *Track) buildSampleChunk(samples []*Sample, path string) (int, error) {
	/* Build chunk atoms */
	b, err := t.buildAtoms("styp", "free", "sidx", "moof", "mdat")
	if err != nil {
		return 0, err
	}
	/* Open file */
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	/* Write generated atoms */
	_, err = f.Write(b)
	t.chunksSize = append(t.chunksSize, len(b))
	return t.computeChunkDuration(), err
}

/* Initialise build for the track */
func (t *Track) InitialiseBuild(path string) error {
	t.builder = Builder{}
	/* Initialise builder */
	t.builder.Initialise()
	/* Create destination directory if it does not exist */
	if !utils.FileExist(path) {
		os.MkdirAll(path, os.ModeDir|os.ModePerm)
	} else if !utils.IsDirectory(path) {
		return errors.New("Path '" + path + "' is not a directory")
	}
	return nil
}

/* Build the init chunk for the track */
func (t *Track) BuildInit(path string) error {
	var typename string
	if (t.isAudio) {
		typename = "audio"
	} else {
		typename = "video"
	}
	return t.buildInitChunk(filepath.Join(path, "init_" + typename + strconv.Itoa(t.index) + ".mp4"))
}

/* Build a chunk for the track */
func (t *Track) BuildChunk(path string) error {
	/* Exit if there is nop sample : nothing to do ! */
	if (len(t.samples) <= 0) {
		return nil
	}
	var typename string
	/* Set string type name */
	if (t.isAudio) {
		typename = "audio"
	} else {
		typename = "video"
	}
	/* Generate chunk file name */
	filename := "chunk_" + typename + strconv.Itoa(t.index) + "_" + strconv.Itoa(t.currentDuration) + ".mp4"
	/* Generate one chunk */
	duration, err := t.buildSampleChunk(t.samples, filepath.Join(path, filename))
	/* Append duration to list for manifest generation */
	t.chunksDuration = append(t.chunksDuration, duration)
	/* Increment current duration for next chunk filename */
	t.currentDuration += duration
	return err
}

/* Build video track specific part of the manifest */
func (t *Track) buildVideoManifestAdaptation() string {
	res := `
      <SegmentTemplate
        timescale="` + strconv.Itoa(t.timescale) + `"
        initialization="init_$RepresentationID$.mp4"
        media="chunk_$RepresentationID$_$Time$.mp4"
        startNumber="1">
        <SegmentTimeline>`
	for i, duration := range t.chunksDuration {
		if i == 0 {
			res += `
          <S t="0" d="` + strconv.Itoa(duration) + `" />`
		} else {
			res += `
          <S d="` + strconv.Itoa(duration) + `" />`
		}
	}
	res += `
        </SegmentTimeline>
      </SegmentTemplate>`
	return res
}

func (t *Track) buildVideoManifestRepresentation() string {
	res := `
      <Representation
        id="video` + strconv.Itoa(t.index) + `"
        bandwidth="` + strconv.Itoa(t.bandwidth) + `"
        codecs="` + t.codec + `"
        width="` + strconv.Itoa(t.width) + `"
        height="` + strconv.Itoa(t.height) + `" />`
	return res
}


func (t *Track) buildAudioManifestAdaptation() string {
	res := `
      <SegmentTemplate
        timescale="` + strconv.Itoa(t.timescale) + `"
        initialization="init_$RepresentationID$.mp4"
        media="chunk_$RepresentationID$_$Time$.mp4"
        startNumber="1">
        <SegmentTimeline>`
	for i, duration := range t.chunksDuration {
		if i == 0 {
			res += `
          <S t="0" d="` + strconv.Itoa(duration) + `" />`
		} else {
			res += `
          <S d="` + strconv.Itoa(duration) + `" />`
		}
	}
	res += `
        </SegmentTimeline>
      </SegmentTemplate>`
	return res
}

/* Build audio track specific part of the manifest */
func (t *Track) buildAudioManifestRepresentation() string {
	res := `
      <Representation
        id="audio` + strconv.Itoa(t.index) + `"
        bandwidth="` + strconv.Itoa(t.bandwidth) + `"
        codecs="` + t.codec + `"
        audioSamplingRate="` + strconv.Itoa(t.sampleRate) + `">
        <AudioChannelConfiguration
          schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011"
          value="2">
        </AudioChannelConfiguration>
      </Representation>`
	return res
}

/* Compute bandwidth for a track */
func (t *Track) computeBandwidth() {
	if t.bandwidth > 0 {
		return
	}
	totalDuration := 0
	totalSize := 0
	for _, duration := range t.chunksDuration {
		totalDuration += duration
	}
	for _, size := range t.chunksSize {
		totalSize += size
	}
	totalDuration /= t.globalTimescale
	if totalDuration > 0 {
		t.bandwidth =  totalSize / totalDuration
	} else {
		t.bandwidth = 0
	}
}

/* Compute codec name from extradata for audio */
func (t *Track) extractAudioCodec() {
	t.codec = "mp4a.40.2"
}

/* Compute codec name from extradata for video */
func (t *Track) extractVideoCodec() {
	t.codec = "avc1." + strings.ToUpper(hex.EncodeToString(t.extradata[1:2]) + hex.EncodeToString(t.extradata[2:3]) + hex.EncodeToString(t.extradata[3:4]))
}

/* Compute codec name from extradata */
func (t *Track) extractCodec() {
	if t.isAudio {
		t.extractAudioCodec()
	} else {
		t.extractVideoCodec()
	}
}

/* Extract codec info and compute bandwidth */
func (t *Track) ComputePrivateInfos() {
	t.computeBandwidth()
	t.extractCodec()
}

/* Build track specific part of the manifest */
func (t *Track) BuildAdaptationSet() string {
	if t.isAudio {
		return t.buildAudioManifestAdaptation()
	} else {
		return t.buildVideoManifestAdaptation()
	}
}

func (t *Track) BuildRepresentation() string {
	if t.isAudio {
		return t.buildAudioManifestRepresentation()
	} else {
		return t.buildVideoManifestRepresentation()
	}
}

/* Set creationTime and modificationTime in Track structure */
func (t *Track) SetTimeFields() {
	t.creationTime = int(time.Since(time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)).Seconds())
	t.modificationTime = t.creationTime
}

/* Return track duration */
func (t *Track) Duration() float64 {
	return float64(t.duration) / float64(t.globalTimescale)
}

/* Return largest duration of segments in track */
func (t *Track) MaxChunkDuration() float64 {
	duration := 0
	for i := 0; i < len(t.chunksDuration); i++ {
		if t.chunksDuration[i] > duration {
			duration = t.chunksDuration[i]
		}
	}
	return float64(duration) / float64(t.globalTimescale)
}

/* Return largest duration of segments in track */
func (t *Track) MinBufferTime() float64 {
	size := 0
	for i := 0; i < len(t.samples); i++ {
		if int(t.samples[i].size) > size {
			size = int(t.samples[i].size)
		}
	}
	return float64(size) / float64(t.bandwidth)
}

/* Clean track private structures for GC */
func (t *Track) Clean() {
	for i := 0; i < len(t.samples); i++ {
		t.samples[i] = nil
	}
	t.samples = t.samples[:0]
	t.samples = nil
}

func (t *Track) IsAudio() bool {
	return t.isAudio
}

func (t *Track) Bandwidth() int {
	return t.bandwidth
}

func (t *Track) Width() int {
	return t.width
}

func (t *Track) Height() int {
	return t.height
}
