package parser

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -ljpeg
#include <libavformat/avformat.h>
#include <libavutil/opt.h>
#include <libavcodec/avcodec.h>
#include <string.h>
#include <stdlib.h>
#include <jpeglib.h>

#define TIMEBASE_Q (AVRational){1, 90000}

#if LIBAVCODEC_VERSION_INT >= AV_VERSION_INT(55,28,1)
# include <libavutil/frame.h>
AVFrame *alloc_frame(void)
{
  return av_frame_alloc();
}

void free_frame(AVFrame *f)
{
  av_frame_unref(f);
  av_frame_free(&f);
}
#else
AVFrame *alloc_frame(void)
{
  return avcodec_alloc_frame();
}

void free_frame(AVFrame *f)
{
  avcodec_free_frame(&f);
}
#endif

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

void set_pkt_data(void *data, AVPacket *pkt)
{
  pkt->data = (uint8_t*)data;
}

void set_extradata(void *data, size_t size, AVCodecContext *ctx)
{
  ctx->extradata = (uint8_t*)data;
  ctx->extradata_size = size;
}

*/
import "C"
import "fmt"
import "errors"
import "unsafe"
import "runtime"
import "image"
import "image/jpeg"
import "bytes"
import "utils"

const (
	THUMBNAIL_WIDTH = 320
	THUMBNAIL_HEIGHT = 180
)

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

/* Create a JPEG image from using samples from a chunk (only for clear streams) */
func FFMPEGGetImageFromSamples(samples []*Sample, extradata []byte) ([]byte, error) {
	var gotPicture C.int
	var pkt C.AVPacket
	gotPicture = 0
	pos := 0
	/* Retrieve FFMPEG H264 codec */
	codec := C.avcodec_find_decoder(C.AV_CODEC_ID_H264)
	if codec == nil {
		return nil, errors.New("Codec not found !")
	}
	/* Allocate context for codec */
	ctx := C.avcodec_alloc_context3(codec)
	if ctx == nil {
		return nil, errors.New("Could not allocate codec context !")
	}
	defer C.av_free(unsafe.Pointer(ctx))
	/* Set track extradata for decryption */
	C.set_extradata(unsafe.Pointer(&extradata[0]), C.size_t(len(extradata)), ctx);
	/* Open codec */
	if C.avcodec_open2(ctx, codec, nil) < 0 {
		return nil, errors.New("Could not open codec !")
	}
	defer C.avcodec_close(ctx)
	/* Allocate ffmpeg frame structure */
	frame := C.alloc_frame()
	if frame == nil {
		return nil, errors.New("Could not allocate frame !")
	}
	defer C.free_frame(frame)
	if codec.capabilities & C.CODEC_CAP_TRUNCATED > 0 {
		ctx.flags |= C.CODEC_FLAG_TRUNCATED;
	}
	/* Decode until we have a complete frame */
	for gotPicture == 0 && pos < len(samples) {
		C.av_init_packet(&pkt)
		C.set_pkt_data(samples[pos].data, &pkt)
		pkt.size = samples[pos].size
		if samples[pos].keyFrame {
			pkt.flags |= C.AV_PKT_FLAG_KEY
		}
		C.avcodec_decode_video2(ctx, frame, &gotPicture, &pkt);
		pos++
	}
	/* No frame decoded for the chunk */
	if gotPicture == 0 {
		return nil, errors.New("Unable to decode picture !")
	}
	/* Create go YCbCr image */
	img := image.NewYCbCr(image.Rect(0, 0, int(frame.width), int(frame.height)), image.YCbCrSubsampleRatio420)
	img.Y = []uint8(C.GoBytes(unsafe.Pointer(frame.data[0]), frame.linesize[0] * frame.height))
	img.Cb = []uint8(C.GoBytes(unsafe.Pointer(frame.data[1]), frame.linesize[1] * (frame.height / 2)))
	img.Cr = []uint8(C.GoBytes(unsafe.Pointer(frame.data[2]), frame.linesize[2] * (frame.height / 2)))
	img.YStride = int(frame.linesize[0])
	img.CStride = int(frame.linesize[1])
	b := new(bytes.Buffer)
	/* Resize image if necessary */
	if (frame.width > THUMBNAIL_WIDTH || frame.height > THUMBNAIL_HEIGHT) {
		jpeg.Encode(b, utils.Resize(img, img.Rect, THUMBNAIL_WIDTH, THUMBNAIL_HEIGHT), nil)
	} else {
		jpeg.Encode(b, img, nil)
	}
	/* Convert YCbCr image to JPEG */
	return b.Bytes(), nil
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
	/* Read frames until reference track sample is a key frame */
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
