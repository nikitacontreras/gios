package codesign

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"
)

// Sign pseudo-signs a Mach-O binary (ad-hoc) and injects entitlements if provided.
func Sign(path string, entitlements string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	// Check if it's a FAT binary or single Mach-O
	fat, err := macho.OpenFat(path)
	if err == nil {
		// FAT binary
		defer fat.Close()
		// For now, let's focus on single-arch implementation for the POC
		return fmt.Errorf("FAT binary support not yet fully implemented in Go version")
	}

	m, err := macho.Open(path)
	if err != nil {
		return fmt.Errorf("failed to parse Mach-O: %v", err)
	}
	defer m.Close()

	return signMachO(f, m, entitlements)
}

func signMachO(f *os.File, m *macho.File, entitlements string) error {
	// 1. Calculate the Code Directory hashes
	info, _ := f.Stat()
	fileSize := info.Size()

	// If there's an existing signature, we'll replace it.
	var oldSigOffset int64
	if cs := m.CodeSignature(); cs != nil {
		oldSigOffset = int64(cs.Offset)
		// We will truncate the file to the start of the old signature for hashing
		fileSize = oldSigOffset
	}

	// 2. Prepare Blobs
	cd := CodeDirectory{
		BlobHeader: BlobHeader{
			Magic: CSMAGIC_CODEDIRECTORY,
		},
		Version:       0x20400, // Supports SHA256
		Flags:         0,       // Ad-hoc
		NSpecialSlots: 0,
		NCodeSlots:    uint32((fileSize + 0xfff) / 0x1000),
		CodeLimit:     uint32(fileSize),
		HashSize:      32,  // SHA256
		HashType:      2,   // SHA256
		PageSize:      12,  // 2^12 = 4096
	}

	// Entitlements blob
	var entBlob []byte
	if entitlements != "" {
		entBlob = append([]byte(nil), []byte(entitlements)...)
		entBlob = append(entBlob, 0)
		cd.NSpecialSlots = 5
	}

	// Identifier (just use a default)
	ident := "com.nikitastrike.gios.ad-hoc"
	identBlob := append([]byte(ident), 0)

	// Calculate CD length
	cd.IdentOffset = uint32(binary.Size(cd))
	cd.HashOffset = cd.IdentOffset + uint32(len(identBlob))
	if cd.NSpecialSlots > 0 {
		cd.HashOffset += cd.NSpecialSlots * uint32(cd.HashSize)
	}
	cd.BlobHeader.Length = cd.HashOffset + cd.NCodeSlots*uint32(cd.HashSize)

	// Build the full signature buffer
	sigBuf := new(bytes.Buffer)
	count := uint32(1)
	if entBlob != nil {
		count++
	}

	sb := SuperBlob{
		BlobHeader: BlobHeader{
			Magic: CSMAGIC_EMBEDDED_SIGNATURE,
		},
		Count: count,
	}

	sbSize := uint32(binary.Size(sb)) + count*uint32(binary.Size(BlobIndex{}))
	cdOffset := sbSize
	var entOffset uint32
	if entBlob != nil {
		entOffset = cdOffset + cd.BlobHeader.Length
	}
	sb.BlobHeader.Length = cdOffset + cd.BlobHeader.Length
	if entBlob != nil {
		sb.BlobHeader.Length += uint32(binary.Size(BlobHeader{})) + uint32(len(entBlob))
	}

	sb.Write(sigBuf)
	binary.Write(sigBuf, binary.BigEndian, BlobIndex{Type: CS_SLOT_CODEDIRECTORY, Offset: cdOffset})
	if entBlob != nil {
		binary.Write(sigBuf, binary.BigEndian, BlobIndex{Type: CS_SLOT_ENTITLEMENTS, Offset: entOffset})
	}

	cd.Write(sigBuf)
	sigBuf.Write(identBlob)

	if cd.NSpecialSlots > 0 {
		hashArray := make([]byte, cd.NSpecialSlots*uint32(cd.HashSize))
		if entBlob != nil {
			entFullBlob := new(bytes.Buffer)
			binary.Write(entFullBlob, binary.BigEndian, BlobHeader{Magic: CSMAGIC_EMBEDDED_ENTITLEMENTS, Length: uint32(len(entBlob)) + 8})
			entFullBlob.Write(entBlob)
			h := sha256.Sum256(entFullBlob.Bytes())
			copy(hashArray[(cd.NSpecialSlots-5)*uint32(cd.HashSize):], h[:])
		}
		sigBuf.Write(hashArray)
	}

	for i := uint32(0); i < cd.NCodeSlots; i++ {
		offset := int64(i) * 0x1000
		size := int64(0x1000)
		if offset+size > fileSize {
			size = fileSize - offset
		}
		page := make([]byte, size)
		f.ReadAt(page, offset)
		h := sha256.Sum256(page)
		sigBuf.Write(h[:])
	}

	if entBlob != nil {
		binary.Write(sigBuf, binary.BigEndian, BlobHeader{Magic: CSMAGIC_EMBEDDED_ENTITLEMENTS, Length: uint32(len(entBlob)) + 8})
		sigBuf.Write(entBlob)
	}

	if oldSigOffset > 0 {
		f.Truncate(oldSigOffset)
	}
	padding := (16 - (fileSize % 16)) % 16
	if padding > 0 {
		f.WriteAt(make([]byte, padding), fileSize)
		fileSize += padding
	}
	f.WriteAt(sigBuf.Bytes(), fileSize)

	return updateLoadCommands(f, m, uint32(fileSize), uint32(sigBuf.Len()))
}

