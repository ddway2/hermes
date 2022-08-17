package hermes

import (
	"bytes"
	"encoding/binary"
	"testing"
)

type DummyPacket struct {
	Index uint64
}

func (p *DummyPacket) Write(b *bytes.Buffer) error {
	err := binary.Write(b, binary.LittleEndian, p.Index)
	return err
}

func (p *DummyPacket) Read(b *bytes.Buffer) error {
	err := binary.Read(b, binary.LittleEndian, &p.Index)
	return err
}

func BenchmarkConnect(b *testing.B) {
	c1 := NewUDPClient()
	srv := newUDPServer()

	srv.Listen(":4567")

	c1.Connect("127.0.0.1:4567")

	go func() {
		for {
			p := &DummyPacket{}
			if err := c1.ReceiveData(p); err != nil {
				return
			}
		}
	}()

	p2 := &DummyPacket{}
	for i := 0; i < b.N; i++ {
		p2.Index = uint64(i)
		c1.SendData(p2)
	}

	c1.Close()
}
