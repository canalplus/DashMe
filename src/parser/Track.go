package parser

import (
	"os"
	"fmt"
	"time"
	"utils"
	"errors"
	"strings"
	"strconv"
	"io/ioutil"
	"path/filepath"
	"encoding/hex"
)

/* Structure used to build chunks */
type Builder struct {
	builders map[string]AtomBuilder
}

/* Structure used to store encryption parts of a sample */
type SubSampleEncryption struct {
	clear     int
	encrypted int
}

/* Structure used to store encryption parameters of a sample */
type SampleEncryption struct {
	initializationVector []byte
	subEncrypt []SubSampleEncryption
}

/* Structure used to specified DRM systems supported byt the track */
type pss struct {
	systemId    string
	privateData []byte
}

/* Strcture used to store encryption specific info of the track */
type EncryptionInfo struct {
	pssList    []pss
	subEncrypt bool
	keyId      string
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
	initOffset			 int
	codec            string
	currentDuration  int64
	extradata        []byte
	samples          []*Sample
	chunksDuration   []int64
	chunksSize       []int
	chunksName       []string
	chunksRanges		 []*Range
	encryptInfos     *EncryptionInfo
	builder          Builder
	chunksDepth      int
	startTime        int64
	segmentType      string
}

/* Structure representing range in a segment base DASH */
type Range struct {
	ts 				int
	duration  int
	ranges 		string
	r         int
}

/* Print track on stdout */
func (s *Sample) Print() {
	fmt.Println("Sample :")
	fmt.Println("\tpts : ", s.pts)
	fmt.Println("\tdts : ", s.dts)
	fmt.Println("\tduration : ", s.duration)
	fmt.Println("\tkeyFrame : ", s.keyFrame)
	fmt.Println("\tdata : ", s.data)
	fmt.Println("\tsize : ", s.size)
	fmt.Println("\tencrypted : ", (s.encrypt != nil))
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
	fmt.Println("\tencrypted : ", (t.encryptInfos != nil))
	fmt.Println("\tsamples count : ", len(t.samples))
	fmt.Println("\tsegment type : ", t.segmentType)
	fmt.Println("\tinit offset : ", t.initOffset)
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
	b.builders["mp4a"] = buildMP4AENCA /**/
b.builders["esds"] = buildESDS /**/
	b.builders["avcC"] = buildAVCC /**/
	b.builders["avc1"] = buildAVC1ENCV /**/
	b.builders["sinf"] = buildSINF /**/
	b.builders["frma"] = buildFRMA /**/
	b.builders["schm"] = buildSCHM /**/
	b.builders["schi"] = buildSCHI /**/
	b.builders["tenc"] = buildTENC /**/
	b.builders["enca"] = buildMP4AENCA /**/
	b.builders["encv"] = buildAVC1ENCV /**/
	b.builders["senc"] = buildSENC
	b.builders["saiz"] = buildSAIZ
	b.builders["saio"] = buildSAIO
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
	res := 16 + /* MFHD size */
		8 + /* TRAF header size */
		16 + /* TFHD size*/
		20 + /* TFDT size */
		20 + 16 * len(t.samples) + /* TRUN size */
		8 /* MOOF header size */
	if t.encryptInfos != nil {
		sencSize := 0
		for i := 0; i < len(t.samples); i++ {
			sencSize += len(t.samples[i].encrypt.initializationVector)
			if t.encryptInfos.subEncrypt {
				sencSize += 2 + 6 * len(t.samples[i].encrypt.subEncrypt)
			}
		}
		res += 16 + sencSize + /* SENC size */
			17 + len(t.samples) + /* SAIZ size */
			20 /* SAIO size */
	}
	return res
}

