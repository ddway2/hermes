package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"ddway2.com/hermes/hermes"
)

type DummyPacket struct {
	Index uint64
	Data  []byte
}

func (p *DummyPacket) Write(b *bytes.Buffer) error {
	err := binary.Write(b, binary.LittleEndian, p.Index)
	binary.Write(b, binary.LittleEndian, p.Data)
	return err
}

func (p *DummyPacket) Read(b *bytes.Buffer) error {
	err := binary.Read(b, binary.LittleEndian, &p.Index)
	binary.Read(b, binary.LittleEndian, &p.Data)
	return err
}

func main() {
	c1 := hermes.NewUDPClient()

	c1.Connect("127.0.0.1:4567")

	var end atomic.Bool
	end.Store(false)

	startTime := time.Now()
	go func() {
		fmt.Println("Start sending Data")
		var i uint64 = 0
		p2 := &DummyPacket{
			Data: make([]byte, 1100),
		}
		for !end.Load() {
			p2.Index = uint64(i)
			c1.SendData(p2)
			i++

			if i%10000 == 0 {
				fmt.Printf("Send %v UDP packets\n", i)
			}
		}
	}()

	go func() {
		p := &DummyPacket{
			Data: make([]byte, 1100),
		}
		for !end.Load() {
			if err := c1.ReceiveData(p); err != nil {
				fmt.Println("Receive error")
			}
		}
	}()

	fmt.Println("Ready")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	end.Store(true)
	stopTime := time.Now()

	diffTime := stopTime.Sub(startTime)

	dT := diffTime.Seconds()

	fmt.Println("Stop required")

	fmt.Printf("Recv=%f KB/s Sent=%f KB/s\n", c1.Stats.BiterateRecv(dT)/1024, c1.Stats.BiterateSent(dT)/1024)
	fmt.Printf("Recv=%f pkt/s Sent=%f okt/s\n", c1.Stats.PacketsRecv(dT), c1.Stats.PacketsSent(dT))

	c1.Close()
	fmt.Println("Done")
}
