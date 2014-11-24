# parser
--
    import "."


## Usage

```go
var (
	DurationRegexp = regexp.MustCompile(`P((?P<year>[\d\.]+)Y)?((?P<month>[\d\.]+)M)?((?P<day>[\d\.]+)D)?(T((?P<hour>[\d\.]+)H)?((?P<minute>[\d\.]+)M)?((?P<second>[\d\.]+)S)?)?`)
)
```

```go
var (
	PlayReadyRegexp = regexp.MustCompile(`^.+<KID>([^<]+)</KID>.+$`)
)
```

#### func  CArray

```go
func CArray(buffer []byte) unsafe.Pointer
```

#### func  CFree

```go
func CFree(ptr unsafe.Pointer)
```

#### func  CInt

```go
func CInt(val int) C.int
```

#### func  FFMPEGInitialise

```go
func FFMPEGInitialise() error
```
Called when starting the program, initialise FFMPEG demuxers

#### func  GetAuthorizedProtocols

```go
func GetAuthorizedProtocols() []string
```

#### func  InitialiseDemuxers

```go
func InitialiseDemuxers() error
```
Initialise specifics for each demuxer interface

#### func  TimebaseRescale

```go
func TimebaseRescale(val int, tbIn int, tbOut int) int
```

#### type AtomBuilder

```go
type AtomBuilder func(t Track) ([]byte, error)
```

Function type used for atom generation

#### type Builder

```go
type Builder struct {
}
```

Structure used to build chunks

#### func (*Builder) Initialise

```go
func (b *Builder) Initialise()
```
Initialise builder building function map

#### type DASHAtomParser

```go
type DASHAtomParser func(d *DASHDemuxer, reader io.ReadSeeker, size int, t *Track)
```


#### type DASHDemuxer

```go
type DASHDemuxer struct {
}
```


#### func (*DASHDemuxer) Close

```go
func (d *DASHDemuxer) Close()
```

#### func (*DASHDemuxer) ExtractChunk

```go
func (d *DASHDemuxer) ExtractChunk(tracks *[]*Track, isLive bool) bool
```

#### func (*DASHDemuxer) GetTracks

```go
func (d *DASHDemuxer) GetTracks(tracks *[]*Track) error
```

#### func (*DASHDemuxer) Open

```go
func (d *DASHDemuxer) Open(path string) error
```

#### type DASHManifest

```go
type DASHManifest struct {
	XMLName  xml.Name `xml:"MPD"`
	Duration string   `xml:"mediaPresentationDuration,attr"`
	Period   DASHXMLPeriod
}
```


#### type DASHXMLAdaptionSet

```go
type DASHXMLAdaptionSet struct {
	XMLName         xml.Name `xml:"AdaptationSet"`
	Group           string   `xml:"group,attr"`
	MimeType        string   `xml:"mimeType,attr"`
	MinWidth        int      `xml:"minWidth,attr"`
	MaxWidth        int      `xml:"maxWidth,attr"`
	MinHeight       int      `xml:"minHeight,attr"`
	Maxheight       int      `xml:"maxHeight,attr"`
	Template        DASHXMLSegmentTemplate
	Representations []DASHXMLRepresentation `xml:"Representation"`
}
```


#### type DASHXMLPeriod

```go
type DASHXMLPeriod struct {
	XMLName        xml.Name             `xml:"Period"`
	BaseURL        string               `xml:"BaseURL"`
	AdaptationSets []DASHXMLAdaptionSet `xml:"AdaptationSet"`
}
```


#### type DASHXMLRepresentation

```go
type DASHXMLRepresentation struct {
	XMLName           xml.Name `xml:"Representation"`
	Id                string   `xml:"id,attr"`
	Bandwidth         string   `xml:"bandwidth,attr"`
	Codecs            string   `xml:"codecs,attr"`
	AudioSamplingRate string   `xml:"audioSamplingRate,attr"`
	Width             int      `xml:"width,attr"`
	Height            int      `xml:"height,attr"`
}
```


#### type DASHXMLSegment

```go
type DASHXMLSegment struct {
	XMLName    xml.Name `xml:"S"`
	Duration   int      `xml:"d,attr"`
	Time       int      `xml:"t,attr"`
	Repetition int      `xml:"r,attr"`
}
```


#### type DASHXMLSegmentTemplate

