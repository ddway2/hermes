package hermes

type Deserialize interface {
	Read(buffer []byte) (uint32, error)
}

type Packet struct {
	Ptr  uint32
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
