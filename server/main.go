package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"

	"ddway2.com/hermes/hermes"
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

func main() {
	srv := hermes.NewUDPServer()

	srv.Listen(":4567")

	fmt.Println("Ready")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	srv.Close()
	fmt.Println("Done")
}
