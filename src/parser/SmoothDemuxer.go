package parser

import (
	"io"
	//"fmt"
	"sync"
	"utils"
	"bytes"
	"strings"
	"strconv"
	"runtime"
	"net/http"
	"io/ioutil"
	"encoding/xml"
	"encoding/hex"
	"path/filepath"
)

type SmoothChunk struct {
	XMLName xml.Name `xml:"c"`
	Duration int `xml:"d,attr"`
}

type SmoothQualityLevel struct {
	XMLName xml.Name `xml:"QualityLevel"`
	Index int `xml:"Index,attr"`
	Bitrate int `xml:"Bitrate,attr"`
	MaxWidth int `xml:"MaxWidth,attr"`
	MaxHeight int `xml:"MaxHeight,attr"`
	FourCC string `xml:"FourCC,attr"`
	CodecPrivateData string `xml:"CodecPrivateData,attr"`
	AudioTag int `xml:"AudioTag,attr"`
	SamplingRate int `xml:"SamplingRate,attr"`
	BitsPerSample int `xml:"BitsPerSample,attr"`
	PacketSize int `xml:"PacketSize,attr"`
	Channels int `xml:"Channels,attr"`
}

type SmoothStreamIndex struct {
	XMLName xml.Name `xml:"StreamIndex"`
	Type string `xml:"Type,attr"`
	Url string `xml:"Url,attr"`
	Name string `xml:"Name,attr"`
	Chunks int `xml:"Chunks,attr"`
	QualityLevels int `xml:"QualityLevels,attr"`
	MaxWidth int `xml:"MaxWidth,attr"`
	MaxHeight int `xml:"MaxHeight,attr"`
	DisplayWidth int `xml:"DisplayWidth,attr"`
	DisplayHeight int `xml:"DisplayHeight,attr"`
	QualityInfos []SmoothQualityLevel `xml:"QualityLevel"`
	ChunksInfos []SmoothChunk `xml:"c"`
}

type SmoothStreamingMedia struct {
	XMLName xml.Name `xml:"SmoothStreamingMedia"`
	MajorVersion int `xml:"MajorVersion,attr"`
	MinorVersion int `xml:"MinorVersion,attr"`
	Timescale int `xml:"TimeScale,attr"`
	Duration int `xml:"Duration,attr"`
	IsLive bool `xml:"IsLive,attr"`
	LookaheadCount int `xml:"LookaheadCount,attr"`
	StreamIndexes []SmoothStreamIndex `xml:"StreamIndex"`
}

type SmoothAtomParser func (d *SmoothDemuxer, reader io.ReadSeeker, size int, t *Track)

type SmoothDemuxer struct {
	manifestURL string
	baseURL string
	chunksURL map[int]*utils.Queue
	atomParsers map[string]SmoothAtomParser
	baseDurations map[int]int
	defaultSampleDuration int
	mutex sync.Mutex
}

func containerSmoothAtom(tag string) bool {
	return tag == "moof" || tag == "traf"
}

func (d *SmoothDemuxer) parseSmoothMDAT(reader io.ReadSeeker, size int, track *Track) {
	for i := 0; i < len(track.samples); i++ {
		buffer, _ := utils.AtomReadBuffer(reader, int(track.samples[i].size))
		track.samples[i].data = CArray(buffer)
	}
}

func (d *SmoothDemuxer) parseSmoothTFHD(reader io.ReadSeeker, size int, track *Track) {
	flags, _ := utils.AtomReadInt32(reader)
	reader.Seek(4, 1)
	if (flags & 0x000001) > 0 {
		reader.Seek(8, 1)
	}
	if (flags & 0x000002) > 0 {
		reader.Seek(4, 1)
	}
	if (flags & 0x000008) > 0 {
		d.defaultSampleDuration, _ = utils.AtomReadInt32(reader)
	} else {
		d.defaultSampleDuration = 0
	}
	if (flags & 0x000010) > 0 {
		reader.Seek(4, 1)
	}
	if (flags & 0x000020) > 0 {
		reader.Seek(4, 1)
	}
}

