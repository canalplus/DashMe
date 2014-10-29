package main

import (
	"os"
	"io"
	"path/filepath"
	"encoding/binary"
	"strings"
)

func FileExist(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

func RemoveExtension(filename string) string {
	extension := filepath.Ext(filename)
	return filename[0:len(filename)-len(extension)]
}

func parseURL(pattern string, path string, params *map[string]string) bool {
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

func isDirectory(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

func readAtomHeader(reader io.ReadSeeker, res *int) (string, error) {
	var size int32
	var tag [4]byte
	err := binary.Read(reader, binary.BigEndian, &size)
	if err != nil { return "", err }
	err = binary.Read(reader, binary.LittleEndian, &tag)
	if err != nil { return "", err }
	*res = int(size)
	return string(tag[:]), nil
}


func currentOffset(reader io.ReadSeeker) (int, error) {
	offset, err := reader.Seek(0, 1)
	return int(offset), err
}

func atomReadInt8(reader io.Reader) (int, error) {
	var val uint8
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

func atomReadInt16(reader io.Reader) (int, error) {
	var val uint16
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

func atomReadInt24(reader io.Reader) (int, error) {
	var val [3]byte
	err := binary.Read(reader, binary.BigEndian, &val)
	return int((val[0] << 16) | (val[1] << 8) | val[0]), err
}

func atomReadInt32(reader io.Reader) (int, error) {
	var val uint32
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

func atomReadInt64(reader io.Reader) (int, error) {
	var val uint64
	err := binary.Read(reader, binary.BigEndian, &val)
	return int(val), err
}

func atomReadTag(reader io.Reader) (string, error) {
	var val [4]byte
	err := binary.Read(reader, binary.LittleEndian, &val)
	return string(val[:]), err
}

func atomReadBuffer(reader io.Reader, size int) ([]byte, error) {
	val := make([]byte, size)
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}
