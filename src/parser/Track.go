package parser

import (
	"os"
	"fmt"
	"time"
	"utils"
	"errors"
	"strings"
	"strconv"
	"path/filepath"
	"encoding/hex"
)

/* Function type used for atom generation */
type AtomBuilder func(t Track) ([]byte, error)

/* Structure used to build chunks */
type Builder struct {
	builders map[string]AtomBuilder
}

/* Structure representing a track inside an input file */
type Track struct {
	index            int
	isAudio	         bool
	creationTime     int
	duration         int
	modificationTime int
	timescale        int
	width            int
	height           int
	sampleRate       int
	bitsPerSample    int
	colorTableId     int
	bandwidth        int
	codec            string
	currentDuration  int
	extradata        []byte
	samples          []*Sample
	chunksDuration   []int
	builder          Builder
}

/* Print track on stdout */
func (t *Track) Print() {
	fmt.Println("Track :")
	fmt.Println("\tindex : ", t.index)
	fmt.Println("\tisAudio : ", t.isAudio)
	fmt.Println("\tcreationTime : ", t.creationTime)
	fmt.Println("\tmodificationTime : ", t.modificationTime)
	fmt.Println("\tduration : ", t.duration)
	fmt.Println("\ttimescale : ", t.timescale)
	fmt.Println("\twidth : ", t.width)
	fmt.Println("\theight : ", t.height)
	fmt.Println("\tsampleRate : ", t.sampleRate)
	fmt.Println("\tbitsPerSample : ", t.bitsPerSample)
	fmt.Println("\tcolorTableId : ", t.colorTableId)
	fmt.Println("\tbandwidth: ", t.bandwidth)
	fmt.Println("\tcodec: ", t.codec)
	fmt.Println("\tsamples count : ", len(t.samples))
}

/*
   Atom specific building functions.
   Refere to ISO 14496-12 2012E for more details on each atom
*/

func buildSTTS(t Track) ([]byte, error) {
	return utils.BuildEmptyAtom("stts", 16)
}

func buildSTSC(t Track) ([]byte, error) {
	return utils.BuildEmptyAtom("stsc", 16)
}

func buildSTCO(t Track) ([]byte, error) {
	return utils.BuildEmptyAtom("stco", 16)
}

func buildSTSZ(t Track) ([]byte, error) {
	return utils.BuildEmptyAtom("stsz", 20)
}

func buildSTSS(t Track) ([]byte, error) {
	return utils.BuildEmptyAtom("stss", 16)
}

func buildSTBL(t Track) ([]byte, error) {
	b, err := t.buildAtoms("stsd", "stts", "stsc", "stco", "stsz", "stss")
	if err != nil { return nil, err }
	return utils.BuildAtom("stbl", b)
}

func buildMINF(t Track) ([]byte, error) {
	var b []byte
	var err error
	if t.isAudio {
		b, err = t.buildAtoms("dinf", "stbl", "smhd")
	} else {
		b, err = t.buildAtoms("dinf", "stbl", "vmhd")
	}
	if err != nil { return nil, err }
	return utils.BuildAtom("minf", b)
}

func buildMDIA(t Track) ([]byte, error) {
	b, err := t.buildAtoms("mdhd", "hdlr", "minf")
	if err != nil { return nil, err }
	return utils.BuildAtom("mdia", b)
}

func buildTRAK(t Track) ([]byte, error) {
	b, err := t.buildAtoms("tkhd", "mdia")
	if err != nil { return nil, err }
	return utils.BuildAtom("trak", b)
}

func buildMVEX(t Track) ([]byte, error) {
	b, err := t.buildAtoms("trex")
	if err != nil { return nil, err }
	return utils.BuildAtom("mvex", b)
}

func buildMOOV(t Track) ([]byte, error) {
	b, err := t.buildAtoms("mvhd", "mvex", "trak")
	if err != nil { return nil, err }
	return utils.BuildAtom("moov", b)
}