```go
type DASHXMLSegmentTemplate struct {
	XMLName        xml.Name         `xml:"SegmentTemplate"`
	Timescale      int              `xml:"timescale,attr"`
	Initialization string           `xml:"initialization,attr"`
	Media          string           `xml:"media,attr"`
	StartNumber    int              `xml:"startNumber,attr"`
	Segments       []DASHXMLSegment `xml:"SegmentTimeline>S"`
}
```


#### type Demuxer

```go
type Demuxer interface {
	Open(path string) error
	GetTracks(tracks *[]*Track) error
	Close()
	ExtractChunk(tracks *[]*Track, isLive bool) bool
}
```


#### func  OpenDemuxer

```go
func OpenDemuxer(path string) (Demuxer, error)
```
Open a file from a path and initialize a demuxer structure

#### type DemuxerConstructor

```go
type DemuxerConstructor func() Demuxer
```


#### type EncryptionInfo

```go
type EncryptionInfo struct {
}
```

Strcture used to store encryption specific info of the track

#### type FFMPEGDemuxer

```go
type FFMPEGDemuxer struct {
}
```

Structure used to reference FFMPEG C AVFormatContext structure

#### func (*FFMPEGDemuxer) AppendSample

```go
func (d *FFMPEGDemuxer) AppendSample(track *Track, stream *C.AVStream)
```
Append a sample to a track

#### func (*FFMPEGDemuxer) Close

```go
func (d *FFMPEGDemuxer) Close()
```
Close demuxer and free FFMPEG specific data

#### func (*FFMPEGDemuxer) ExtractChunk

```go
func (d *FFMPEGDemuxer) ExtractChunk(tracks *[]*Track, isLive bool) bool
```
Extract one chunk for each track from input, size of the chunk depends on the
first

    video track found.

#### func (*FFMPEGDemuxer) GetTracks

```go
func (d *FFMPEGDemuxer) GetTracks(tracks *[]*Track) error
```
Retrieve tracks from previously opened file using FFMPEG

#### func (*FFMPEGDemuxer) Open

```go
func (d *FFMPEGDemuxer) Open(path string) error
```
Open FFMPEG specific demuxer

#### type Sample

```go
type Sample struct {
}
```

Structure used to store a Sample for chunk generation

#### func (*Sample) GetData

```go
func (s *Sample) GetData() []byte
```
Return byte data from a sample

#### func (*Sample) Print

```go
func (s *Sample) Print()
```
Print track on stdout

#### type SampleEncryption

```go
type SampleEncryption struct {
}
```

Structure used to store encryption parameters of a sample

#### type SmoothAtomParser

```go
type SmoothAtomParser func(d *SmoothDemuxer, reader io.ReadSeeker, size int, t *Track)
```


#### type SmoothChunk

```go
type SmoothChunk struct {
	XMLName   xml.Name `xml:"c"`
	Duration  int64    `xml:"d,attr"`
	StartTime int64    `xml:"t,attr"`
}
```


#### type SmoothDemuxer

```go
type SmoothDemuxer struct {
}
```


#### func (*SmoothDemuxer) Close

```go
func (d *SmoothDemuxer) Close()
```

#### func (*SmoothDemuxer) ExtractChunk

```go
func (d *SmoothDemuxer) ExtractChunk(tracks *[]*Track, isLive bool) bool
```

#### func (*SmoothDemuxer) GetTracks

```go
func (d *SmoothDemuxer) GetTracks(tracks *[]*Track) error
```

#### func (*SmoothDemuxer) Open

```go
func (d *SmoothDemuxer) Open(path string) error
```

#### type SmoothProtectionHeader

```go
type SmoothProtectionHeader struct {
	XMLName  xml.Name `xml:"ProtectionHeader"`
	SystemId string   `xml:"SystemID,attr"`
	Blob     string   `xml:",chardata"`
}
```


#### type SmoothQualityLevel

```go
type SmoothQualityLevel struct {
	XMLName          xml.Name `xml:"QualityLevel"`
	Index            int      `xml:"Index,attr"`
	Bitrate          int      `xml:"Bitrate,attr"`
	MaxWidth         int      `xml:"MaxWidth,attr"`
	MaxHeight        int      `xml:"MaxHeight,attr"`
	FourCC           string   `xml:"FourCC,attr"`
	CodecPrivateData string   `xml:"CodecPrivateData,attr"`
	AudioTag         int      `xml:"AudioTag,attr"`
	SamplingRate     int      `xml:"SamplingRate,attr"`
	BitsPerSample    int      `xml:"BitsPerSample,attr"`
	PacketSize       int      `xml:"PacketSize,attr"`
	Channels         int      `xml:"Channels,attr"`
}
```


