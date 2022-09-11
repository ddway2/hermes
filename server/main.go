package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

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

type DummyServerConnection struct {
	RemoteAddr *net.UDPAddr
}

func NewConnection() (hermes.UDPServerConn, error) {
	return &DummyServerConnection{}, nil
}

func (c *DummyServerConnection) OnNewConnect(s *hermes.UDPServer, addr *net.UDPAddr) error {
	c.RemoteAddr = addr
	return nil
}

func (c *DummyServerConnection) OnReceiveData(s *hermes.UDPServer, p *hermes.Packet) error {
	s.SendRawDataToClient(p.Addr, p)
	return nil
}

func main() {
	srv := hermes.NewUDPServer(NewConnection)

	srv.Listen(":4567")

	fmt.Println("Ready")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	srv.Close()
	fmt.Println("Done")
}