/* Called by GC to free sample data memory */
func smoothPacketFinalizer(s *Sample) {
	CFree(s.data)
}

func (d *SmoothDemuxer) parseSmoothTRUN(reader io.ReadSeeker, size int, track *Track) {
	flags, _ := utils.AtomReadInt32(reader)
	count, _ := utils.AtomReadInt32(reader)
	duration := d.defaultSampleDuration
	composition := 0
	decodeTime := d.baseDurations[track.index]

	if (flags & 0x1) > 0 {
		reader.Seek(4, 1)
	}
	if (flags & 0x4) > 0 {
		reader.Seek(4, 1)
	}

	for i := 0; i < count; i++ {
		sample := new(Sample)
		runtime.SetFinalizer(sample, smoothPacketFinalizer)
		sample.pts = decodeTime
		if (flags & 0x100) > 0 {
			duration, _ = utils.AtomReadInt32(reader)
		}
		if (flags & 0x200) > 0 {
			size, _ := utils.AtomReadInt32(reader)
			sample.size = CInt(size)
		}
		if (flags & 0x400) > 0 {
			reader.Seek(4, 1)
		}
		if (flags & 0x800) > 0 {
			composition, _ = utils.AtomReadInt32(reader)
		}
		decodeTime += duration
		sample.dts = sample.pts + composition
		sample.keyFrame = (i == 0 || track.isAudio)
		sample.duration = duration
		track.appendSample(sample)
	}
	d.baseDurations[track.index] = decodeTime
}

func (d *SmoothDemuxer) buildAudioExtradata(privateData string, freq int, chans int) []byte {
	if privateData != "" {
		res, _ := hex.DecodeString(privateData)
		return res
	} else {
		freqs := []int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}
		i := 0
		for i = 0; i < len(freqs); i++ {
			if freqs[i] == freq {
				break
			}
		}
		res := 2 << 0x4
		res = (res | (i & 0x1F)) << 0x4
		res = (res | (chans & 0x1f)) << 0x3
		return []byte{ byte(res >> 8),  byte(res & 0xFF) }
	}
}

func (d *SmoothDemuxer) buildVideoExtradata(privateData string) []byte {
	split := strings.Split(privateData, "00000001")
	sps, _ := hex.DecodeString(split[1])
	spsLen := len(sps)
	pps, _ := hex.DecodeString(split[2])
	ppsLen := len(pps)
	prof, _ := strconv.ParseInt(split[1][2:4], 16, 8)
	cpro, _ := strconv.ParseInt(split[1][4:6], 16, 8)
	levl, _ := strconv.ParseInt(split[1][6:8], 16, 8)
	return append([]byte{
		1, byte(prof), byte(cpro), byte(levl), 0xFF, 0xE1,
		byte(spsLen >> 8), byte(spsLen & 0xFF),
	}, append(sps,
		append([]byte{
			1, byte(ppsLen >> 8), byte(ppsLen & 0xFF),
		}, pps...)...
	)...)
}

func (d *SmoothDemuxer) getChunksURL(bitrate int, url string, chunks []SmoothChunk) *utils.Queue {
	res := utils.Queue{}
	current := 0
	for i := 0; i < len(chunks); i++ {
		suffix := strings.Replace(url, "{start time}", strconv.Itoa(current), 1)
		suffix = strings.Replace(suffix, "{bitrate}", strconv.Itoa(bitrate), 1)
		res.Push(d.baseURL + "/" + suffix)
		current += chunks[i].Duration
	}
	return &res
}