func buildDINF(t Track) ([]byte, error) {
	b, err := t.buildAtoms("dref")
	if err != nil {
		return nil, err
	}
	return utils.BuildAtom("dinf", b)
}

func buildMOOF(t Track) ([]byte, error) {
	b, err := t.buildAtoms("mfhd", "traf")
	if err != nil {
		return nil, err
	}
	return utils.BuildAtom("moof", b)
}

func buildTRAF(t Track) ([]byte, error) {
	b, err := t.buildAtoms("tfhd", "tfdt", "trun")
	if err != nil {
		return nil, err
	}
	return utils.BuildAtom("traf", b)
}

func buildAVCC(t Track) ([]byte, error) {
	return utils.BuildAtom("avcC", t.extradata)
}

func buildAVC1(t Track) ([]byte, error) {
	b, err := t.buildAtoms("avcC")
	if err != nil { return nil, err }
	return utils.BuildAtom("avc1", append([]byte{
		/* Reserved */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* index */
		0x0, 0x1,
		/* version */
		0x0, 0x0,
		/* Revision level */
		0x0, 0x0,
		/* Vendor */
		0x0, 0x0, 0x0, 0x0,
		/* Temporal quality */
		0x0, 0x0, 0x0, 0x0,
		/* Spatial quality */
		0x0, 0x0, 0x0, 0x0,
		/* width */
		byte((t.width >> 8) & 0xFF), byte((t.width) & 0xFF),
 		/* Height */
		byte((t.height >> 8) & 0xFF), byte((t.height) & 0xFF),
		/* Horizontal resolution */
		0x0, 0x48, 0x0, 0x0,
		/* Vertical resolution */
		0x0, 0x48, 0x0, 0x0,
		/* Data size */
		0x0, 0x0, 0x0, 0x0,
		/* Frames per sample */
		0x0, 0x1,
		/* Compressor Name */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Depth */
		byte((t.bitsPerSample >> 8) & 0xFF), byte((t.bitsPerSample) & 0xFF),
		/* Color table id */
		byte((t.colorTableId >> 8) & 0xFF), byte((t.colorTableId) & 0xFF),
	}, b...))
}

func buildESDS(t Track) ([]byte, error) {
	size := len(t.extradata)
	data := append(t.extradata, []byte{
		0x6, 0x80, 0x80, 0x80, 0x1, 0x2,
	}...)
	data = append([]byte{
		0x0, 0x0, 0x0, 0x0,
		0x3, 0x80, 0x80, 0x80, 0x22, 0x0, 0x0, 0x0,
		0x4, 0x80, 0x80, 0x80, 0x14, 0x40, 0x15, 0x0, 0x1, 0x18, 0x0,
		0x1, 0x64, 0xf0, 0x0, 0x1, 0x44, 0x6b, 0x05, 0x80, 0x80, 0x80,
		byte((size) & 0xFF),
	}, data...)
	return utils.BuildAtom("esds",  data)
}

func buildMP4A(t Track) ([]byte, error) {
	b, err := t.buildAtoms("esds")
	if err != nil { return nil, err }
	return utils.BuildAtom("mp4a", append([]byte{
		/* Reserved */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* index */
		0x0, 0x1,
		/* Version */
		0x0, 0x0,
		/* Revision level */
		0x0, 0x0,
		/* Vendor */
		0x0, 0x0, 0x0, 0x0,
		/* channels */
		0x0, 0x2,
		/* Sample size */
		0x0, 0x10,
		/* Compression ID */
		0x0, 0x0,
		/* Packet size */
		0x0, 0x0,
		/* Sample rate */
		byte((t.sampleRate >> 8) & 0xFF),
		byte((t.sampleRate) & 0xFF),
		0x0, 0x0,
	}, b...))
}

