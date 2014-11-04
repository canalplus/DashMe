package utils

import (
	"os"
	"io"
	"path/filepath"
	"encoding/binary"
	"strings"
	"fmt"
	"bytes"
)

/* Test if a file or directory exist */
func FileExist(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

/* Remove extension from a filename */
func RemoveExtension(filename string) string {
	extension := filepath.Ext(filename)
	return filename[0:len(filename)-len(extension)]
}

/* parse an URL and extract information according to a pattern */
func ParseURL(pattern string, path string, params *map[string]string) bool {
	patternSplit := strings.Split(pattern, "/")
	pathSplit := strings.Split(path, "/")
	if len(patternSplit) != len(pathSplit) {
		return false
	}
	for i := 0; i < len(patternSplit); i++ {
		if len(patternSplit[i]) != 0 && patternSplit[i][0] == ':' && pathSplit[i] != "" {
			(*params)[strings.Trim(patternSplit[i], ":")] = pathSplit[i]
		} else if patternSplit[i] != pathSplit[i] {
			return false
		}
	}
	return true
}

/* Test if a path exist and is a directory */
func IsDirectory(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

/* Read atom header : tag and size*/
func ReadAtomHeader(reader io.ReadSeeker, res *int) (string, error) {
	var size int32
	var tag [4]byte
	err := binary.Read(reader, binary.BigEndian, &size)
	if err != nil { return "", err }
	err = binary.Read(reader, binary.LittleEndian, &tag)
	fmt.Printf("Tag read : %q\n", tag)
	if err != nil { return "", err }
	*res = int(size)
	return string(tag[:]), nil
}

/* Return current offset of a io.ReadSeeker */
func CurrentOffset(reader io.ReadSeeker) (int, error) {
	offset, err := reader.Seek(0, 1)
	return int(offset), err
}

/* Return an int read from one byte */
func AtomReadInt8(reader io.Reader) (int, error) {
	var val uint8
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

/* Return an int read from 2 bytes */
func AtomReadInt16(reader io.Reader) (int, error) {
	var val uint16
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

/* Return an int read from 3 bytes */
func AtomReadInt24(reader io.Reader) (int, error) {
	var val [3]byte
	err := binary.Read(reader, binary.BigEndian, &val)
	return int((val[0] << 16) | (val[1] << 8) | val[0]), err
}

/* Return an int read from 4 bytes */
func AtomReadInt32(reader io.Reader) (int, error) {
	var val uint32
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

/* Return an int read from 8 bytes */
func AtomReadInt64(reader io.Reader) (int, error) {
	var val uint64
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

/* Return a string read from a 4 byte int */
func AtomReadTag(reader io.Reader) (string, error) {
	var val [4]byte
	err := binary.Read(reader, binary.LittleEndian, &val)
	return string(val[:]), err
}

/* Return a buffer of arbitrary size */
func AtomReadBuffer(reader io.Reader, size int) ([]byte, error) {
	val := make([]byte, size)
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

/* Build atom from a tag and its content */
func BuildAtom(tag string, content []byte) ([]byte, error) {
	var b bytes.Buffer
	err := binary.Write(&b, binary.BigEndian, int32(len(content) + 8))
	if err != nil { return nil, err }
	err = binary.Write(&b, binary.LittleEndian, []byte(tag))
	if err != nil { return nil, err }
	err = binary.Write(&b, binary.BigEndian, content)
	if err != nil { return nil, err }
	return b.Bytes(), nil
}

/* Build atom of a specific size filled with 0 */
func BuildEmptyAtom(tag string, size int) ([]byte, error) {
	return BuildAtom(tag, make([]byte, size))
}
