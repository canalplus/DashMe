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

/* Initialise specifics for each demuxer interface */
func InitialiseDemuxers() error {
	err := FFMPEGInitialise()
	return err
}

/* Open a file from a path and initialize a demuxer structure */
func OpenDemuxer(path string) (Demuxer, error) {
	var demux Demuxer
	if (strings.HasPrefix(path, "http://")) {
		return nil, errors.New("HTTP streams (Smooth Streaming, Dash) are not supported yet !")
	} else if strings.HasPrefix(path, "file://") {
		demux = new(FFMPEGDemuxer)
		return demux, demux.Open(strings.Replace(path, "file://", "", 1))
	} else {
		return nil, errors.New("Unknown protocol for '" + path + "'")
	}
}