func buildSTSD(t Track) ([]byte, error) {
	var b []byte
	var err error
	if t.isAudio {
		b, err = t.buildAtoms("mp4a")
	} else {
		b, err = t.buildAtoms("avc1")
	}
	if err != nil { return nil, err }
	return utils.BuildAtom("stsd", append([]byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Entry count */
		0x0, 0x0, 0x0, 0x1,
	}, b...))
}

func buildFTYP(t Track) ([]byte, error) {
	return utils.BuildAtom("ftyp", []byte{
		/* Major brand */
		0x64, 0x61, 0x73, 0x68,
		/* Minor version */
		0x0, 0x0, 0x0, 0x0,
		/* Compatibility */
		0x69, 0x73, 0x6f, 0x36, 0x61, 0x76, 0x63, 0x31, 0x6d, 0x70,
		0x34, 0x31,
	})
}

func buildMVHD(t Track) ([]byte, error) {
	return utils.BuildAtom("mvhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Creation time */
		byte((t.creationTime >> 24) & 0xFF),
		byte((t.creationTime >> 16) & 0xFF),
		byte((t.creationTime >> 8) & 0xFF),
		byte((t.creationTime) & 0xFF),
		/* Modification time */
		byte((t.modificationTime >> 24) & 0xFF),
		byte((t.modificationTime >> 16) & 0xFF),
		byte((t.modificationTime >> 8) & 0xFF),
		byte((t.modificationTime) & 0xFF),
		/* Timescale */
		byte((t.timescale >> 24) & 0xFF),
		byte((t.timescale >> 16) & 0xFF),
		byte((t.timescale >> 8) & 0xFF),
		byte((t.timescale) & 0xFF),
		/* Duration */
		byte((t.duration >> 24) & 0xFF),
		byte((t.duration >> 16) & 0xFF),
		byte((t.duration >> 8) & 0xFF),
		byte((t.duration) & 0xFF),
		/* Rate */
		0x0, 0x1, 0x0, 0x0,
		/* Volume */
		0x1, 0x0,
		/* Reserved */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Matrix */
		0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
		/* Predefined */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Predefined */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Predefined */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Next track ID */
		0x0, 0x0, 0x0, 0x2,
	})
}

func buildTREX(t Track) ([]byte, error) {
	return utils.BuildAtom("trex", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Track ID */
		0x0, 0x0, 0x0, 0x1,
		/* Default sample description index */
		0x0, 0x0, 0x0, 0x1,
		/* Default sample duration */
		0x0, 0x0, 0x0, 0x0,
		/* Default sample size */
		0x0, 0x0, 0x0, 0x0,
		/* Default sample flags */
		0x0, 0x0, 0x0, 0x0,
	})
}

func buildTKHD(t Track) ([]byte, error) {
	var volume uint8
	if t.isAudio {
		volume = 1
	} else {
		volume = 0
	}
	return utils.BuildAtom("tkhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x3,
		/* Creation time */
		byte((t.creationTime >> 24) & 0xFF),
		byte((t.creationTime >> 16) & 0xFF),
		byte((t.creationTime >> 8) & 0xFF),
		byte((t.creationTime) & 0xFF),
		/* Modification time */
		byte((t.modificationTime >> 24) & 0xFF),
		byte((t.modificationTime >> 16) & 0xFF),
		byte((t.modificationTime >> 8) & 0xFF),
		byte((t.modificationTime) & 0xFF),
		/* Track ID */
		0x0, 0x0, 0x0, 0x1,
		/* Reserved */
		0x0, 0x0, 0x0, 0x0,
		/* Duration */
		byte((t.duration >> 24) & 0xFF),
		byte((t.duration >> 16) & 0xFF),
		byte((t.duration >> 8) & 0xFF),
		byte((t.duration) & 0xFF),
		/* Reserved */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Layer */
		0x0, 0x0,
		/* Alternate group */
		0x0, 0x0,
		/* Volume */
		(volume & 0xFF), 0x0,
		/* Reserved */
		0x0, 0x0,
		/* Matrix */
		0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
		/* Width */
		byte((t.width >> 8) & 0xFF), byte((t.width) & 0xFF), 0x0, 0x0,
 		/* Height */
		byte((t.height >> 8) & 0xFF), byte((t.height) & 0xFF), 0x0, 0x0,
	})
}

