package main

import (
	"io"
	"fmt"
	//"errors"
	"encoding/binary"
)

type MP4Parser struct {
	count int
	sizes []int

}

func isAtomContainer(name string) bool {
	return (name == "moov" ||
		name == "trak" ||
		name == "mdia" ||
		name == "minf" ||
		name == "stbl")
}

func (p MP4Parser) Initialise() {
	p.count = 0
	p.sizes = nil
}

func (p MP4Parser) Probe(reader io.ReadSeeker, isDir bool) int {
	var b [4]byte
	if isDir { return 0 }
	_, err := reader.Seek(4, 0)
	if err != nil {
		return 0
	}
	err = binary.Read(reader, binary.LittleEndian, &b)
	if err != nil {
		return 0
	}
	if string(b[:]) == "ftyp" {
		return 100
	}
	return 0
}

func (p MP4Parser) seekToAtom(reader io.ReadSeeker, name string, size *int) error {
	var tag string
	var err error
	for tag, err = readAtomHeader(reader, size); err == nil && tag != name; tag, err = readAtomHeader(reader, size) {
		if isAtomContainer(tag) {
			return p.seekToAtom(reader, name, size)
		}
		reader.Seek(int64(*size - 8), 1)
	}
	if err != nil { return err }
	return nil
}

func (p MP4Parser) parseTKHD(reader io.ReadSeeker, size int, track *Track) error {
	var version int
	var err error
	if version, err = atomReadInt8(reader); err != nil { return err }
	reader.Seek(3, 1)
	if version != 0 {
		if track.creationTime, err = atomReadInt64(reader); err != nil {
			return err
		}
		if track.modificationTime, err = atomReadInt64(reader); err != nil {
			return err
		}
		reader.Seek(8, 1)
		if track.duration, err = atomReadInt64(reader); err != nil {
			return err
		}
	} else {
		if track.creationTime, err = atomReadInt32(reader); err != nil {
			return err
		}
		if track.modificationTime, err = atomReadInt32(reader); err != nil {
			return err
		}
		reader.Seek(8, 1)
		if track.duration, err = atomReadInt32(reader); err != nil {
			return err
		}
	}
	reader.Seek(52, 1)
	if track.width, err = atomReadInt32(reader); err != nil {
		return err
	}
	if track.height, err = atomReadInt32(reader); err != nil {
		return err
	}
	track.width = (track.width >> 16) & 0xFFFF
	track.height = (track.height >> 16) & 0xFFFF
	return nil
}

func (p MP4Parser) parseMDHD(reader io.ReadSeeker, size int, track *Track) error {
	var version int
	var err error
	if version, err = atomReadInt8(reader); err != nil { return err }
	reader.Seek(3, 1)
	if version != 0 {
		reader.Seek(16, 1)
		if track.timescale, err = atomReadInt32(reader); err != nil {
			return err
		}
		reader.Seek(8, 1)
	} else {
		reader.Seek(8, 1)
		if track.timescale, err = atomReadInt32(reader); err != nil {
			return err
		}
		reader.Seek(4, 1)
	}
	reader.Seek(4, 1)
	return nil
}

func (p MP4Parser) parseHDLR(reader io.ReadSeeker, size int, track *Track) error {
	var err error
	var tag string
	reader.Seek(8, 1)
	if tag, err = atomReadTag(reader); err != nil {
		return err
	}
	if (tag == "soun") {
		track.isAudio = true
	} else {
		track.isAudio = false
	}
	reader.Seek(int64(size - 20), 1)
	return nil
}

func (p MP4Parser) parseSTSD(reader io.ReadSeeker, size int, track *Track) error {
	var err error
	reader.Seek(8, 1)
	track.extradata, err = atomReadBuffer(reader, size - 16)
	return err
}

func (p MP4Parser) parseSTTS(reader io.ReadSeeker, size int, track *Track) error {
	fmt.Printf("Parsed STTS\n")
	reader.Seek(int64(size - 8), 1)
	return nil
}

func (p MP4Parser) parseSTSZ(reader io.ReadSeeker, size int, track *Track) error {
	fmt.Printf("Parsed STSZ\n")
	reader.Seek(int64(size - 8), 1)
	return nil
}

func (p MP4Parser) parseSTSC(reader io.ReadSeeker, size int, track *Track) error {
	fmt.Printf("Parsed STSC\n")
	reader.Seek(int64(size - 8), 1)
	return nil
}

func (p MP4Parser) parseSTCO(reader io.ReadSeeker, size int, track *Track) error {
	fmt.Printf("Parsed STCO\n")
	reader.Seek(int64(size - 8), 1)
	return nil
}

func (p MP4Parser) parseMOOV(reader io.ReadSeeker, tracks *[]Track) error {
	var track *Track
	var err error
	var size int
	for p.seekToAtom(reader, "trak", &size) == nil {
		fmt.Printf("Adding new Track\n")
		track = new(Track)
		if err = p.seekToAtom(reader, "tkhd", &size); err != nil { return err }
		if err = p.parseTKHD(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "mdhd", &size); err != nil { return err }
		if err = p.parseMDHD(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "hdlr", &size); err != nil { return err }
		if err = p.parseHDLR(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "stsd", &size); err != nil { return err }
		if err = p.parseSTSD(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "stts", &size); err != nil { return err }
		if err = p.parseSTTS(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "stsz", &size); err != nil { return err }
		if err = p.parseSTSZ(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "stsc", &size); err != nil { return err }
		if err = p.parseSTSC(reader, size, track); err != nil { return err }
		if err = p.seekToAtom(reader, "stco", &size); err != nil { return err }
		if err = p.parseSTCO(reader, size, track); err != nil { return err }
		track.Print()
		*tracks = append(*tracks, *track)
	}
	return nil
}

func (p MP4Parser) extractSamples(reader io.ReadSeeker,tracks *[]Track) error {
	//return errors.New("parseMDAT not implemented")
	return nil
}

func (p MP4Parser) Parse(reader io.ReadSeeker, tracks *[]Track, isDir bool) error {
	var size int
	reader.Seek(0, 0)
	err := p.seekToAtom(reader, "moov", &size)
	if err != nil { return err }
	err = p.parseMOOV(reader, tracks)
	if err != nil { return err }
	reader.Seek(0, 0)
	err = p.seekToAtom(reader, "mdat", &size)
	if err != nil { return err }
	err = p.extractSamples(reader,tracks)
	return err
}
