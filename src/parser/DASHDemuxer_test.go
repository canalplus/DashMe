package parser

import "testing"

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
