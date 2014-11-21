
package parser

import "utils"
import "encoding/hex"

/* Function type used for atom generation */
type AtomBuilder func(t Track) ([]byte, error)

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

func buildPSSH(systemId string, privateData []byte) ([]byte, error) {
	b, err := hex.DecodeString(systemId)
	if err != nil { return nil, err }
	size := len(privateData)
	b = append(b, []byte{
		byte((size >> 24) & 0xFF),
		byte((size >> 16) & 0xFF),
		byte((size >> 8) & 0xFF),
		byte((size) & 0xFF),
	}...)
	b = append(b, privateData...)
	return utils.BuildAtom("pssh", append([]byte{
		0x0, 0x0, 0x0, 0x0,
	}, b...))
}

func buildMOOV(t Track) ([]byte, error) {
	b, err := t.buildAtoms("mvhd", "mvex", "trak")
	if err != nil { return nil, err }
	if t.encryptInfos != nil {
		for i := 0; i < len(t.encryptInfos.pssList); i++ {
			pssh, err := buildPSSH(t.encryptInfos.pssList[i].systemId, t.encryptInfos.pssList[i].privateData)
			if err != nil { return nil, err }
			b = append(b, pssh...)
		}
	}
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
	var b []byte
	var err error
	if t.encryptInfos != nil {
		b, err = t.buildAtoms("tfhd", "tfdt", "trun", "senc", "saiz", "saio")
	} else {
		b, err = t.buildAtoms("tfhd", "tfdt", "trun")
	}
	if err != nil {
		return nil, err
	}
	return utils.BuildAtom("traf", b)
}

func buildSENC(t Track) ([]byte, error) {
	flags := 0
	length := len(t.samples)
	if t.encryptInfos.subEncrypt {
		flags = 0x2
	}
	b := []byte{
		0x0, 0x0, 0x0, byte(flags),
	}
	b = append(b, []byte{
		byte((length >> 24) & 0xFF),
		byte((length >> 16) & 0xFF),
		byte((length >> 8) & 0xFF),
		byte((length) & 0xFF),
	}...)
	for i := 0; i < len(t.samples); i++ {
		length = len(t.samples[i].encrypt.subEncrypt)
		b = append(b, t.samples[i].encrypt.initializationVector...)
		if t.encryptInfos.subEncrypt {
			b = append(b, []byte{
				byte((length >> 8) & 0xFF),
				byte((length) & 0xFF),
			}...)
			for j := 0; j < length; j++ {
				clear := t.samples[i].encrypt.subEncrypt[j].clear
				encrypted := t.samples[i].encrypt.subEncrypt[j].encrypted
				b = append(b, []byte{
					byte((clear >> 8) & 0xFF),
					byte((clear) & 0xFF),
					byte((encrypted >> 24) & 0xFF),
					byte((encrypted >> 16) & 0xFF),
					byte((encrypted >> 8) & 0xFF),
					byte((encrypted) & 0xFF),
				}...)
			}
		}
	}
	return utils.BuildAtom("senc", b)
}

func buildSAIZ(t Track) ([]byte, error) {
	length := len(t.samples)
	b := []byte{
		0x0, 0x0, 0x0, 0x0, 0x0,
		byte((length >> 24) & 0xFF),
		byte((length >> 16) & 0xFF),
		byte((length >> 8) & 0xFF),
		byte((length) & 0xFF),
	}
	for i := 0; i < len(t.samples); i++ {
		size := len(t.samples[i].encrypt.initializationVector)
		if t.encryptInfos.subEncrypt {
			size += 2 + len(t.samples[i].encrypt.subEncrypt) * 6
		}
		b = append(b, byte(size))
	}
	return utils.BuildAtom("saiz", b)
}

func buildSAIO(t Track) ([]byte, error) {
	offset := 8 + /* MOOF header */
		16 + /* MFHD */
		8 + /* TRAF header */
		16 + /* TFHD */
		20 + /* TFDT */
	 	20 + 16 * len(t.samples) + /* TRUN size */
		16 /* SENC Header */
	return utils.BuildAtom("saio", []byte{
		0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x1,
		byte((offset >> 24) & 0xFF),
		byte((offset >> 16) & 0xFF),
		byte((offset >> 8) & 0xFF),
		byte((offset) & 0xFF),
	})
}

func buildTENC(t Track) ([]byte, error) {
	b, err := hex.DecodeString(t.encryptInfos.keyId)
	if err != nil { return nil, err }
	return utils.BuildAtom("tenc", append([]byte{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x1, 0x8,
	}, b...))
}

