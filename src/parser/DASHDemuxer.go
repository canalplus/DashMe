package parser

import (
	"fmt"
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
	"path/filepath"
)

var (
	DurationRegexp = regexp.MustCompile(`P((?P<year>[\d\.]+)Y)?((?P<month>[\d\.]+)M)?((?P<day>[\d\.]+)D)?(T((?P<hour>[\d\.]+)H)?((?P<minute>[\d\.]+)M)?((?P<second>[\d\.]+)S)?)?`)
)

type HTTPRequest struct {
	Url string
	Headers []struct { name, value string }
}

type DASHXMLSegment struct {
	XMLName xml.Name `xml:"S"`
	Duration int `xml:"d,attr"`
	Time int `xml:"t,attr"`
	Repetition int `xml:"r,attr"`
}

type DASHXMLSegmentTemplate struct {
	XMLName xml.Name `xml:"SegmentTemplate"`
	Timescale int `xml:"timescale,attr"`
	Initialization string `xml:"initialization,attr"`
	Media string `xml:"media,attr"`
	StartNumber int `xml:"startNumber,attr"`
	Segments []DASHXMLSegment `xml:"SegmentTimeline>S"`
}

type DASHXMLSegmentBase struct {
	XMLName xml.Name `xml:"SegmentBase"`
	Timescale int `xml:"timescale,attr"`
	Range string `xml:"indexRange,attr"`
	Initialization []DASHXMLInitialization `xml:"Initialization"`
}

type DASHXMLInitialization struct {
	XMLName xml.Name `xml:"Initialization"`
	Range string `xml:"range,attr"`
}

type DASHXMLRepresentation struct {
	XMLName xml.Name `xml:"Representation"`
	Id string `xml:"id,attr"`
	Bandwidth string `xml:"bandwidth,attr"`
	Codecs string `xml:"codecs,attr"`
	AudioSamplingRate string `xml:"audioSamplingRate,attr"`
	Width int `xml:"width,attr"`
	Height int `xml:"height,attr"`
	Sar int `xml:"sar,attr"`
	Base DASHXMLSegmentBase
	BaseURL string `xml:"BaseURL">`
}

type DASHXMLAdaptionSet struct {
	XMLName xml.Name `xml:"AdaptationSet"`
	Group string `xml:"group,attr"`
	MimeType string `xml:"mimeType,attr"`
	MinWidth int `xml:"minWidth,attr"`
	MaxWidth int `xml:"maxWidth,attr"`
	MinHeight int `xml:"minHeight,attr"`
	Maxheight int `xml:"maxHeight,attr"`
	Template DASHXMLSegmentTemplate
	Representations []DASHXMLRepresentation `xml:"Representation"`
}

type DASHXMLPeriod struct {
	XMLName xml.Name `xml:"Period"`
	BaseURL string `xml:"BaseURL"`
	AdaptationSets []DASHXMLAdaptionSet `xml:"AdaptationSet"`
}

type DASHManifest struct {
	XMLName xml.Name `xml:"MPD"`
	Duration string `xml:"mediaPresentationDuration,attr"`
	Period  DASHXMLPeriod
}

type DASHAtomParser func (d *DASHDemuxer, reader io.ReadSeeker, size int, t *Track)

/* Demuxer structure foàr DASH streaming parsing */
type DASHDemuxer struct {
	manifestURL string
	baseURL string
	atomParsers map[string]DASHAtomParser
	chunksURL map[int]*utils.Queue
	defaultSampleDuration int64
	mediaTime int64
	baseMediaDecodeTime int64
	mutex sync.Mutex
}