func buildMDHD (t Track) ([]byte, error) {
	return utils.BuildAtom("mdhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Creation time */
		byte((t.creationTime >> 24) & 0xFF),
		byte((t.creationTime >> 16) & 0xFF),
		byte((t.creationTime >> 8) & 0xFF),
		byte((t.creationTime) & 0xFF),
		/* Modification time */
		byte((t.modificationTime >> 24) & 0xFF),
		byte((t.modificationTime >> 16) & 0xFF),
		byte((t.modificationTime >> 8) & 0xFF),
		byte((t.modificationTime) & 0xFF),
		/* Timescale */
		byte((t.timescale >> 24) & 0xFF),
		byte((t.timescale >> 16) & 0xFF),
		byte((t.timescale >> 8) & 0xFF),
		byte((t.timescale) & 0xFF),
		/* Duration */
		byte((t.duration >> 24) & 0xFF),
		byte((t.duration >> 16) & 0xFF),
		byte((t.duration >> 8) & 0xFF),
		byte((t.duration) & 0xFF),
		/* Language */
		0x55, 0xC4, 0x0, 0x0,
	})
}

func buildHDLR (t Track) ([]byte, error) {
	var handler uint32
	var name string
	if t.isAudio {
		handler = 0x736f756e
		name = "SoundHandler"
	} else {
		handler = 0x76696465
		name = "VideoHandler"
	}
	return utils.BuildAtom("hdlr", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Predefined */
		0x0, 0x0, 0x0, 0x0,
 		/* Handler */
		byte((handler >> 24) & 0xFF), byte((handler >> 16) & 0xFF),
		byte((handler >> 8) & 0xFF), byte((handler) & 0xFF),
		/* Reserved */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		/* Name */
		name[0], name[1], name[2], name[3], name[4], name[5], name[6],
		name[7], name[8], name[9], name[10], name[11], 0x0,
	})
}

func buildDREF(t Track) ([]byte, error) {
	return utils.BuildAtom("dref", []byte{
		0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x1,
		0x0, 0x0, 0x0, 0xC,
		0x75, 0x72, 0x6C, 0x20,
		0x0, 0x0, 0x0, 0x1,
	})
}

func buildVMHD(t Track) ([]byte, error) {
	return utils.BuildAtom("vmhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x1,
		/* Graphics mode */
		0x0, 0x0,
		/* OP color */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	})
}

func buildSMHD(t Track) ([]byte, error) {
	return utils.BuildAtom("smhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Balance */
		0x0, 0x0,
		/* Reserved */
		0x0, 0x0, 0x0, 0x0,
	})
}

func buildMFHD(t Track) ([]byte, error) {
	return utils.BuildAtom("mfhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Sequence number */
		0x0, 0x0, 0x0, 0x1,
	})
}

func buildTFDT(t Track) ([]byte, error) {
	return utils.BuildAtom("tfdt", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Base media decode time */
		byte((t.samples[0].pts >> 24) & 0xFF),
		byte((t.samples[0].pts >> 16) & 0xFF),
		byte((t.samples[0].pts >> 8) & 0xFF),
		byte((t.samples[0].pts) & 0xFF),
	})
}

func buildFREE(t Track) ([]byte, error) {
	return utils.BuildAtom("free", []byte("DashMe"))
}