/* Compute duration of to be generated chunk */
func (t *Track) computeChunkDuration() int64 {
	duration := int64(0)
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
func (t *Track) buildSampleChunk(samples []*Sample, path string) (int64, error) {
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
	t.chunksName = append(t.chunksName, filepath.Base(path))
	return t.computeChunkDuration(), err
}

/* Initialise build for the track */
func (t *Track) InitialiseBuild(path string) error {
	t.builder = Builder{}
	t.chunksDepth = 30
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
func (t *Track) BuildChunk(path string) (float64, error) {
	/* Exit if there is no sample : nothing to do ! */
	if (len(t.samples) <= 0) {
		return 0, nil
	}
	var typename string
	/* Set string type name */
	if (t.isAudio) {
		typename = "audio"
	} else {
		typename = "video"
	}
	/* Generate chunk file name */
	filename := "chunk_" + typename + strconv.Itoa(t.index) + "_" + strconv.FormatInt(t.currentDuration, 10) + ".mp4"
	/* Generate one chunk */
	duration, err := t.buildSampleChunk(t.samples, filepath.Join(path, filename))
	/* Append duration to list for manifest generation */
	t.chunksDuration = append(t.chunksDuration, duration)
	/* Increment current duration for next chunk filename */
	t.currentDuration += duration
	return float64(duration) / float64(t.globalTimescale), err
}

/* Build video track adaptation part of the manifest */
func (t *Track) buildVideoManifestAdaptation() string {
	chunksDuration := int64(0)
	for i:= 0; i < len(t.chunksDuration); i++ {
		chunksDuration += t.chunksDuration[i]
	}
	res := `
      <SegmentTemplate
        timescale="` + strconv.Itoa(t.timescale) + `"
        initialization="init_$RepresentationID$.mp4"
        media="chunk_$RepresentationID$_$Time$.mp4"
        startNumber="1">
        <SegmentTimeline>`
	/* Build each chunk entry */
	for i, duration := range t.chunksDuration {
		if i == 0 {
			res += `
          <S t="` + strconv.FormatInt(t.currentDuration - chunksDuration, 10) + `" d="` + strconv.FormatInt(duration, 10) + `" />`
		} else {
			res += `
          <S d="` + strconv.FormatInt(duration, 10) + `" />`
		}
	}
	res += `
        </SegmentTimeline>
      </SegmentTemplate>`
	return res
}

/* Build video representation part of the manifest */
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

/* Build audio adpation part of the manifest */
func (t *Track) buildAudioManifestAdaptation() string {
	chunksDuration := int64(0)
	for i:= 0; i < len(t.chunksDuration); i++ {
		chunksDuration += t.chunksDuration[i]
	}
	res := `
      <SegmentTemplate
        timescale="` + strconv.Itoa(t.timescale) + `"
        initialization="init_$RepresentationID$.mp4"
        media="chunk_$RepresentationID$_$Time$.mp4"
        startNumber="1">
        <SegmentTimeline>`
	/* Build each chunk entry */
	for i, duration := range t.chunksDuration {
		if i == 0 {
			res += `
          <S t="` + strconv.FormatInt(t.currentDuration - chunksDuration, 10) + `" d="` + strconv.FormatInt(duration, 10) + `" />`
		} else {
			res += `
          <S d="` + strconv.FormatInt(duration, 10) + `" />`
		}
	}
	res += `
        </SegmentTimeline>
      </SegmentTemplate>`
	return res
}

/* Build audio representation part of the manifest */
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
	totalDuration := int64(0)
	totalSize := int64(0)
	/* Accumulate duration and size for all chunks */
	for _, duration := range t.chunksDuration {
		totalDuration += duration
	}
	for _, size := range t.chunksSize {
		totalSize += int64(size)
	}
	/* Rescale duration to seconds */
	totalDuration /= int64(t.globalTimescale)
	if totalDuration > 0 {
		/* Return bandwidth */
		t.bandwidth =  int(totalSize / totalDuration)
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

/* Build track adaptation part of the manifest */
func (t *Track) BuildAdaptationSet() string {
	if t.isAudio {
		return t.buildAudioManifestAdaptation()
	} else {
		return t.buildVideoManifestAdaptation()
	}
}

/* Build track representation part of the manifest */
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
	duration := int64(0)
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

/* Partially clean internal list in order to generate an up to date manifest */
func (t *Track) CleanForLive() {
	if len(t.chunksDuration) > t.chunksDepth {
		t.chunksDuration = t.chunksDuration[len(t.chunksDuration) - (t.chunksDepth + 1):]
	}
	if len(t.chunksSize) > t.chunksDepth {
		t.chunksSize = t.chunksSize[len(t.chunksSize) - (t.chunksDepth + 1):]
	}
	if len(t.chunksName) > t.chunksDepth {
		t.chunksName = t.chunksName[len(t.chunksName) - (t.chunksDepth + 1):]
	}
}

/* Return if the track is audio or not */
func (t *Track) IsAudio() bool {
	return t.isAudio
}

/* Return track bandwidth */
func (t *Track) Bandwidth() int {
	return t.bandwidth
}

/* Return track width (0 for audio) */
func (t *Track) Width() int {
	return t.width
}

/* Return track height (0 for audio) */
func (t *Track) Height() int {
	return t.height
}

/* Compute track buffer depth */
func (t *Track) BufferDepth() float64 {
	duration := int64(0)
	for i := 0; i < len(t.chunksDuration); i++ {
		if t.chunksDuration[i] > duration {
			duration = t.chunksDuration[i]
		}
	}
	return float64(duration) / float64(len(t.chunksDuration))
}

/* Clean track directory for unreferenced file in manifest */
func (t *Track) CleanDirectory(path string) {
	var name string
	if t.isAudio {
		name = "audio"
	} else {
		name = "video"
	}
	files, _ := ioutil.ReadDir(path)
	for _, fi := range files {
		if strings.Contains(fi.Name(), "chunk_" + name + strconv.Itoa(t.index)) {
			i := 0
			for ; i < len(t.chunksName); i++ {
				if fi.Name() == t.chunksName[i] {
					break
				}
			}
			if i == len(t.chunksName) {
				os.Remove(filepath.Join(path, fi.Name()))
			}
		}
	}
}