/* Initialise DASH demuxer */
func (d *DASHDemuxer) Open(path string) error {
	d.manifestURL = "http://" + path
	d.baseURL = "http://" + filepath.Dir(path)
	d.atomParsers = make(map[string]DASHAtomParser)
	d.atomParsers["mdat"] = (*DASHDemuxer).parseDASHMDAT
	d.atomParsers["mdhd"] = (*DASHDemuxer).parseDASHMDHD
	d.atomParsers["mvhd"] = (*DASHDemuxer).parseDASHMVHD
	d.atomParsers["stsd"] = (*DASHDemuxer).parseDASHSTSD
	d.atomParsers["mp4a"] = (*DASHDemuxer).parseDASHMP4A
	d.atomParsers["avc1"] = (*DASHDemuxer).parseDASHAVC1
	d.atomParsers["hdlr"] = (*DASHDemuxer).parseDASHHDLR
	d.atomParsers["tfhd"] = (*DASHDemuxer).parseDASHTFHD
	d.atomParsers["elst"] = (*DASHDemuxer).parseDASHELST
	d.atomParsers["trun"] = (*DASHDemuxer).parseDASHTRUN
	d.atomParsers["tfdt"] = (*DASHDemuxer).parseDASHTFDT
	d.atomParsers["pssh"] = (*DASHDemuxer).parseDASHPSSH
	d.atomParsers["encv"] = (*DASHDemuxer).parseDASHENCV
	d.atomParsers["enca"] = (*DASHDemuxer).parseDASHENCA
	d.atomParsers["tenc"] = (*DASHDemuxer).parseDASHTENC
	d.atomParsers["senc"] = (*DASHDemuxer).parseDASHSENC
	return nil
}

func containerDASHAtom(tag string) bool {
	return tag == "moov" || tag == "mvex" || tag == "trak" ||
		tag == "traf" || tag == "mdia" || tag == "minf" ||
		tag == "stbl" || tag == "moof" || tag == "edts" ||
		tag == "schi" || tag == "sinf" || tag == "schi"
}

/* Extract samples data from DASH MDAT atom */
func (d *DASHDemuxer) parseDASHMDAT(reader io.ReadSeeker, size int, track *Track) {
	for i := 0; i < len(track.samples); i++ {
		buffer, _ := utils.AtomReadBuffer(reader, int(track.samples[i].size))
		track.samples[i].data = CArray(buffer)
	}
}

/* Extract track specific time scale from DASH MDHD atom */
func (d *DASHDemuxer) parseDASHMDHD(reader io.ReadSeeker, size int, track *Track) {
	version, _ := utils.AtomReadInt8(reader)
	reader.Seek(3, 1)
	if version == 1 {
		reader.Seek(16, 1)
		track.timescale, _ = utils.AtomReadInt32(reader)
		reader.Seek(8, 1)
	} else {
		reader.Seek(8, 1)
		track.timescale, _ = utils.AtomReadInt32(reader)
		reader.Seek(4, 1)
	}
	reader.Seek(4, 1)
}

/* Extract global stream timescale from DASH MVHD atom */
func (d *DASHDemuxer) parseDASHMVHD(reader io.ReadSeeker, size int, track *Track) {
	version, _ := utils.AtomReadInt8(reader)
	reader.Seek(3, 1)
	if version == 1 {
		reader.Seek(16, 1)
	} else {
		reader.Seek(8, 1)
	}
	track.globalTimescale, _ = utils.AtomReadInt32(reader)
	if version == 1 {
		reader.Seek(88, 1)
	} else {
		reader.Seek(84, 1)
	}
}

/* Helper function for ESDS atom parsing */
func (d *DASHDemuxer) parseDASHMP4Descr(reader io.ReadSeeker, tag *int) int {
	*tag, _ = utils.AtomReadInt8(reader)
	len := 0
	count := 4
	for count > 0 {
		count--
		c, _ := utils.AtomReadInt8(reader);
		len = int((len << 7) | (c & 0x7f))
		if (c & 0x80 == 0) {
			break
		}
	}
	return len
}

/* Extract audio extradata from DASH ESDS atom */
func (d *DASHDemuxer) parseDASHESDS(reader io.ReadSeeker, track *Track) {
	var tag int
	base, _ := utils.CurrentOffset(reader)
	size, _ := utils.AtomReadInt32(reader)
	reader.Seek(8, 1)
	len := d.parseDASHMP4Descr(reader, &tag)
	if tag == 0x03 {
		reader.Seek(3, 1)
	} else {
		reader.Seek(2, 1)
	}
	len = d.parseDASHMP4Descr(reader, &tag)
	if tag == 0x04 {
		reader.Seek(13, 1)
		len = d.parseDASHMP4Descr(reader, &tag)
		if tag == 0x05 {
			track.extradata, _ = utils.AtomReadBuffer(reader, len)
		}
	}
	cur, _ := utils.CurrentOffset(reader)
	if cur - base < size {
		reader.Seek(int64(size - (cur - base)), 1)
	}
}