func buildSIDX(t Track) ([]byte, error) {
	size := t.computeMOOFSize()
	size += t.computeMDATSize()
	duration := t.computeChunkDuration()
	res, err := utils.BuildAtom("sidx", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x0,
		/* Reference id */
		0x0, 0x0, 0x0, 0x1,
		/* Timescale */
		byte((t.timescale >> 24) & 0xFF),
		byte((t.timescale >> 16) & 0xFF),
		byte((t.timescale >> 8) & 0xFF),
		byte((t.timescale) & 0xFF),
		/* Earliest presentation time */
		byte((t.samples[0].pts >> 24) & 0xFF),
		byte((t.samples[0].pts >> 16) & 0xFF),
		byte((t.samples[0].pts >> 8) & 0xFF),
		byte((t.samples[0].pts) & 0xFF),
		/* First Offset */
		0x0, 0x0, 0x0, 0x0,
		/* Reserved */
		0x0, 0x0,
		/* Reference count = 1 */
		0x0, 0x1,
		/* Reference type + reference size*/
		byte((size >> 24) & 0xFF),
		byte((size >> 16) & 0xFF),
		byte((size >> 8) & 0xFF),
		byte((size) & 0xFF),
		/* Subsegment duration */
		byte((duration >> 24) & 0xFF),
		byte((duration >> 16) & 0xFF),
		byte((duration >> 8) & 0xFF),
		byte((duration) & 0xFF),
		/* Starts with SAP + SAP type + SAP delta time  */
		0x90, 0x0, 0x0, 0x0,
	})
	return res, err
}

func buildTFHD(t Track) ([]byte, error) {
	return utils.BuildAtom("tfhd", []byte{
		/* Flags + version */
		0x0, 0x2, 0x0, 0x0,
		/* Track id */
		0x0, 0x0, 0x0, 0x1,
	})
}

func buildSTYP(t Track) ([]byte, error) {
	return utils.BuildAtom("styp", []byte{
		/* Major brand */
		0x64, 0x61, 0x73, 0x68,
		/* Minor version */
		0x0, 0x0, 0x0, 0x0,
		/* Compatibility */
		0x69, 0x73, 0x6f, 0x36, 0x61, 0x76, 0x63, 0x31, 0x6d, 0x70,
		0x34, 0x31,
	})
}

func buildMDAT(t Track) ([]byte, error) {
	var b []byte
	for i := 0; i < len(t.samples); i++ {
	 	b = append(b, t.samples[i].GetData()...)
	 }
	return utils.BuildAtom("mdat", b)
}

func buildTRUN(t Track) ([]byte, error) {
	var b []byte
	var size int
	var composition int
	var flags int
	for i :=0; i < len(t.samples); i++ {
		size = int(t.samples[i].size)
		composition = t.samples[i].pts - t.samples[i].dts
		if t.samples[i].keyFrame {
			flags = 0x02400004
		} else {
			flags = 0x014100C0
		}
		b = append(b, []byte{
			/* Sample duration */
			byte((t.samples[i].duration >> 24) & 0xFF),
			byte((t.samples[i].duration >> 16) & 0xFF),
			byte((t.samples[i].duration >> 8) & 0xFF),
			byte((t.samples[i].duration) & 0xFF),
			/* Sample size */
			byte((size >> 24) & 0xFF),
			byte((size >> 16) & 0xFF),
			byte((size >> 8) & 0xFF),
			byte((size) & 0xFF),
			/* Sample duration */
			byte((flags >> 24) & 0xFF),
			byte((flags >> 16) & 0xFF),
			byte((flags >> 8) & 0xFF),
			byte((flags) & 0xFF),
			/* Sample composition offset */
			byte((composition >> 24) & 0xFF),
			byte((composition >> 16) & 0xFF),
			byte((composition >> 8) & 0xFF),
			byte((composition) & 0xFF),
		}...)
	}
	count := len(t.samples)
	offset := t.computeMOOFSize() + 8
	return utils.BuildAtom("trun", append([]byte{
		/* Flags + version */
		0x1, 0x0, 0xF, 0x1,
		/* Sample count */
		byte((count >> 24) & 0xFF),
		byte((count >> 16) & 0xFF),
		byte((count >> 8) & 0xFF),
		byte((count) & 0xFF),
		/* Data Offset */
		byte((offset >> 24) & 0xFF),
		byte((offset >> 16) & 0xFF),
		byte((offset >> 8) & 0xFF),
		byte((offset) & 0xFF),
	}, b...))
}

