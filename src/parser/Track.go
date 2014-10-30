package parsers

import (
	"os"
	"fmt"
	"utils"
	"bytes"
	"errors"
	"path/filepath"
	"encoding/binary"
)

type Sample struct {
	pts      int
	dts      int
	duration int
	isKey    bool
	data     []byte
}

type Track struct {
	isAudio	         bool
	creationTime     int
	duration         int
	modificationTime int
	timescale        int
	width            int
	height           int
	extradata        []byte
	samples          []Sample
}

func (t *Track) Print() {
	fmt.Println("Track :")
	fmt.Println("\tisAudio : ", t.isAudio)
	fmt.Println("\tcreationTime : ", t.creationTime)
	fmt.Println("\tmodificationTime : ", t.modificationTime)
	fmt.Println("\tduration : ", t.duration)
	fmt.Println("\ttimescale : ", t.timescale)
	fmt.Println("\twidth : ", (t.width >> 16) & 0xFFFF)
	fmt.Println("\theight : ", (t.height >> 16) & 0xFFFF)
}

type AtomBuilder func(t Track) ([]byte, error)

var builders map[string]AtomBuilder

/* Atom generic building functions */
func buildAtom(tag string, content []byte) ([]byte, error) {
	var b bytes.Buffer
	err := binary.Write(&b, binary.BigEndian, int32(len(content) + 8))
	if err != nil { return nil, err }
	err = binary.Write(&b, binary.LittleEndian, []byte(tag))
	if err != nil { return nil, err }
	err = binary.Write(&b, binary.BigEndian, content)
	if err != nil { return nil, err }
	return b.Bytes(), nil
}

func buildEmptyAtom(tag string, size int) ([]byte, error) {
	return buildAtom(tag, make([]byte, size))
}

func buildAtomFromMap(t Track, atoms ...string) ([]byte, error) {
	var buf []byte
	var tmp []byte
	var err error

	for _, atom := range atoms {
		tmp, err = builders[atom](t)
		if err != nil { return nil, err }
		buf = append(buf, tmp...)
	}
	return buf, nil
}

/* Atom specific building functions */

func buildSTTS(t Track) ([]byte, error) {
	return buildEmptyAtom("stts", 16)
}

func buildSTSC(t Track) ([]byte, error) {
	return buildEmptyAtom("stsc", 16)
}

func buildSTCO(t Track) ([]byte, error) {
	return buildEmptyAtom("stco", 16)
}

func buildSTSZ(t Track) ([]byte, error) {
	return buildEmptyAtom("stsz", 20)
}

func buildSTSS(t Track) ([]byte, error) {
	return buildEmptyAtom("stss", 16)
}

func buildSTBL(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "stsd", "stts", "stsc", "stco", "stsz", "stss")
	if err != nil { return nil, err }
	return buildAtom("stbl", b)
}

func buildMINF(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "dinf", "stbl", "vmhd")
	if err != nil { return nil, err }
	return buildAtom("minf", b)
}

func buildMDIA(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "mdhd", "hdlr", "minf")
	if err != nil { return nil, err }
	return buildAtom("mdia", b)
}

func buildTRAK(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "tkhd", "mdia")
	if err != nil { return nil, err }
	return buildAtom("trak", b)
}

func buildMVEX(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "trex")
	if err != nil { return nil, err }
	return buildAtom("mvex", b)
}

func buildMOOV(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "mvhd", "mvex", "trak")
	if err != nil { return nil, err }
	return buildAtom("moov", b)
}

func buildDINF(t Track) ([]byte, error) {
	b, err := buildAtomFromMap(t, "dref")
	if err != nil {
		return nil, err
	}
	return buildAtom("dinf", b)
}

func buildSTSD(t Track) ([]byte, error) {
	return buildAtom("stsd", append([]byte{
		/* Major brand */
		0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x1,
	}, t.extradata...))
}

func buildFTYP(t Track) ([]byte, error) {
	return buildAtom("ftyp", []byte{
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
	return buildAtom("mvhd", []byte{
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
	return buildAtom("trex", []byte{
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
	return buildAtom("tkhd", []byte{
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
		byte((t.width >> 24) & 0xFF), byte((t.width >> 16) & 0xFF),
		byte((t.width >> 8) & 0xFF), byte((t.width) & 0xFF),
 		/* Height */
		byte((t.height >> 24) & 0xFF), byte((t.height >> 16) & 0xFF),
		byte((t.height >> 8) & 0xFF), byte((t.height) & 0xFF),
	})
}

func buildMDHD (t Track) ([]byte, error) {
	return buildAtom("mdhd", []byte{
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
	return buildAtom("hdlr", []byte{
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
	return buildAtom("dref", []byte{
		0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x1,
		0x0, 0x0, 0x0, 0xC,
		0x75, 0x72, 0x6C, 0x20,
		0x0, 0x0, 0x0, 0x1,
	})
}

func buildVMHD(t Track) ([]byte, error) {
	return buildAtom("vmhd", []byte{
		/* Flags + version */
		0x0, 0x0, 0x0, 0x1,
		/* Graphics mode */
		0x0, 0x0,
		/* OP color */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	})
}

/* Track building core functions */

func InitialiseTrackBuilders() {
	builders = make(map[string]AtomBuilder)
	builders["ftyp"] = buildFTYP /**/
	builders["moov"] = buildMOOV /**/
	builders["mvhd"] = buildMVHD /**/
	builders["mvex"] = buildMVEX /**/
	builders["trex"] = buildTREX /**/
	builders["trak"] = buildTRAK /**/
	builders["tkhd"] = buildTKHD /**/
	builders["mdia"] = buildMDIA /**/
	builders["mdhd"] = buildMDHD /**/
	builders["hdlr"] = buildHDLR /**/
	builders["minf"] = buildMINF /**/
	builders["dinf"] = buildDINF /**/
	builders["dref"] = buildDREF /**/
	builders["stbl"] = buildSTBL /**/
	builders["vmhd"] = buildVMHD /**/
	builders["stsd"] = buildSTSD /**/
	builders["stts"] = buildSTTS /**/
	builders["stsc"] = buildSTSC /**/
	builders["stco"] = buildSTCO /**/
	builders["stsz"] = buildSTSZ /**/
	builders["stss"] = buildSTSS /**/
}

func (t *Track) buildInitChunk(path string) error {
	var filename string
	if (t.isAudio) {
		filename = "init_audio.mp4"
	} else {
		filename = "init_video.mp4"
	}
	b, err := buildAtomFromMap(*t, "ftyp", "moov")
	if err != nil {
		return err
	}
	fmt.Printf("Generation done in %q\n", filepath.Join(path, filename))
	f, err := os.OpenFile(filepath.Join(path, filename), os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Printf("Error while opening : " + err.Error() + "\n")
		return err
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		fmt.Printf("Error while writing : " + err.Error() + "\n")
	}
	return err
}

func (t *Track) BuildChunks(count int, path string) error {
	if !utils.FileExist(path) {
		os.MkdirAll(path, os.ModeDir|os.ModePerm)
	} else if !utils.IsDirectory(path) {
		return errors.New("Path '" + path + "' is not a directory")
	}
	return t.buildInitChunk(path)
}