func (d *SmoothDemuxer) parseSmoothManifest(manifest *SmoothStreamingMedia, tracks *[]*Track) error {
	var track *Track
	if manifest.Timescale == 0 {
		manifest.Timescale = 10000000
	}
	acc := 0
	d.chunksURL = make(map[int]*utils.Queue)
	d.baseDurations = make(map[int]int)
	for i := 0; i < len(manifest.StreamIndexes); i++ {
		for j := 0; j < len(manifest.StreamIndexes[i].QualityInfos); j++ {
			track = new(Track)
			track.index = acc
			track.isAudio = (manifest.StreamIndexes[i].Type == "audio")
			track.timescale = manifest.Timescale
			track.globalTimescale = manifest.Timescale
			track.duration = manifest.Duration
			track.bandwidth = manifest.StreamIndexes[i].QualityInfos[j].Bitrate
			if track.isAudio {
				track.sampleRate = manifest.StreamIndexes[i].QualityInfos[j].SamplingRate
				track.extradata = d.buildAudioExtradata(manifest.StreamIndexes[i].QualityInfos[j].CodecPrivateData, manifest.StreamIndexes[i].QualityInfos[j].SamplingRate, manifest.StreamIndexes[i].QualityInfos[j].Channels)
			} else {
				track.width = manifest.StreamIndexes[i].QualityInfos[j].MaxWidth
				track.height = manifest.StreamIndexes[i].QualityInfos[j].MaxHeight
				track.bitsPerSample = 0
				track.colorTableId = 24
				track.extradata = d.buildVideoExtradata(manifest.StreamIndexes[i].QualityInfos[j].CodecPrivateData)
			}
			acc += 1
			*tracks = append(*tracks, track)
			d.chunksURL[track.index] = d.getChunksURL(manifest.StreamIndexes[i].QualityInfos[j].Bitrate, manifest.StreamIndexes[i].Url, manifest.StreamIndexes[i].ChunksInfos)
			d.baseDurations[track.index] = 0
		}
	}
	return nil
}

func (d *SmoothDemuxer) Open(path string) error {
	d.manifestURL = "http://" + path
	d.baseURL = "http://" + filepath.Dir(path)
	d.atomParsers = make(map[string]SmoothAtomParser)
	d.atomParsers["mdat"] = (*SmoothDemuxer).parseSmoothMDAT
	d.atomParsers["tfhd"] = (*SmoothDemuxer).parseSmoothTFHD
	d.atomParsers["trun"] = (*SmoothDemuxer).parseSmoothTRUN
	return nil
}

func (d *SmoothDemuxer) GetTracks(tracks *[]*Track) error {
	var manifest SmoothStreamingMedia
	resp, err := http.Get(d.manifestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&manifest)
	if err != nil {
		return err
	}
	err = d.parseSmoothManifest(&manifest, tracks)
	if err != nil {
		return err
	}
	return err
}

func (d *SmoothDemuxer) parseSmoothChunk(url string, track *Track) error {
	var size int
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	d.mutex.Lock()
	for {
		tag, err := utils.ReadAtomHeader(reader, &size)
		if err != nil && err != io.EOF {
			d.mutex.Unlock()
			return err
		} else if size == 0 || err == io.EOF {
			d.mutex.Unlock()
			return nil
		}
		if d.atomParsers[tag] != nil {
			d.atomParsers[tag](d, reader, size, track)
		} else if !containerSmoothAtom(tag) {
			reader.Seek(int64(size - 8), 1)
		}
	}
	d.mutex.Unlock()
	return nil
}


func (d *SmoothDemuxer) Close() {
	for k := range d.chunksURL {
		d.chunksURL[k].Clear()
		d.chunksURL[k] = nil
		delete(d.chunksURL, k)
	}
	for k := range d.baseDurations {
		delete(d.baseDurations, k)
	}
}

func (d *SmoothDemuxer) ExtractChunk(tracks *[]*Track) bool {
	var track *Track
	var waitList []chan error
	res := false
	for k := range d.chunksURL {
		res = res || !d.chunksURL[k].Empty()
		if d.chunksURL[k].Empty() {
			continue
		}
		track = nil
		for i := 0; i < len(*tracks); i++ {
			if (*tracks)[i].index == k {
				track = (*tracks)[i]
				break
			}
		}
		if track != nil {
			url := d.chunksURL[k].Pop().(string)
			c := make(chan error)
			go func(c chan error, track *Track) {
				c <- d.parseSmoothChunk(url, track)
			}(c, track)
			waitList = append(waitList, c)
		}
	}
	for i := 0; i < len(waitList); i++ {
		<- waitList[i]
		close(waitList[i])
	}
	return res
}
