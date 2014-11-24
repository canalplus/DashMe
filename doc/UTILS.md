# utils
--
    import "."

Package inotify implements a wrapper for the Linux inotify system.

Example:

    watcher, err := inotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }
    err = watcher.Watch("/tmp")
    if err != nil {
        log.Fatal(err)
    }
    for {
        select {
        case ev := <-watcher.Event:
            log.Println("event:", ev)
        case err := <-watcher.Error:
            log.Println("error:", err)
        }
    }

## Usage

```go
const (

	// Options for AddWatch
	IN_DONT_FOLLOW uint32 = syscall.IN_DONT_FOLLOW
	IN_ONESHOT     uint32 = syscall.IN_ONESHOT
	IN_ONLYDIR     uint32 = syscall.IN_ONLYDIR

	// Events
	IN_ACCESS        uint32 = syscall.IN_ACCESS
	IN_ALL_EVENTS    uint32 = syscall.IN_ALL_EVENTS
	IN_ATTRIB        uint32 = syscall.IN_ATTRIB
	IN_CLOSE         uint32 = syscall.IN_CLOSE
	IN_CLOSE_NOWRITE uint32 = syscall.IN_CLOSE_NOWRITE
	IN_CLOSE_WRITE   uint32 = syscall.IN_CLOSE_WRITE
	IN_CREATE        uint32 = syscall.IN_CREATE
	IN_DELETE        uint32 = syscall.IN_DELETE
	IN_DELETE_SELF   uint32 = syscall.IN_DELETE_SELF
	IN_MODIFY        uint32 = syscall.IN_MODIFY
	IN_MOVE          uint32 = syscall.IN_MOVE
	IN_MOVED_FROM    uint32 = syscall.IN_MOVED_FROM
	IN_MOVED_TO      uint32 = syscall.IN_MOVED_TO
	IN_MOVE_SELF     uint32 = syscall.IN_MOVE_SELF
	IN_OPEN          uint32 = syscall.IN_OPEN

	// Special events
	IN_ISDIR      uint32 = syscall.IN_ISDIR
	IN_IGNORED    uint32 = syscall.IN_IGNORED
	IN_Q_OVERFLOW uint32 = syscall.IN_Q_OVERFLOW
	IN_UNMOUNT    uint32 = syscall.IN_UNMOUNT
)
```

#### func  AtomReadBuffer

```go
func AtomReadBuffer(reader io.Reader, size int) ([]byte, error)
```
Return a buffer of arbitrary size

#### func  AtomReadInt16

```go
func AtomReadInt16(reader io.Reader) (int, error)
```
Return an int read from 2 bytes

#### func  AtomReadInt24

```go
func AtomReadInt24(reader io.Reader) (int, error)
```
Return an int read from 3 bytes

#### func  AtomReadInt32

```go
func AtomReadInt32(reader io.Reader) (int, error)
```
Return an int read from 4 bytes

#### func  AtomReadInt64

```go
func AtomReadInt64(reader io.Reader) (int64, error)
```
Return an int read from 8 bytes

#### func  AtomReadInt8

```go
func AtomReadInt8(reader io.Reader) (int, error)
```
Return an int read from one byte

#### func  AtomReadTag

```go
func AtomReadTag(reader io.Reader) (string, error)
```
Return a string read from a 4 byte int

#### func  BuildAtom

```go
func BuildAtom(tag string, content []byte) ([]byte, error)
```
Build atom from a tag and its content

#### func  BuildEmptyAtom

```go
func BuildEmptyAtom(tag string, size int) ([]byte, error)
```
Build atom of a specific size filled with 0

#### func  CurrentOffset

```go
func CurrentOffset(reader io.ReadSeeker) (int, error)
```
Return current offset of a io.ReadSeeker

#### func  DisplayMemStats

```go
func DisplayMemStats()
```

#### func  FileExist

```go
func FileExist(filename string) bool
```
Test if a file or directory exist

#### func  IsDirectory

```go
func IsDirectory(path string) bool
```
Test if a path exist and is a directory

#### func  ParseURL

```go
func ParseURL(pattern string, path string, params *map[string]string) bool
```
parse an URL and extract information according to a pattern

#### func  ReadAtomHeader

```go
func ReadAtomHeader(reader io.ReadSeeker, res *int) (string, error)
```
Read atom header : tag and size

#### func  RemoveExtension

```go
func RemoveExtension(filename string) string
```
Remove extension from a filename

#### type Event

```go
type Event struct {
	Mask   uint32 // Mask of events
	Cookie uint32 // Unique cookie associating related events (for rename(2))
	Name   string // File name (optional)
}
```


#### func (*Event) String

```go
func (e *Event) String() string
```
String formats the event e in the form "filename: 0xEventMask =
IN_ACCESS|IN_ATTRIB_|..."

#### type Queue

```go
type Queue struct {
}
```

Data structure used to represent a queue

#### func (*Queue) Clear

```go
func (s *Queue) Clear()
```
Clear queue

#### func (*Queue) Empty

```go
func (s *Queue) Empty() bool
```
Return if the queue is empty

#### func (*Queue) Pop

```go
func (s *Queue) Pop() interface{}
```
Pop the first element from the queue

#### func (*Queue) Push

```go
func (s *Queue) Push(elms ...interface{})
```
Push an element to the end of the queue

#### func (*Queue) Size

```go
func (s *Queue) Size() int
```
Return queue size

#### type Watcher

```go
type Watcher struct {
	Error chan error  // Errors are sent on this channel
	Event chan *Event // Events are returned on this channel
}
```


#### func  NewWatcher

```go
func NewWatcher() (*Watcher, error)
```
NewWatcher creates and returns a new inotify instance using inotify_init(2)

#### func (*Watcher) AddWatch

```go
func (w *Watcher) AddWatch(path string, flags uint32) error
```
AddWatch adds path to the watched file set. The flags are interpreted as
described in inotify_add_watch(2).

#### func (*Watcher) Close

```go
func (w *Watcher) Close() error
```
Close closes an inotify watcher instance It sends a message to the reader
goroutine to quit and removes all watches associated with the inotify instance

#### func (*Watcher) RemoveWatch

```go
func (w *Watcher) RemoveWatch(path string) error
```
RemoveWatch removes path from the watched file set.

#### func (*Watcher) Watch

```go
func (w *Watcher) Watch(path string) error
```
Watch adds path to the watched file set, watching all events.
