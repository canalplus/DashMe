package parser

/*
#include <libavformat/avformat.h>
#include <libavutil/opt.h>

#define TIMEBASE_Q (AVRational){1, 90000}

AVStream* get_stream(AVStream **streams, int pos)
{
  return streams[pos];
}

int64_t rescale_to_timebase(int64_t val, AVRational timebase)
{
  return av_rescale_q(val, timebase, TIMEBASE_Q);
}
*/
import "C"
import "fmt"
import "errors"
import "unsafe"

/* Structure used to reference FFMPEG C AVFormatContext structure */
type Demuxer struct {
	context *C.AVFormatContext
}

/* Called when starting the program, initialise FFMPEG demuxers */
func Initialise() error {
	_, err := C.av_register_all()
	return err
}

/* Open a file from a path and initialize a demuxer structure */
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

/* Retrieve tracks from previously opened file using FFMPEG */
func (d *Demuxer) GetTracks(tracks *[]Track) error {
	var track *Track
	var stream *C.AVStream
	var pkt C.AVPacket
	var sample Sample
	/* Use FFMPEG to extract stream info from file */
	res, err := C.avformat_find_stream_info(d.context, nil)
	if err != nil {
		return err
	} else if res < 0 {
		return errors.New("Could not find stream information")
	}
	/* Iterate over streams found */
	for i := 0; i < int(d.context.nb_streams); i++ {
		/* Little hack to retrieve the stream due to pointer arithmetic */
		stream = C.get_stream(d.context.streams, C.int(i))
		track = nil
		if stream.codec.codec_type == C.AVMEDIA_TYPE_VIDEO {
			/* Test if video is H264 */
			if stream.codec.codec_id != C.AV_CODEC_ID_H264 {
				return fmt.Errorf("Video track is not encoded in H264 (codec_id=%d)", stream.codec.codec_id)
			}
			/* Set video specific info in track structure */
			track = new(Track)
			track.width = int(stream.codec.width)
			track.height = int(stream.codec.height)
			track.bitsPerSample = int(stream.codec.bits_per_coded_sample)
			track.colorTableId = int(stream.codec.color_table_id)
			track.isAudio = false
		} else if stream.codec.codec_type == C.AVMEDIA_TYPE_AUDIO {
			/* Test if audio is AAC */
			if stream.codec.codec_id != C.AV_CODEC_ID_AAC  {
				return fmt.Errorf("Audio track is not encoded in AAC (codec_id=%d)", stream.codec.codec_id)
			}
			/* Set audio specific info in track structure */
			track = new(Track)
			track.sampleRate = int(stream.codec.sample_rate)
			track.isAudio = true
		}
		if track != nil {
			/* Set common properties in track structure */
			track.SetTimeFields()
			track.duration = int(C.rescale_to_timebase(stream.duration, stream.time_base))
			track.timescale = 90000
			track.extradata = C.GoBytes(unsafe.Pointer(stream.codec.extradata), stream.codec.extradata_size)
			track.index = int(stream.index)
			/* Append track to slice */
			*tracks = append(*tracks, *track)
		}
	}
	/* Now that we have all interesting tracks we can extract samples */
	for C.av_read_frame(d.context, &pkt) >= 0 {
		/* Retrieve track corresponding to packet, if we have one*/
		track = findTrack(*tracks, int(pkt.stream_index))
		stream = C.get_stream(d.context.streams, C.int(pkt.stream_index))
		if track != nil && pkt.pts != C.AV_NOPTS_VALUE {
			/* Sample is from an interesting track, so set info from packet */
			sample = Sample{}
			sample.pts = int(C.rescale_to_timebase(pkt.pts, stream.time_base))
			sample.dts = int(C.rescale_to_timebase(pkt.dts, stream.time_base))
			sample.duration = int(C.rescale_to_timebase(C.int64_t(pkt.duration), stream.time_base))
			sample.keyFrame = (pkt.flags) & 0x1 > 0
			sample.dataPtr = unsafe.Pointer(pkt.data)
			sample.data = C.GoBytes(sample.dataPtr, pkt.size)
			/* Append sample to track samples */
			track.appendSample(sample)
		}
	}
	return nil
}

/* Free all C allocated data inside tracks */
func (d *Demuxer) CleanTracks(tracks []Track) {
	/* Iterate over all samples in all tracks to free their data allocated in C */
	for i := 0; i < len(tracks); i++ {
		tracks[i].Print()
		for j := 0; j < len(tracks[i].samples); j++ {
			C.av_free(tracks[i].samples[j].dataPtr)
		}
	}
}
