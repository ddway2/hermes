package hermes

import "bytes"

type Serialize interface {
	Write(b *bytes.Buffer) error
}

type Deserialize interface {
	Read(b *bytes.Buffer) error
}

type Packet struct {
	Data bytes.Buffer
	Size int
}

type InitPackage struct {
}

type Flag uint8

const (
	FlagPush Flag = 1 << iota
	FlagAck
	FlagReset
)

type Package struct {
	Seq  uint16
	Ack  uint16
	Flag Flag
	Crc  uint16
	Data []byte
}
