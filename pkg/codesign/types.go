package codesign

import (
	"encoding/binary"
	"io"
)

const (
	CSMAGIC_EMBEDDED_SIGNATURE   = 0xfade0cc0
	CSMAGIC_CODEDIRECTORY        = 0xfade0c02
	CSMAGIC_EMBEDDED_ENTITLEMENTS = 0xfade7171

	CS_SLOT_CODEDIRECTORY = 0
	CS_SLOT_ENTITLEMENTS  = 5
)

type BlobHeader struct {
	Magic  uint32
	Length uint32
}

type SuperBlob struct {
	BlobHeader
	Count uint32
}

type BlobIndex struct {
	Type   uint32
	Offset uint32
}

type CodeDirectory struct {
	BlobHeader
	Version       uint32
	Flags         uint32
	HashOffset    uint32
	IdentOffset   uint32
	NSpecialSlots uint32
	NCodeSlots    uint32
	CodeLimit     uint32
	HashSize      uint8
	HashType      uint8
	Platform      uint8
	PageSize      uint8
	Reserved      uint32
}

// Write (BigEndian)
func (s *SuperBlob) Write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, s)
}

func (i *BlobIndex) Write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, i)
}

func (c *CodeDirectory) Write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, c)
}
