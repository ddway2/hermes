package hermes

import (
	"fmt"
	"testing"
)

type DummyPacket struct {
	Value string
}

func (p *DummyPacket) Write(b *bytes.Buffer) error {
	_, err := b.WriteString(p.Value)
	return err
}

func (p *DummyPacket) Read(b *bytes.Buffer) error {

	return nil
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
	for i := 0; i < 1000000; i++ {
		p2.Value = "toto"
		fmt.Printf("step: %v\n", i)
		c1.SendData(p2)
	}

	c1.Close()
}