/* Builder structure methods */

/* Initialise builder building function map */
func (b *Builder) Initialise() {
	b.builders = make(map[string]AtomBuilder)
	b.builders["ftyp"] = buildFTYP /**/
	b.builders["free"] = buildFREE /**/
	b.builders["moov"] = buildMOOV /**/
	b.builders["mvhd"] = buildMVHD /**/
	b.builders["mvex"] = buildMVEX /**/
	b.builders["trex"] = buildTREX /**/
	b.builders["trak"] = buildTRAK /**/
	b.builders["tkhd"] = buildTKHD /**/
	b.builders["mdia"] = buildMDIA /**/
	b.builders["mdhd"] = buildMDHD /**/
	b.builders["hdlr"] = buildHDLR /**/
	b.builders["minf"] = buildMINF /**/
	b.builders["dinf"] = buildDINF /**/
	b.builders["dref"] = buildDREF /**/
	b.builders["stbl"] = buildSTBL /**/
	b.builders["vmhd"] = buildVMHD /**/
	b.builders["smhd"] = buildSMHD /**/
	b.builders["stsd"] = buildSTSD /**/
	b.builders["stts"] = buildSTTS /**/
	b.builders["stsc"] = buildSTSC /**/
	b.builders["stco"] = buildSTCO /**/
	b.builders["stsz"] = buildSTSZ /**/
	b.builders["stss"] = buildSTSS /**/
	b.builders["styp"] = buildSTYP /**/
	b.builders["sidx"] = buildSIDX /**/
	b.builders["moof"] = buildMOOF /**/
	b.builders["mfhd"] = buildMFHD /**/
	b.builders["traf"] = buildTRAF /**/
	b.builders["tfhd"] = buildTFHD /**/
	b.builders["tfdt"] = buildTFDT /**/
	b.builders["trun"] = buildTRUN /**/
	b.builders["mdat"] = buildMDAT /**/
	b.builders["mp4a"] = buildMP4A /**/
	b.builders["esds"] = buildESDS /**/
	b.builders["avcC"] = buildAVCC /**/
	b.builders["avc1"] = buildAVC1 /**/
}

/* Build atoms from their tag passed as string */
func (b Builder) build(t Track, atoms ...string) ([]byte, error) {
	var buf []byte
	var tmp []byte
	var err error
	for i := 0; i < len(atoms); i++ {
		tmp, err = b.builders[atoms[i]](t)
		if err != nil { return nil, err }
		buf = append(buf, tmp...)
	}
	return buf, nil
}

/* Track structure methods */

/* Compute size of to be generated MOOF atom */
func (t *Track) computeMOOFSize() int {
	return 16 + /* MFHD size */
		8 + /* TRAF header size */
		16 + /* TFHD size*/
		16 + /* TFDT size */
		20 + 16 * len(t.samples) + /* TRUN size */
		8 /* MOOF header size */
}

/* Compute duration of to be generated chunk */
func (t *Track) computeChunkDuration() int {
	duration := 0
	for i := 0; i < len(t.samples); i++ {
		duration += t.samples[i].duration
	}
	return duration
}

/* Compute size of to be generated MDAT atom */
func (t *Track) computeMDATSize() int {
	acc := 0
	for i := 0; i < len(t.samples); i++ {
		acc += int(t.samples[i].size)
	}
	return acc + 8
}

/* Build atoms from their tag passed as string */
func (t *Track) buildAtoms(atoms ...string) ([]byte, error) {
	return t.builder.build(*t, atoms...)
}

