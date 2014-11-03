package parser

//#include <libavformat/avformat.h>
//#include <libavutil/opt.h>
//
//AVStream* get_stream(AVStream **streams, int pos)
//{
//  return streams[pos];
//}
import "C"
import "errors"
import "unsafe"
import "fmt"

type Demuxer struct {
	context *C.AVFormatContext
}

func Initialise() error {
	_, err := C.av_register_all()
	return err
}

func OpenDemuxer(path string) (*Demuxer, error) {
	demux := new(Demuxer)
	res, err := C.avformat_open_input(&(demux.context), C.CString(path), nil, nil)
	if err != nil {
		return nil, err
	} else if res < 0 {
		return nil, errors.New("Could not open source file " + path)
	} else {
		C.av_opt_set_int(unsafe.Pointer(demux.context), C.CString("max_analyze_duration"), C.int64_t(0), C.int(0))
		return demux, nil
	}
}

func findTrack(tracks []Track, index int) *Track {
	for i := 0; i < len(tracks); i++ {
		if tracks[i].index == index {
			return &tracks[i]
		}
	}
	return nil
}

func (d *Demuxer) GetTracks(tracks *[]Track) error {
	var track *Track
	var stream *C.AVStream
	var pkt C.AVPacket
	var sample Sample
	res, err := C.avformat_find_stream_info(d.context, nil)
	if err != nil {
		return err
	} else if res < 0 {
		return errors.New("Could not find stream information")
	}
	for i := 0; i < int(d.context.nb_streams); i++ {
		stream = C.get_stream(d.context.streams, C.int(i))
		track = nil
		if stream.codec.codec_type == C.AVMEDIA_TYPE_VIDEO {
			fmt.Printf("New video Track !\n")
			track = new(Track)
			track.width = int(stream.codec.width)
			track.height = int(stream.codec.height)
			track.bitsPerSample = int(stream.codec.bits_per_coded_sample)
			track.colorTableId = int(stream.codec.color_table_id)
			track.isAudio = false
		} else if stream.codec.codec_type == C.AVMEDIA_TYPE_AUDIO {
			fmt.Printf("New audio Track !\n")
			track = new(Track)
			track.sampleRate = int(stream.codec.sample_rate)
			track.isAudio = true
		}
		if track != nil {
			track.duration = int(stream.duration)
			track.timescale = int(stream.time_base.den)
			track.extradata = C.GoBytes(unsafe.Pointer(stream.codec.extradata), stream.codec.extradata_size)
			track.index = int(stream.index)
			*tracks = append(*tracks, *track)
		}
	}
	for C.av_read_frame(d.context, &pkt) >= 0 {
		track = findTrack(*tracks, int(pkt.stream_index))
		if track != nil {
			sample = Sample{}
			sample.pts = int(pkt.pts)
			sample.dts = int(pkt.dts)
			sample.duration = int(pkt.duration)
			if (pkt.flags) & 0x1 > 0 {
				sample.keyFrame = true
			} else {
				sample.keyFrame = false
			}
			sample.dataPtr = unsafe.Pointer(pkt.data)
			sample.data = C.GoBytes(sample.dataPtr, pkt.size)
			track.samples = append(track.samples, sample)
		}
	}
	return nil
}

func (d *Demuxer) CleanTracks(tracks []Track) {
	for _, track := range tracks {
		for _, sample := range track.samples {
			C.av_free(sample.dataPtr)
		}
	}
}
