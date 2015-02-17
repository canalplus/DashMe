package parser

import "testing"

const segmentBaseManifest string = "dash-vod-aka-test.canal-bis.com/test/1fps/index.mpd"
const segmentTemplateManifest string = "www.digitalprimates.net/dash/streams/mp4-live-template/mp4-live-mpd-AV-BS.mpd"

var demuxer DASHDemuxer

func TestOpen(t *testing.T) {
  demuxer := new(DASHDemuxer)
  demuxer.Open("path/to/base/manifest.mpd")

  urlCases := []struct {
    got, want string
  }{
    {demuxer.manifestURL, "http://path/to/base/manifest.mpd"},
    {demuxer.baseURL,     "http://path/to/base"},
  }

  for _, c :=  range urlCases {
    if c.want != c.got {
      t.Errorf("bad manfest url. want %q, got %q", c.want, c.got)
    }
  }
}

func TestSegemenBaseGetTracks(t *testing.T) {
  demuxer := new(DASHDemuxer)
  demuxer.Open(segmentBaseManifest)
  var tracks []*Track

  err := demuxer.GetTracks(&tracks)

  if err != nil {
    t.Errorf("got error in GetTracks %q", err)
  }
}

func TestSegemenTemplateGetTracks(t *testing.T) {
  demuxer := new(DASHDemuxer)
  demuxer.Open(segmentTemplateManifest)
  var tracks []*Track

  err := demuxer.GetTracks(&tracks)

  if err != nil {
    t.Errorf("got error in GetTracks %q", err)
  }
}