func buildSCHI(t Track) ([]byte, error) {
	b, err := t.buildAtoms("tenc")
	if err != nil { return nil, err }
	return utils.BuildAtom("schi", b)
}

func buildSCHM(t Track) ([]byte, error) {
	return utils.BuildAtom("schm", []byte{
		0x0, 0x0, 0x0, 0x0,
		byte('c'), byte('e'), byte('n'), byte('c'),
		0x0, 0x1, 0x0, 0x0,
	})
}

func buildFRMA(t Track) ([]byte, error) {
	var name string
	if t.isAudio {
		name = "mp4a"
	} else {
		name = "avc1"
	}
	return utils.BuildAtom("frma", []byte(name))
}

func buildSINF(t Track) ([]byte, error) {
	b, err := t.buildAtoms("frma", "schm", "schi")
	if err != nil { return nil, err }
	return utils.BuildAtom("sinf", b)
}

func buildAVCC(t Track) ([]byte, error) {
	return utils.BuildAtom("avcC", t.extradata)
}

func buildAVC1ENCV(t Track) ([]byte, error) {
	var b []byte
	var err error
	var name string
	if t.encryptInfos == nil {
		name = "avc1"
		b, err = t.buildAtoms("avcC")
	} else {
		name = "encv"
		b, err = t.buildAtoms("avcC", "sinf")
	}
	if err != nil { return nil, err }
	return utils.BuildAtom(name, append([]byte{
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
		0x0, 0x0, 0x0, 0x0, 0x3, 0x80, 0x80, 0x80, 0x22, 0x0, 0x0,
		0x0, 0x4, 0x80, 0x80, 0x80, 0x14, 0x40, 0x15, 0x0, 0x1, 0x18, 0x0,
		0x1, 0x64, 0xf0, 0x0, 0x1, 0x44, 0x6b, 0x05, 0x80, 0x80, 0x80,
		byte((size) & 0xFF),
	}, data...)
	return utils.BuildAtom("esds",  data)
}

func buildMP4AENCA(t Track) ([]byte, error) {
	var b []byte
	var err error
	var name string
	if t.encryptInfos == nil {
		name = "mp4a"
		b, err = t.buildAtoms("esds")
	} else {
		name = "enca"
		b, err = t.buildAtoms("esds", "sinf")
	}
	if err != nil { return nil, err }
	return utils.BuildAtom(name, append([]byte{
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
	if t.encryptInfos == nil {
		if t.isAudio {
			b, err = t.buildAtoms("mp4a")
		} else {
			b, err = t.buildAtoms("avc1")
		}
	} else {
		if t.isAudio {
			b, err = t.buildAtoms("enca")
		} else {
			b, err = t.buildAtoms("encv")
		}
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
		byte((t.globalTimescale >> 24) & 0xFF),
		byte((t.globalTimescale >> 16) & 0xFF),
		byte((t.globalTimescale >> 8) & 0xFF),
		byte((t.globalTimescale) & 0xFF),
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
		0x1, 0x0, 0x0, 0x0,
		/* Base media decode time */
		byte((t.samples[0].pts >> 56) & 0xFF),
		byte((t.samples[0].pts >> 48) & 0xFF),
		byte((t.samples[0].pts >> 40) & 0xFF),
		byte((t.samples[0].pts >> 32) & 0xFF),
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
		0x1, 0x0, 0x0, 0x0,
		/* Reference id */
		0x0, 0x0, 0x0, 0x1,
		/* Timescale */
		byte((t.timescale >> 24) & 0xFF),
		byte((t.timescale >> 16) & 0xFF),
		byte((t.timescale >> 8) & 0xFF),
		byte((t.timescale) & 0xFF),
		/* Earliest presentation time */
		byte((t.samples[0].pts >> 56) & 0xFF),
		byte((t.samples[0].pts >> 48) & 0xFF),
		byte((t.samples[0].pts >> 40) & 0xFF),
		byte((t.samples[0].pts >> 32) & 0xFF),
		byte((t.samples[0].pts >> 24) & 0xFF),
		byte((t.samples[0].pts >> 16) & 0xFF),
		byte((t.samples[0].pts >> 8) & 0xFF),
		byte((t.samples[0].pts) & 0xFF),
		/* First Offset */
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
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
	var composition int64
	var flags int
	for i :=0; i < len(t.samples); i++ {
		size = int(t.samples[i].size)
		composition = t.samples[i].dts - t.samples[i].pts
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
