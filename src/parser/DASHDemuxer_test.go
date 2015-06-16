// Copyright 2015 CANAL+ Group
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
