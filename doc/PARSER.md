# parser
--

## Usage

#### func  Initialise

```go
func Initialise() error
```
Called when starting the program, initialise FFMPEG demuxers

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

#### type Demuxer

```go
type Demuxer struct {
}
```

Structure used to reference FFMPEG C AVFormatContext structure

#### func  OpenDemuxer

```go
func OpenDemuxer(path string) (*Demuxer, error)
```
Open a file from a path and initialize a demuxer structure

#### func (*Demuxer) AppendSample

```go
func (d *Demuxer) AppendSample(track *Track, stream *C.AVStream)
```
Append a sample to a track

#### func (*Demuxer) Close

```go
func (d *Demuxer) Close()
```
Close demuxer and free FFMPEG specific data

#### func (*Demuxer) ExtractChunk

```go
func (d *Demuxer) ExtractChunk(tracks *[]*Track) bool
```
Extract one chunk for each track from input, size of the chunk depends on the
first

    video track found.

#### func (*Demuxer) GetTracks

```go
func (d *Demuxer) GetTracks(tracks *[]*Track) error
```
Retrieve tracks from previously opened file using FFMPEG

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

#### type Track

```go
type Track struct {
}
```

Structure representing a track inside an input file

#### func (*Track) BuildAdaptationSet

```go
func (t *Track) BuildAdaptationSet() string
```
Build track specific part of the manifest

#### func (*Track) BuildChunk

```go
func (t *Track) BuildChunk(path string) error
```
Build a chunk for the track

#### func (*Track) BuildInit

```go
func (t *Track) BuildInit(path string) error
```
Build the init chunk for the track

#### func (*Track) Clean

```go
func (t *Track) Clean()
```
Clean track private structures for GC

#### func (*Track) Duration

```go
func (t *Track) Duration() float64
```
Return track duration

#### func (*Track) InitialiseBuild

```go
func (t *Track) InitialiseBuild(path string) error
```
Initialise build for the track

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