#### type SmoothStreamIndex

```go
type SmoothStreamIndex struct {
	XMLName       xml.Name             `xml:"StreamIndex"`
	Type          string               `xml:"Type,attr"`
	Url           string               `xml:"Url,attr"`
	Name          string               `xml:"Name,attr"`
	Chunks        int                  `xml:"Chunks,attr"`
	QualityLevels int                  `xml:"QualityLevels,attr"`
	MaxWidth      int                  `xml:"MaxWidth,attr"`
	MaxHeight     int                  `xml:"MaxHeight,attr"`
	DisplayWidth  int                  `xml:"DisplayWidth,attr"`
	DisplayHeight int                  `xml:"DisplayHeight,attr"`
	QualityInfos  []SmoothQualityLevel `xml:"QualityLevel"`
	ChunksInfos   []SmoothChunk        `xml:"c"`
}
```


#### type SmoothStreamingMedia

```go
type SmoothStreamingMedia struct {
	XMLName        xml.Name                 `xml:"SmoothStreamingMedia"`
	MajorVersion   int                      `xml:"MajorVersion,attr"`
	MinorVersion   int                      `xml:"MinorVersion,attr"`
	Timescale      int                      `xml:"TimeScale,attr"`
	Duration       int                      `xml:"Duration,attr"`
	IsLive         bool                     `xml:"IsLive,attr"`
	LookaheadCount int                      `xml:"LookaheadCount,attr"`
	StreamIndexes  []SmoothStreamIndex      `xml:"StreamIndex"`
	Protection     []SmoothProtectionHeader `xml:"Protection>ProtectionHeader"`
}
```


#### type SmoothTrackInfo

```go
type SmoothTrackInfo struct {
}
```


#### type SubSampleEncryption

```go
type SubSampleEncryption struct {
}
```

Structure used to store encryption parts of a sample

#### type Track

```go
type Track struct {
}
```

Structure representing a track inside an input file

#### func (*Track) Bandwidth

```go
func (t *Track) Bandwidth() int
```
Return track bandwidth

#### func (*Track) BufferDepth

```go
func (t *Track) BufferDepth() float64
```
Compute track buffer depth

#### func (*Track) BuildAdaptationSet

```go
func (t *Track) BuildAdaptationSet() string
```
Build track adaptation part of the manifest

#### func (*Track) BuildChunk

```go
func (t *Track) BuildChunk(path string) (float64, error)
```
Build a chunk for the track

#### func (*Track) BuildInit

```go
func (t *Track) BuildInit(path string) error
```
Build the init chunk for the track

#### func (*Track) BuildRepresentation

```go
func (t *Track) BuildRepresentation() string
```
Build track representation part of the manifest

#### func (*Track) Clean

```go
func (t *Track) Clean()
```
Clean track private structures for GC

#### func (*Track) CleanDirectory

```go
func (t *Track) CleanDirectory(path string)
```
Clean track directory for unreferenced file in manifest

#### func (*Track) CleanForLive

```go
func (t *Track) CleanForLive()
```
Partially clean internal list in order to generate an up to date manifest

#### func (*Track) ComputePrivateInfos

```go
func (t *Track) ComputePrivateInfos()
```
Extract codec info and compute bandwidth

#### func (*Track) Duration

```go
func (t *Track) Duration() float64
```
Return track duration

#### func (*Track) Height

```go
func (t *Track) Height() int
```
Return track height (0 for audio)

#### func (*Track) InitialiseBuild

```go
func (t *Track) InitialiseBuild(path string) error
```
Initialise build for the track

#### func (*Track) IsAudio

```go
func (t *Track) IsAudio() bool
```
Return if the track is audio or not

#### func (*Track) MaxChunkDuration

```go
func (t *Track) MaxChunkDuration() float64
```
Return largest duration of segments in track

#### func (*Track) MinBufferTime

```go
func (t *Track) MinBufferTime() float64
```
Return largest duration of segments in track

#### func (*Track) Print

```go
func (t *Track) Print()
```
Print track on stdout

#### func (*Track) SetTimeFields

```go
func (t *Track) SetTimeFields()
```
Set creationTime and modificationTime in Track structure

#### func (*Track) Width

```go
func (t *Track) Width() int
```
Return track width (0 for audio)