func updateLoadCommands(f *os.File, m *macho.File, sigOffset, sigSize uint32) error {
	foundCS := false

	// Calculate start of load commands
	var headerSize int64
	if m.Magic == types.Magic64 {
		headerSize = 32 // Size of Mach-O 64-bit header
	} else {
		headerSize = 28 // Size of Mach-O 32-bit header
	}

	offset := headerSize
	for _, l := range m.Loads {
		cmd := l.Command()
		currPos := offset

		if cmd == types.LC_CODE_SIGNATURE {
			var lcs types.CodeSignatureCmd
			f.Seek(int64(currPos), io.SeekStart)
			binary.Read(f, m.ByteOrder, &lcs)
			lcs.Offset = sigOffset
			lcs.Size = sigSize
			f.Seek(int64(currPos), io.SeekStart)
			binary.Write(f, m.ByteOrder, lcs)
			foundCS = true
		} else if cmd == types.LC_SEGMENT || cmd == types.LC_SEGMENT_64 {
			if cmd == types.LC_SEGMENT_64 {
				var seg types.Segment64
				f.Seek(int64(currPos), io.SeekStart)
				binary.Read(f, m.ByteOrder, &seg)
				segName := string(bytes.TrimRight(seg.Name[:], "\x00"))
				if segName == "__LINKEDIT" {
					newFilesz := uint64(sigOffset) + uint64(sigSize) - seg.Offset
					if newFilesz > seg.Filesz {
						seg.Filesz = newFilesz
						seg.Memsz = (newFilesz + 0xfff) &^ 0xfff
						f.Seek(int64(currPos), io.SeekStart)
						binary.Write(f, m.ByteOrder, seg)
					}
				}
			} else {
				var seg types.Segment32
				f.Seek(int64(currPos), io.SeekStart)
				binary.Read(f, m.ByteOrder, &seg)
				segName := string(bytes.TrimRight(seg.Name[:], "\x00"))
				if segName == "__LINKEDIT" {
					newFilesz := uint32(sigOffset) + uint32(sigSize) - seg.Offset
					if newFilesz > seg.Filesz {
						seg.Filesz = newFilesz
						seg.Memsz = (newFilesz + 0xfff) &^ 0xfff
						f.Seek(int64(currPos), io.SeekStart)
						binary.Write(f, m.ByteOrder, seg)
					}
				}
			}
		}
		offset += int64(len(l.Raw()))
	}

	if !foundCS {
		return fmt.Errorf("LC_CODE_SIGNATURE not found")
	}

	return nil
}





// PseudoSign shorthand for easy use
func PseudoSign(path string, entitlements string) error {
	return Sign(path, entitlements)
}