/* Append a sample to the track sample slice */
func (t *Track) appendSample(sample *Sample) {
	t.samples = append(t.samples, sample)
}

/* Build an init chunk from internal information */
func (t *Track) buildInitChunk(path string) error {
	/* Build init chunk atoms */
	b, err := t.buildAtoms("ftyp", "free", "moov")
	if err != nil {
		return err
	}
	/* Open file */
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	/* Write generated atoms */
	_, err = f.Write(b)
	return err
}

/* Build a chunk with samples from internal information */
func (t *Track) buildSampleChunk(samples []*Sample, path string) (int, error) {
	/* Set samples for this chunk to the builder */
	//t.builder.samples = samples
	/* Build chunk atoms */
	b, err := t.buildAtoms("styp", "free", "sidx", "moof", "mdat")
	if err != nil {
		return 0, err
	}
	/* Open file */
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	/* Write generated atoms */
	_, err = f.Write(b)
	return t.computeChunkDuration(), err
}

func (t *Track) InitialiseBuild(path string) error {
	t.computeBandwidth()
	t.builder = Builder{}
	t.builder.Initialise()
	if !utils.FileExist(path) {
		os.MkdirAll(path, os.ModeDir|os.ModePerm)
	} else if !utils.IsDirectory(path) {
		return errors.New("Path '" + path + "' is not a directory")
	}
	return nil
}

func (t *Track) BuildInit(path string) error {
	var typename string
	if (t.isAudio) {
		typename = "audio"
	} else {
		typename = "video"
	}
	return t.buildInitChunk(filepath.Join(path, "init_" + typename + ".mp4"))
}

func (t *Track) BuildChunk(path string) error {
	if (len(t.samples) <= 0) {
		return nil
	}
	var typename string
	var err error
	if (t.isAudio) {
		typename = "audio"
	} else {
		typename = "video"
	}
	duration := 0
	filename := "chunk_" + typename + "_" + strconv.Itoa(t.currentDuration) + ".mp4"
	/* Generate one chunk */
	duration, err = t.buildSampleChunk(t.samples, filepath.Join(path, filename))
	t.chunksDuration = append(t.chunksDuration, duration)
	t.currentDuration += duration
	return err
}

func (t *Track) buildVideoManifest() string {
	res := `
    <AdaptationSet
      group="` + strconv.Itoa(t.index) + `"
      mimeType="video/mp4"
      par="16:9"
      minBandwidth="` + strconv.Itoa(t.bandwidth) + `"
      maxBandwidth="` + strconv.Itoa(t.bandwidth) + `"
      minWidth="` + strconv.Itoa(t.width) + `"
      maxWidth="` + strconv.Itoa(t.width) + `"
      minHeight="` + strconv.Itoa(t.height) + `"
      maxHeight="` + strconv.Itoa(t.height) + `"
      segmentAlignment="true"
      startWithSAP="1">
      <SegmentTemplate
        timescale="` + strconv.Itoa(t.timescale) + `"
        initialization="init_$RepresentationID$.mp4"
        media="chunk_$RepresentationID$_$Time$.mp4"
        startNumber="1">
        <SegmentTimeline>`
	for i, duration := range t.chunksDuration {
		if i == 0 {
			res += `
          <S t="0" d="` + strconv.Itoa(duration) + `" />`
		} else {
			res += `
          <S d="` + strconv.Itoa(duration) + `" />`
		}
	}
	res += `
        </SegmentTimeline>
      </SegmentTemplate>
      <Representation
        id="video"
        bandwidth="` + strconv.Itoa(t.bandwidth) + `"
        codecs="` + t.codec + `"
        width="` + strconv.Itoa(t.width) + `"
        height="` + strconv.Itoa(t.height) + `">
        <AudioChannelConfiguration
          schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011"
          value="2">
        </AudioChannelConfiguration>
      </Representation>
    </AdaptationSet>`
	return res
}

