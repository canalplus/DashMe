# utils
--

## Usage

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
func AtomReadInt64(reader io.Reader) (int, error)
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