/* Extract audio info from DASH MP4A atom */
func (d *DASHDemuxer) parseDASHMP4A(reader io.ReadSeeker, size int, track *Track) {
	reader.Seek(24, 1)
	track.sampleRate, _ = utils.AtomReadInt32(reader)
	track.sampleRate = (track.sampleRate >> 16)
	d.parseDASHESDS(reader, track)
}

/* Extract video extradata from DASH AVCD atom */
func (d *DASHDemuxer) parseDASHAVCC(reader io.ReadSeeker, track *Track) {
	size, _ := utils.AtomReadInt32(reader)
	reader.Seek(4, 1)
	track.extradata, _ = utils.AtomReadBuffer(reader, size - 8)
}

/* Extract video info from DASH AVC1 atom */
func (d *DASHDemuxer) parseDASHAVC1(reader io.ReadSeeker, size int, track *Track) {
	reader.Seek(24, 1)
	track.width, _ = utils.AtomReadInt16(reader)
	track.height, _ = utils.AtomReadInt16(reader)
	reader.Seek(46, 1)
	track.bitsPerSample, _ = utils.AtomReadInt16(reader)
	track.colorTableId, _ = utils.AtomReadInt16(reader)
	d.parseDASHAVCC(reader, track)
}

/* Extract track type specific information from DASH STSD atom */
func (d *DASHDemuxer) parseDASHSTSD(reader io.ReadSeeker, size int, track *Track) {
	initPos, _ := utils.CurrentOffset(reader);
	reader.Seek(4, 1)
	entries, _ := utils.AtomReadInt32(reader)
	/* Iterate over each antries declared in atom */
	for i := 0; i < entries; i++ {
		subSize := 0
		tag, _ := utils.ReadAtomHeader(reader, &subSize)
		if d.atomParsers[tag] != nil {
			d.atomParsers[tag](d, reader, subSize, track)
		} else if !containerDASHAtom(tag) {
			reader.Seek(int64(subSize - 8), 1)
		}
	}
	cur, _ := utils.CurrentOffset(reader)
	reader.Seek(int64(initPos + size - 8 - cur), 1)
}

/* Extract track type from DASH HDLR atom */
func (d *DASHDemuxer) parseDASHHDLR(reader io.ReadSeeker, size int, track *Track) {
	reader.Seek(8, 1)
	tag, _ := utils.AtomReadTag(reader);
	if tag == "soun" {
		track.isAudio = true
	} else {
		track.isAudio = false
	}
	reader.Seek(int64(size - 20), 1)
}

