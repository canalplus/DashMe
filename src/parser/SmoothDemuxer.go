package parser

import (
	"io"
	"sync"
	"utils"
	"bytes"
	"regexp"
	"strings"
	"strconv"
	"runtime"
	"net/http"
	"io/ioutil"
	"encoding/xml"
	"encoding/hex"
	"unicode/utf16"
	"path/filepath"
	"encoding/base64"
)

var (
	PlayReadyRegexp = regexp.MustCompile(`^.+<KID>([^<]+)</KID>.+$`)
)

type SmoothChunk struct {
	XMLName xml.Name `xml:"c"`
	Duration int64 `xml:"d,attr"`
	StartTime int64 `xml:"t,attr"`
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

type SmoothProtectionHeader struct {
	XMLName xml.Name `xml:"ProtectionHeader"`
	SystemId string `xml:"SystemID,attr"`
	Blob string `xml:",chardata"`
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
	Protection []SmoothProtectionHeader `xml:"Protection>ProtectionHeader"`
}

type SmoothAtomParser func (d *SmoothDemuxer, reader io.ReadSeeker, size int, t *Track)

type SmoothTrackInfo struct {
	baseDecodeTime int64
	bitrate        int
	urlTemplate    string
}

type SmoothDemuxer struct {
	manifestURL string
	baseURL string
	chunksURL map[int]*utils.Queue
	atomParsers map[string]SmoothAtomParser
	trackInfos map[int]*SmoothTrackInfo
	defaultSampleDuration int64
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
		tmp, _ := utils.AtomReadInt32(reader)
		d.defaultSampleDuration = int64(tmp)
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
	composition := int64(0)
	decodeTime := d.trackInfos[track.index].baseDecodeTime

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
			tmp, _ := utils.AtomReadInt32(reader)
			duration = int64(tmp)
		}
		if (flags & 0x200) > 0 {
			size, _ := utils.AtomReadInt32(reader)
			sample.size = CInt(size)
		}
		if (flags & 0x400) > 0 {
			reader.Seek(4, 1)
		}
		if (flags & 0x800) > 0 {
			tmp, _ := utils.AtomReadInt32(reader)
			composition = int64(tmp)
		}
		decodeTime += duration
		sample.dts = sample.pts + composition
		sample.keyFrame = (i == 0 || track.isAudio)
		sample.duration = duration
		track.appendSample(sample)
	}
	d.trackInfos[track.index].baseDecodeTime = decodeTime
}

func parseSMOOTHSENC(reader io.ReadSeeker, track *Track) {
	flags, _ := utils.AtomReadInt32(reader)
	if flags & 0x1 > 0 {
		reader.Seek(20, 1)
	}
	count, _ := utils.AtomReadInt32(reader)
	for i := 0; i < count; i++ {
		track.samples[i].encrypt = new(SampleEncryption)
		/* ASSUME size = 8 */
		track.samples[i].encrypt.initializationVector, _ = utils.AtomReadBuffer(reader, 8)
		if flags & 0x2 > 0 {
			track.encryptInfos.subEncrypt = true
			nb, _ := utils.AtomReadInt16(reader)
			for j := 0; j < nb; j++ {
				clear, _ := utils.AtomReadInt16(reader)
				encrypted, _ := utils.AtomReadInt32(reader)
				track.samples[i].encrypt.subEncrypt = append(track.samples[i].encrypt.subEncrypt, SubSampleEncryption{clear, encrypted})
			}
		}
	}
}

