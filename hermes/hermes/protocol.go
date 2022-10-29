package hermes

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/go-git/go-git/v5/utils/binary"
)

type Serialize interface {
	Write(b *bytes.Buffer) error
}

type Deserialize interface {
	Read(b *bytes.Buffer) error
}

type Packet struct {
	Addr *net.UDPAddr
	Data bytes.Buffer
	Size int
}

type Flag uint32

const (
	PTYPE_INIT Flag = iota
	PTYPE_CHALLENGE
	PTYPE_CHALLENGE_ACK
	PTYPE_DATA
	PTYPE_ACK
	PTYPE_DISCONNECT
)

type PHeader struct {
	Game uint32
	Type Flag
	Id   uint32
	Crc  uint32
}

type PMsgInit struct {
	Salt uint32
	Addr uint32
	Port uint16
}

func (h *PHeader) Deserialize(buf *bytes.Buffer) error {
	if buf.Len() < 16 {
		return fmt.Errorf("PHeader read error")
	}
	rbuf := buf.Next(16)
	h.Game = binary.LittleEndian.Uint32(rbuf[0:])
	h.Type = Flag(binary.LittleEndian.Uint32(rbuf[4:]))
	h.Id = binary.LittleEndian.Uint32(rbuf[8:])
	h.Crc = binary.LittleEndian.Uint32(rbuf[12:])
	return nil
}

func (h *PMsgInit) Deserialize(buf *bytes.Buffer) error {
	if buf.Len() < 10 {
		return fmt.Errorf("PMsgInit read error")
	}
	rbuf := buf.Next(10)
	h.Salt = binary.LittleEndian.Uint32(rbuf[0:])
	h.Addr = binary.LittleEndian.Uint32(rbuf[4:])
	h.Port = binary.LittleEndian.Uint16(rbuf[8:])
	return nil
}