func (t *Track) buildAudioManifest() string {
	res := `
    <AdaptationSet
      group="` + strconv.Itoa(t.index) + `"
      mimeType="audio/mp4"
      minBandwidth="` + strconv.Itoa(t.bandwidth) + `"
      maxBandwidth="` + strconv.Itoa(t.bandwidth) + `"
      segmentAlignment="true">
      <SegmentTemplate
        timescale="` + strconv.Itoa(t.timescale) + `"
        initialization="init_$RepresentationID$.mp4"
        media="chunk_$RepresentationID$_$Time$.mp4"
        startNumber="1">
        <SegmentTimeline>`
	for i, duration := range t.chunksDuration {
		if i == 0 {
			res += `
          <S t="0" d="` + strconv.Itoa(duration) + `" />`
		} else {
			res += `
          <S d="` + strconv.Itoa(duration) + `" />`
		}
	}
	res += `
        </SegmentTimeline>
      </SegmentTemplate>
      <Representation
        id="audio"
        bandwidth="` + strconv.Itoa(t.bandwidth) + `"
        codecs="` + t.codec + `"
        audioSamplingRate="` + strconv.Itoa(t.sampleRate) + `">
        <AudioChannelConfiguration
          schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011"
          value="2">
        </AudioChannelConfiguration>
      </Representation>
    </AdaptationSet>`
	return res
}

/* Compute bandwidth for a track */
func (t *Track) computeBandwidth() {
	totalDuration := 0
	totalSize := 0
	for _, sample := range t.samples {
		totalDuration += sample.duration
		totalSize += int(sample.size)
	}
	totalDuration /= t.timescale
	if totalDuration > 0 {
		t.bandwidth =  totalSize / totalDuration
	} else {
		t.bandwidth = 0
	}
}

/* Compute codec name from extradata for audio */
func (t *Track) extractAudioCodec() {
	t.codec = "mp4a.40.2"
}

/* Compute codec name from extradata for video */
func (t *Track) extractVideoCodec() {
	t.codec = "avc1." + strings.ToUpper(hex.EncodeToString(t.extradata[1:2]) + hex.EncodeToString(t.extradata[2:3]) + hex.EncodeToString(t.extradata[3:4]))
}

/* Compute codec name from extradata */
func (t *Track) extractCodec() {
	if t.isAudio {
		t.extractAudioCodec()
	} else {
		t.extractVideoCodec()
	}
}

/* Build all chunk with samples from internal information */
func (t *Track) BuildAdaptationSet() string {
	t.extractCodec()
	if t.isAudio {
		return t.buildAudioManifest()
	} else {
		return t.buildVideoManifest()
	}
}

/* Set creationTime and modificationTime in Track structure */
func (t *Track) SetTimeFields() {
	t.creationTime = int(time.Since(time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)).Seconds())
	t.modificationTime = t.creationTime
}

/* Return track duration */
func (t *Track) Duration() float64 {
	return float64(t.duration) / float64(t.timescale)
}

/* Return largest duration of segments in track */
func (t *Track) MaxChunkDuration() float64 {
	duration := 0
	for i := 0; i < len(t.chunksDuration); i++ {
		if t.chunksDuration[i] > duration {
			duration = t.chunksDuration[i]
		}
	}
	return float64(duration) / float64(t.timescale)
}

/* Return largest duration of segments in track */
func (t *Track) MinBufferTime() float64 {
	size := 0
	for i := 0; i < len(t.samples); i++ {
		if int(t.samples[i].size) > size {
			size = int(t.samples[i].size)
		}
	}
	return float64(size) / float64(t.bandwidth)
}

func (t *Track) Clean() {
	for i := 0; i < len(t.samples); i++ {
		t.samples[i] = nil
	}
	t.samples = t.samples[:0]
	t.samples = nil
}
