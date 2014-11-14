package parser

import (
	"strings"
	"errors"
)

type Demuxer interface {
	Open(path string) error
	GetTracks(tracks *[]*Track) error
	Close()
	ExtractChunk(tracks *[]*Track) bool
}

type DemuxerConstructor func() Demuxer

var demuxerConstructors map[string]DemuxerConstructor

func fileConstructor() Demuxer {
	return new(FFMPEGDemuxer)
}

func dashConstructor() Demuxer {
	return new(DASHDemuxer)
}

/*func smoothConstructor() Demuxer {
	return new(SMOOTHDemuxer)
}*/

/* Initialise specifics for each demuxer interface */
func InitialiseDemuxers() error {
	demuxerConstructors = make(map[string]DemuxerConstructor)
	demuxerConstructors["file"] = fileConstructor
	demuxerConstructors["dash"] = dashConstructor
	//demuxerConstructors["smooth"] = smoothConstructor
	err := FFMPEGInitialise()
	return err
}

func GetAuthorizedProtocols() []string {
	keys := make([]string, 0, len(demuxerConstructors))
	for k := range demuxerConstructors {
		keys = append(keys, k)
	}
	return keys
}

func extractProto(path string) string {
	var i int
	for i = 0; i < len(path); i++ {
		if path[i] == ':' {
			break
		}
	}
	return path[:i]
}

/* Open a file from a path and initialize a demuxer structure */
func OpenDemuxer(path string) (Demuxer, error) {
	var demux Demuxer
	proto := extractProto(path)
	if demuxerConstructors[proto] == nil {
		return nil, errors.New("Unknown protocol for '" + path + "'")
	}
	demux = demuxerConstructors[proto]()
	return demux, demux.Open(strings.Replace(path, proto + "://", "", 1))
}