func (d *SmoothDemuxer) parseSmoothUUID(reader io.ReadSeeker, size int, track *Track) {
	high, _ := utils.AtomReadInt64(reader)
	low, _ := utils.AtomReadInt64(reader)
	if uint(high) == 0xa2394f525a9b4f14 && uint(low) == 0xa2446c427c648df4 {
		parseSMOOTHSENC(reader, track)
	} else {
		reader.Seek(int64(size - 24), 1)
	}
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

func (d *SmoothDemuxer) buildChunkURL(time int64, bitrate int, url string) string {
	suffix := strings.Replace(url, "{start time}", strconv.FormatInt(time, 10), 1)
	suffix = strings.Replace(suffix, "{bitrate}", strconv.Itoa(bitrate), 1)
	return d.baseURL + "/" + suffix
}

func (d *SmoothDemuxer) getChunksURL(bitrate int, url string, chunks []SmoothChunk) *utils.Queue {
	res := utils.Queue{}
	current := chunks[0].StartTime
	for i := 0; i < len(chunks); i++ {
		res.Push(d.buildChunkURL(current, bitrate, url))
		current += chunks[i].Duration
	}
	return &res
}

func buildWidevinePSS(key []byte) pss {
	blob := []byte{
		0x8, 0x1, 0x12, 0x10,
	}
	blob = append(blob, key...)
	return pss{"EDEF8BA979D64ACEA3C827DCD51D21ED", blob}
}

func extractKeyId(data []byte) []byte {
	shorts := make([]uint16, len(data)/2 - 5)
	for i := 0; i < len(data) - 10; i += 2 {
		shorts[(i)/2] = (uint16(data[i + 11]) << 8) | uint16(data[i + 10])
	}
	bytes, err := base64.StdEncoding.DecodeString(PlayReadyRegexp.FindStringSubmatch(string(utf16.Decode(shorts)))[1])
	if err != nil {
		return nil
	}
	key := hex.EncodeToString(bytes)
	res, err := hex.DecodeString(string([]uint8{
		key[6], key[7], key[4], key[5], key[2], key[3], key[0], key[1],
		key[10], key[11], key[8], key[9],
		key[14], key[15], key[12], key[13],
		key[16], key[17], key[18], key[19],
		key[20], key[21], key[22], key[23], key[24], key[25],
		key[26], key[27], key[28], key[29], key[30], key[31],
	}))
	if err != nil {
		return nil
	}
	return res
}

func (d *SmoothDemuxer) buildEncryptionInfos(headers []SmoothProtectionHeader) *EncryptionInfo {
	var res EncryptionInfo
	var i int
	for i = 0; i < len(headers); i++ {
		if strings.ToUpper(headers[i].SystemId) == "9A04F079-9840-4286-AB92-E65BE0885F95" {
			break
		}
	}
	blob, err := base64.StdEncoding.DecodeString(headers[i].Blob)
	if err != nil {
		return nil
	}
	keyId := extractKeyId(blob)
	res.keyId = hex.EncodeToString(keyId)
	if err != nil {
		return nil
	}
	res.pssList = append(res.pssList, buildWidevinePSS(keyId))
	res.pssList = append(res.pssList, pss{"9A04F07998404286AB92E65BE0885F95", blob})
	return &res
}

func (d *SmoothDemuxer) parseSmoothManifest(manifest *SmoothStreamingMedia, tracks *[]*Track) error {
	var track *Track
	if manifest.Timescale == 0 {
		manifest.Timescale = 10000000
	}
	acc := 0
	d.chunksURL = make(map[int]*utils.Queue)
	d.trackInfos = make(map[int]*SmoothTrackInfo)
	for i := 0; i < len(manifest.StreamIndexes); i++ {
		for j := 0; (manifest.StreamIndexes[i].Type == "audio" || manifest.StreamIndexes[i].Type == "video") && j < len(manifest.StreamIndexes[i].QualityInfos); j++ {
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
			if len(manifest.Protection) > 0 {
				track.encryptInfos = d.buildEncryptionInfos(manifest.Protection)
			}
			acc += 1
			track.SetTimeFields()
			*tracks = append(*tracks, track)
			d.chunksURL[track.index] = d.getChunksURL(manifest.StreamIndexes[i].QualityInfos[j].Bitrate, manifest.StreamIndexes[i].Url, manifest.StreamIndexes[i].ChunksInfos)
			d.trackInfos[track.index] = new(SmoothTrackInfo)
			d.trackInfos[track.index].baseDecodeTime = manifest.StreamIndexes[i].ChunksInfos[0].StartTime
			d.trackInfos[track.index].urlTemplate = manifest.StreamIndexes[i].Url
			d.trackInfos[track.index].bitrate = manifest.StreamIndexes[i].QualityInfos[j].Bitrate
			track.currentDuration = d.trackInfos[track.index].baseDecodeTime
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
	d.atomParsers["uuid"] = (*SmoothDemuxer).parseSmoothUUID
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
	/*for k := range d.baseDecodeTime {
		delete(d.baseDecodeTime, k)
	}*/
}

func (d *SmoothDemuxer) ExtractChunk(tracks *[]*Track, isLive bool) bool {
	var track *Track
	var waitList []chan error
	res := false
	for k := range d.chunksURL {
		res = res || !d.chunksURL[k].Empty()
		if d.chunksURL[k].Empty() && !isLive {
			continue
		} else if isLive {
			d.chunksURL[k].Push(d.buildChunkURL(d.trackInfos[k].baseDecodeTime, d.trackInfos[k].bitrate, d.trackInfos[k].urlTemplate))
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
