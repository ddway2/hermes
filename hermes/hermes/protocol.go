package hermes

import (
	"bytes"
	"encoding/binary"
	"net"
)

type Serialize interface {
	Write(b *bytes.Buffer) error
}

type Deserialize interface {
	Read(b *bytes.Buffer) error
}

type Packet struct {
	Addr  *net.UDPAddr
	Index int
	Data  bytes.Buffer
	Size  int
}

type InitPackage struct {
}

type Flag uint16

const (
	FlagPush Flag = 1 << iota
	FlagAck
	FlagReset
)

type PacketLayer struct {
	Seq  uint16
	Ack  uint16
	Flag Flag
	Crc  uint16
	Data [100]byte // Only 100 bytes
}

func (p *PacketLayer) Write(b *bytes.Buffer) error {
	binary.Write(b, binary.LittleEndian, p)
	return nil
}

func (p *PacketLayer) Read(b *bytes.Buffer) error {
	binary.Read(b, binary.LittleEndian, p)
	return nil
}
