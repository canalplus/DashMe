package parser

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec
#include <libavformat/avformat.h>
#include <libavutil/opt.h>
#include <libavcodec/avcodec.h>
#include <string.h>
#include <stdlib.h>

#define TIMEBASE_Q (AVRational){1, 90000}

AVStream* get_stream(AVStream **streams, int pos)
{
  return streams[pos];
}

int64_t rescale_to_generic_timebase(int64_t val, AVRational timebase)
{
  return av_rescale_q(val, timebase, TIMEBASE_Q);
}

int64_t rescale_to_timebase(int64_t val, int in_tb, int out_tb)
{
  AVRational in_r;
  AVRational out_r;

  in_r.num = 1;
  in_r.den = in_tb;

  out_r.num = 1;
  out_r.den = out_tb;

  return av_rescale_q(val, in_r, out_r);
}

char *convert_byte_slice(void *buffer, int size)
{
  char *res = malloc(size);
  memcpy(res, (char *)buffer, size);
  return res;
}
*/
import "C"
import "fmt"
import "errors"
import "unsafe"
import "runtime"

/* Structure used to reference FFMPEG C AVFormatContext structure */
type FFMPEGDemuxer struct {
	context   *C.AVFormatContext
	pkt       C.AVPacket
}

/* Structure used to store a Sample for chunk generation */
type Sample struct {
	pts      int64
	dts      int64
	duration int64
	keyFrame bool
	data	 unsafe.Pointer
	size     C.int
	encrypt  *SampleEncryption
}

/* Called when starting the program, initialise FFMPEG demuxers */
func FFMPEGInitialise() error {
	_, err := C.av_register_all()
	return err
}

func CInt(val int) C.int {
	return C.int(val)
}

func CArray(buffer []byte) unsafe.Pointer {
	if len(buffer) > 0 {
		return unsafe.Pointer(C.convert_byte_slice(unsafe.Pointer(&buffer[0]), C.int(len(buffer))))
	}
	return nil
}

func CFree(ptr unsafe.Pointer) {
	if ptr != nil {
		C.free(ptr)
	}
}

func TimebaseRescale(val int, tbIn int, tbOut int) int {
	return int(C.rescale_to_timebase(C.int64_t(val), C.int(tbIn), C.int(tbOut)))
}

/* Return byte data from a sample */
func (s *Sample) GetData() []byte {
	return C.GoBytes(s.data, s.size)
}

/* Find a track using its index */
func findTrack(tracks []*Track, index int) *Track {
	for i := 0; i < len(tracks); i++ {
		if tracks[i].index == index {
			return tracks[i]
		}
	}
	return nil
}

/* Open FFMPEG specific demuxer */
func (d *FFMPEGDemuxer) Open(path string) error {
	res, err := C.avformat_open_input(&(d.context), C.CString(path), nil, nil)
	if err != nil {
		return err
	} else if res < 0 {
		return errors.New("Could not open source file " + path)
	} else {
		C.av_opt_set_int(unsafe.Pointer(d.context), C.CString("max_analyze_duration"), C.int64_t(0), C.int(0))
		return nil
	}
}

/* Find the first Video track for use as chunk size reference */
func (d *FFMPEGDemuxer) findMainIndex() int {
	var stream *C.AVStream
	for i := 0; i < int(d.context.nb_streams); i++ {
		stream = C.get_stream(d.context.streams, C.int(i))
		if (stream.codec.codec_type == C.AVMEDIA_TYPE_VIDEO) {
			return int(stream.index)
		}
	}
	return 0
}

/* Called by GC to free sample data memory */
func packetFinalizer(s *Sample) {
	C.av_free(unsafe.Pointer(s.data))
}

/* Append a sample to a track */
func (d *FFMPEGDemuxer) AppendSample(track *Track, stream *C.AVStream) {
	sample := new(Sample)
	/* Copy packet metadata in sample */
	sample.pts = int64(C.rescale_to_generic_timebase(d.pkt.pts, stream.time_base))
	sample.dts = int64(C.rescale_to_generic_timebase(d.pkt.dts, stream.time_base))
	sample.duration = int64(C.rescale_to_generic_timebase(C.int64_t(d.pkt.duration), stream.time_base))
	sample.keyFrame = (d.pkt.flags) & 0x1 > 0
	/* Copy packet data in sample */
	sample.size = d.pkt.size
	sample.data = unsafe.Pointer(C.av_malloc(C.size_t(d.pkt.size)))
	C.memcpy(sample.data, unsafe.Pointer(d.pkt.data), C.size_t(d.pkt.size))
	/* Set finalizer to free memory when GC is called */
	runtime.SetFinalizer(sample, packetFinalizer)
	/* Append sample to track */
	track.appendSample(sample)
}

/*
Extract one chunk for each track from input, size of the chunk depends on the first
 video track found.
 */
func (d *FFMPEGDemuxer) ExtractChunk(tracks *[]*Track, isLive bool) bool {
	var track *Track
	var stream *C.AVStream
	/* Find first video track to use as reference for chunk size */
	mainIndex := d.findMainIndex()
	/* Append last extracted chunk */
	d.AppendSample(findTrack(*tracks, int(d.pkt.stream_index)), C.get_stream(d.context.streams, C.int(d.pkt.stream_index)))
	C.av_free_packet(&d.pkt)
	/* Read frames until reference track sample uis a key frame */
	res := C.av_read_frame(d.context, &d.pkt)
	for ; res >= 0; res = C.av_read_frame(d.context, &d.pkt) {
		/* Retrieve track corresponding to packet, if we have one*/
		track = findTrack(*tracks, int(d.pkt.stream_index))
		stream = C.get_stream(d.context.streams, C.int(d.pkt.stream_index))
		if track != nil {
			/* quit if pkt is from reference track and is a key frame*/
			if (track.index == mainIndex && ((d.pkt.flags) & 0x1 > 0) && len(track.samples) > 0) {
				break
			}
			/* Otherwise append sample to chunk */
			d.AppendSample(track, stream)
			C.av_free_packet(&d.pkt)
		}
	}
	/* Return if we have reached EOF */
	return res >= 0
}

/* Retrieve tracks from previously opened file using FFMPEG */
func (d *FFMPEGDemuxer) GetTracks(tracks *[]*Track) error {
	var track *Track
	var stream *C.AVStream
	/* Iterate over streams found by ffmpeg */
	for i := 0; i < int(d.context.nb_streams); i++ {
		/* Little hack to retrieve the stream due to pointer arithmetic */
		stream = C.get_stream(d.context.streams, C.int(i))
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
			// track.colorTableId = int(stream.codec.color_table_id)
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
		} else {
			continue
		}
		/* Set common properties in track structure */
		track.SetTimeFields()
		track.duration = int(C.rescale_to_generic_timebase(stream.duration, stream.time_base))
		track.globalTimescale = 90000
		track.timescale = 90000
		track.extradata = C.GoBytes(unsafe.Pointer(stream.codec.extradata), stream.codec.extradata_size)
		track.index = int(stream.index)
		/* Append track to slice */
		*tracks = append(*tracks, track)
	}
	C.av_read_frame(d.context, &d.pkt)
	return nil
}

/* Close demuxer and free FFMPEG specific data */
func (d *FFMPEGDemuxer) Close() {
	C.avformat_close_input(&d.context);
}