/* Extract default sample duration from TFHD atom if declared */
func (d *DASHDemuxer) parseDASHTFHD(reader io.ReadSeeker, size int, track *Track) {
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

/* Extract media time offset from DASH ELST atom if present */
func (d *DASHDemuxer) parseDASHELST(reader io.ReadSeeker, size int, track *Track) {
	version, _ := utils.AtomReadInt8(reader)
	reader.Seek(3, 1)
	count, _ := utils.AtomReadInt32(reader)
	for i := 0; i < count; i++ {
		if version == 1 {
			tmp, _ := utils.AtomReadInt64(reader)
			if tmp == 0 {
				d.mediaTime, _ = utils.AtomReadInt64(reader)
			} else {
				d.mediaTime = 0
				reader.Seek(8, 1)
			}
		} else {
			tmp, _ := utils.AtomReadInt32(reader)
			if tmp == 0 {
				tmp, _ := utils.AtomReadInt32(reader)
				d.mediaTime = int64(tmp)
			} else {
				d.mediaTime = 0
				reader.Seek(4, 1)
			}

		}
		reader.Seek(4, 1)
	}
}

/* Extract encryption challenges from DASH PSSH atom */
func (d *DASHDemuxer) parseDASHPSSH(reader io.ReadSeeker, size int, track *Track) {
	var infos pss
	if track.encryptInfos == nil {
		track.encryptInfos = new(EncryptionInfo)
	}
	reader.Seek(4, 1)
	buf, _ := utils.AtomReadBuffer(reader, 16)
	infos.systemId = hex.EncodeToString(buf)
	length, _ := utils.AtomReadInt32(reader)
	infos.privateData, _ = utils.AtomReadBuffer(reader, length)
	track.encryptInfos.pssList = append(track.encryptInfos.pssList, infos)
}

/* Extract video info and encryption from DASH ENCV atom */
func (d *DASHDemuxer) parseDASHENCV(reader io.ReadSeeker, size int, track *Track) {
	base, _ := utils.CurrentOffset(reader)
	/* Extract video info */
	d.parseDASHAVC1(reader, size, track)
	cur, _ := utils.CurrentOffset(reader)
	/* Iterate over the other atoms and extract encryption info */
	for cur - base < (size - 8) {
		subSize := 0
		tag, _ := utils.ReadAtomHeader(reader, &subSize)
		if d.atomParsers[tag] != nil {
			d.atomParsers[tag](d, reader, subSize, track)
		} else if !containerDASHAtom(tag) {
			reader.Seek(int64(subSize - 8), 1)
		}
		cur, _ = utils.CurrentOffset(reader)
	}
}

/* Extract audio info and encryption from DASH ENCV atom */
func (d *DASHDemuxer) parseDASHENCA(reader io.ReadSeeker, size int, track *Track) {
	base, _ := utils.CurrentOffset(reader)
	/* Extract audio info */
	d.parseDASHMP4A(reader, size, track)
	cur, _ := utils.CurrentOffset(reader)
	/* Iterate over the other atoms and extract encryption info */
	for cur - base < (size - 8) {
		subSize := 0
		tag, _ := utils.ReadAtomHeader(reader, &subSize)
		if d.atomParsers[tag] != nil {
			d.atomParsers[tag](d, reader, subSize, track)
		} else if !containerDASHAtom(tag) {
			reader.Seek(int64(subSize - 8), 1)
		}
		cur, _ = utils.CurrentOffset(reader)
	}
}

/* Extract ecnryption keyId from DASH TENC atom */
func (d *DASHDemuxer) parseDASHTENC(reader io.ReadSeeker, size int, track *Track) {
	if track.encryptInfos == nil {
		track.encryptInfos = new(EncryptionInfo)
	}
	reader.Seek(8, 1)
	buf, _ := utils.AtomReadBuffer(reader, 16)
	track.encryptInfos.keyId = hex.EncodeToString(buf)
}

/* Extract sample encryption info from DASH SENC atom */
func (d *DASHDemuxer) parseDASHSENC(reader io.ReadSeeker, size int, track *Track) {
	flags, _ := utils.AtomReadInt32(reader)
	if flags & 0x1 > 0 {
		reader.Seek(20, 1)
	}
	count, _ := utils.AtomReadInt32(reader)
	/* Iterate over each declared sample */
	for i := 0; i < count; i++ {
		/* Add encryption infos for sample. TODO : handle size != 8 */
		track.samples[i].encrypt = new(SampleEncryption)
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

/* Called by GC to free sample data memory */
func dashPacketFinalizer(s *Sample) {
	CFree(s.data)
}

/* Extract sample information about the chunk from DASH TRUN atom */
func (d *DASHDemuxer) parseDASHTRUN(reader io.ReadSeeker, size int, track *Track) {
	flags, _ := utils.AtomReadInt32(reader)
	count, _ := utils.AtomReadInt32(reader)
	duration := d.defaultSampleDuration
	composition := int64(0)
	decodeTime := d.baseMediaDecodeTime
	/* Skip unused values if present */
	if (flags & 0x1) > 0 {
		reader.Seek(4, 1)
	}
	if (flags & 0x4) > 0 {
		reader.Seek(4, 1)
	}
	/* Iterate over each declared sample */
	for i := 0; i < count; i++ {
		/* Construct sample and set finalizer for memory liberation */
		sample := new(Sample)
		runtime.SetFinalizer(sample, dashPacketFinalizer)
		sample.pts = decodeTime
		/* Read duration from atom if present, use default duration otherwise */
		if (flags & 0x100) > 0 {
			tmp, _ := utils.AtomReadInt32(reader)
			duration = int64(tmp)
		}
		/* Read sample size */
		if (flags & 0x200) > 0 {
			size, _ := utils.AtomReadInt32(reader)
			sample.size = CInt(size)
		}
		if (flags & 0x400) > 0 {
			reader.Seek(4, 1)
		}
		/* Read sample composition offset to compute DTS */
		if (flags & 0x800) > 0 {
			tmp, _ := utils.AtomReadInt32(reader)
			composition = int64(tmp)
		}
		/* Increment current decodeTime with duration */
		decodeTime += duration
		/* Compute sample fields */
		sample.dts = sample.pts + composition - d.mediaTime
		sample.keyFrame = (i == 0 || track.isAudio)
		sample.duration = duration
		/* Append sample to track */
		track.appendSample(sample)
	}
}

/* Extract base timing info from DASH TFDT atom */
func (d *DASHDemuxer) parseDASHTFDT(reader io.ReadSeeker, size int, track *Track) {
	version, _ := utils.AtomReadInt8(reader)
	reader.Seek(3, 1)

	if (version == 1) {
		d.baseMediaDecodeTime, _ = utils.AtomReadInt64(reader)
	} else {
		tmp, _ := utils.AtomReadInt32(reader)
		d.baseMediaDecodeTime = int64(tmp)
	}

}

/* Parse a DASH chunk, either an init or data */
func (d *DASHDemuxer) parseDASHFile(request HTTPRequest, track *Track) error {
	client := new(http.Client)

	fmt.Printf("Url: %q, Headers %q \n", request.Url, request.Headers)

	var size int
	/* Retrieve chunk data */
	req, err := http.NewRequest("GET", request.Url, nil)
	if err != nil {
		return err
	}
	if request.Headers != nil {
		for _, h := range request.Headers {
			req.Header.Add(h.name, h.value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	/* Retrieve reader from dowloaded data */
	reader := bytes.NewReader(buffer)
	d.mutex.Lock()
	for {
		/* Read atoms until the end of file */
		tag, err := utils.ReadAtomHeader(reader, &size)
		if err != nil && err != io.EOF {
			d.mutex.Unlock()
			return err
		} else if size == 0 || err == io.EOF {
			d.mutex.Unlock()
			return nil
		}
		/* Call corresponding atom function or skip if there is none */
		if d.atomParsers[tag] != nil {
			d.atomParsers[tag](d, reader, size, track)
		} else if !containerDASHAtom(tag) {
			reader.Seek(int64(size - 8), 1)
		}
	}
	d.mutex.Unlock()
	return nil
}

/* Extract DASH duration from manifest */
func parseDASHDuration(duration string) float64 {
	var res float64
	match := DurationRegexp.FindStringSubmatch(duration)
	for i, name := range DurationRegexp.SubexpNames() {
		part := match[i]
		if i == 0 || name == "" || part == "" {
			continue
		}

		val, _ := strconv.ParseFloat(part, 64)
		switch name {
		case "year":
			res = res + (val * 3600 * 24 * 365.242)
		case "week":
			res = res + (val * 3600 * 24 * 7)
		case "day":
			res = res + (val * 3600 * 24)
		case "hour":
			res = res + (val * 3600)
		case "minute":
			res = res + (val * 60)
		case "second":
			res = res + val
		}
	}
	return res
}

/* Retrieve URL for all chunks passed as argument */
func (d *DASHDemuxer) getChunksURL(adaptationSet DASHXMLAdaptionSet, representation DASHXMLRepresentation) *utils.Queue {
	res := utils.Queue{}
	time := 0
	number := adaptationSet.Template.StartNumber
	/* Iterate over each adaptation set in manifest */
	for i := 0; i < len(adaptationSet.Template.Segments); i++ {
		if adaptationSet.Template.Segments[i].Time > 0 {
			time = adaptationSet.Template.Segments[i].Time
		}
		/* Iterate over each segment in representation */
		for j := 0; j < adaptationSet.Template.Segments[i].Repetition + 1; j++ {
			/* Build URL from template */
			name := adaptationSet.Template.Media
			name = strings.Replace(name, "$RepresentationID$", representation.Id, 1)
			name = strings.Replace(name, "$Bandwidth$", representation.Bandwidth, 1)
			name = strings.Replace(name, "$Time$", strconv.Itoa(time), 1)
			name = strings.Replace(name, "$Number$", strconv.Itoa(number), 1)
			res.Push(d.baseURL + "/" + name)
			/* Increment time from duration for next chunk */
			time += adaptationSet.Template.Segments[i].Duration
			number += 1
		}
	}
	return &res
}



/* Parse a SegmentTemplate track to return init segment url */
func (d *DASHDemuxer) parseSegmentTemplate(adaptationSet DASHXMLAdaptionSet, representation DASHXMLRepresentation) HTTPRequest {
	name := adaptationSet.Template.Initialization
	name = strings.Replace(name, "$RepresentationID$", representation.Id, 1)
	name = strings.Replace(name, "$Bandwidth$", 			 representation.Bandwidth, 1)

	request := HTTPRequest{Url: d.baseURL + name}
	return request
}

/* Parse a SegmentBase track to return init segment url */
func (d *DASHDemuxer) parseSegmentBase(adaptationSet DASHXMLAdaptionSet, representation DASHXMLRepresentation) HTTPRequest {
	request := HTTPRequest{Url: d.baseURL + representation.BaseURL}
	return request
}

/* Parse a DASH manifest and extract all tracks declared in it */
func (d *DASHDemuxer) parseDASHManifest(manifest *DASHManifest, tracks *[]*Track) error {
	var track *Track
	/* Retrieve duration */
	duration := parseDASHDuration(manifest.Duration)
	acc := 0
	d.chunksURL = make(map[int]*utils.Queue)
	/* Iterate over each adaptation set */
	for i := 0; i < len(manifest.Period.AdaptationSets); i++ {
		/* Iterate over each representation */
		for j := 0; j < len(manifest.Period.AdaptationSets[i].Representations); j++ {
			/* Add nex track and fill common info */
			track = new(Track)
			track.index = acc
			track.SetTimeFields()

			var initSegmentRequest HTTPRequest

			adaptationSet := manifest.Period.AdaptationSets[i]
			representation := adaptationSet.Representations[j]

			if adaptationSet.Template.XMLName.Local != "" {
				track.segmentType = "template"
				initSegmentRequest = d.parseSegmentTemplate(adaptationSet, representation)
			}

			if representation.Base.XMLName.Local != "" {
				track.segmentType = "base"
				initSegmentRequest = d.parseSegmentBase(adaptationSet, representation)
			}

			err := d.parseDASHFile(initSegmentRequest, track)
			if err != nil {
				fmt.Printf("error: %s \n", err)
				return err
			}

			track.duration = int(duration * float64(track.globalTimescale))
			track.bandwidth, _ = strconv.Atoi(representation.Bandwidth)
			if track.timescale == 0 {
				track.timescale = track.globalTimescale
			}

			acc++
			*tracks = append(*tracks, track)
			d.chunksURL[track.index] = d.getChunksURL(adaptationSet, representation)
		}
	}
	return nil
}

/* Retrieve all tracks from a DASH source */
func (d *DASHDemuxer) GetTracks(tracks *[]*Track) error {
	/* Retrieve manifest */
	var manifest DASHManifest
	resp, err := http.Get(d.manifestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	/* Transform XML to usable data structures */
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&manifest)
	if err != nil {
		return err
	}
	/* Parse manifest */
	err = d.parseDASHManifest(&manifest, tracks)
	if err != nil {
		return err
	}
	return err
}

/* Clean demuxer internal info */
func (d *DASHDemuxer) Close() {
	for k := range d.chunksURL {
		d.chunksURL[k].Clear()
		d.chunksURL[k] = nil
		delete(d.chunksURL, k)
	}
}

/* Extract samples from one chunk for each track declared */
func (d *DASHDemuxer) ExtractChunk(tracks *[]*Track, isLive bool) bool {
	var track *Track
	var waitList []chan error
	res := false
	/* Iterate over collection of chunk URL list */
	for k := range d.chunksURL {
		res = res || !d.chunksURL[k].Empty()
		if d.chunksURL[k].Empty() {
			/* No URL left and it is not live so do nothing */
			continue
		}
		track = nil
		/* Look for the corresponding track */
		for i := 0; i < len(*tracks); i++ {
			if (*tracks)[i].index == k {
				track = (*tracks)[i]
				break
			}
		}
		if track != nil {
			/* Retrieve URL to chunk and parallelised download and parsing */
			url := d.chunksURL[k].Pop().(string)
			request := HTTPRequest{Url: url}
			c := make(chan error)
			go func(c chan error, track *Track) {
				c <- d.parseDASHFile(request, track)
			}(c, track)
			waitList = append(waitList, c)
		}
	}
	/* Wait for all parsing routines to end */
	for i := 0; i < len(waitList); i++ {
		<- waitList[i]
		close(waitList[i])
	}
	return res
}
