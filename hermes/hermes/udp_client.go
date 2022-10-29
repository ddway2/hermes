package hermes

import (
	"fmt"
	"net"
	"time"
)

const (
	CLIENTUDP_DEFAULT_OutputTimeoutMs  = 50
	CLIENTUDP_DEFAULT_OutputPacketSize = 50000
	CLIENTUDP_DEFAULT_OutputMaxSize    = 1300
	CLIENTUDP_DEFAULT_InputTimeoutMs   = 50
	CLIENTUDP_DEFAULT_InputPacketSize  = 50000
	CLIENTUDP_DEFAULT_InputMaxSize     = 1300
)

type UDPClient struct {
	DNS        string
	PacketSize int
	MaxSize    int

	OutputTimeoutMs uint32
	InputTimeoutMs  uint32

	Packets []*Packet

	InputStream  chan *Packet
	OutputStream chan *Packet

	currentInputBuffer int
	currentInputPacket int

	currentOutputBuffer int
	currentOutputPacket int

	conn  *net.UDPConn
	done  chan error
	queue chan int

	Stats StatsClient
}

func NewUDPClient() *UDPClient {
	return &UDPClient{
		done: make(chan error, 1),

		OutputTimeoutMs: CLIENTUDP_DEFAULT_OutputTimeoutMs,
		InputTimeoutMs:  CLIENTUDP_DEFAULT_InputTimeoutMs,
		PacketSize:      CLIENTUDP_DEFAULT_OutputPacketSize,
		MaxSize:         CLIENTUDP_DEFAULT_OutputMaxSize,
	}
}

func (c *UDPClient) init() error {
	c.Packets = make([]*Packet, c.PacketSize)
	c.queue = make(chan int, c.PacketSize)
	for i := 0; i < c.PacketSize; i++ {
		p := &Packet{Index: i}
		p.Data.Grow(c.MaxSize)
		c.Packets[i] = p
		c.queue <- i
	}

	c.InputStream = make(chan *Packet, c.PacketSize)
	c.OutputStream = make(chan *Packet, c.PacketSize)
	c.currentInputBuffer = 0
	c.currentInputPacket = 0

	return nil
}

func (c *UDPClient) Connect(DNS string) error {
	raddr, err := net.ResolveUDPAddr("udp", DNS)
	if err != nil {
		return err
	}

	if c.conn, err = net.DialUDP("udp", nil, raddr); err != nil {
		return err
	}

	if err = c.init(); err != nil {
		return err
	}

	// For reconnection
	c.DNS = DNS

	go c.receiveData()
	go c.sendData()

	return nil
}

func (s *UDPClient) NewPacket() (*Packet, error) {
	var p *Packet
	select {
	case index := <-s.queue:
		p = s.Packets[index]
		p.Data.Reset()
	case <-time.After(2 * time.Second):
		return nil, fmt.Errorf("No packet available")
	}
	return p, nil
}

func (s *UDPClient) ReleasePacket(p *Packet) {
	s.queue <- p.Index
}

func (c *UDPClient) Close() {
	c.done <- nil
}

func (c *UDPClient) sendData() {

	for {
		var p *Packet
		select {
		case p = <-c.OutputStream:
			deadline := time.Now().Add(time.Duration(c.OutputTimeoutMs) * time.Millisecond)
			c.conn.SetWriteDeadline(deadline)
			size, err := c.conn.Write(p.Data.Bytes())
			c.ReleasePacket(p)
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					time.Sleep(1 * time.Second)
					continue
				} else {
					c.done <- err
				}
			} else {
				c.Stats.PacketSentSize = c.Stats.PacketSentSize + uint64(size)
				c.Stats.PacketSentCount++
			}
		case <-c.done:
			return
		}
	}
	fmt.Println("sendData thread done")
}

func (c *UDPClient) receiveData() {
	rdbuf := make([]byte, c.MaxSize)
	for {
		var p *Packet
		var err error

		if p, err = c.NewPacket(); err != nil {
			// TODO error
			fmt.Println("Error de recuperation des packet")
			continue
		}

		deadline := time.Now().Add(time.Duration(c.InputTimeoutMs) * time.Millisecond)
		c.conn.SetReadDeadline(deadline)

		n, err := c.conn.Read(rdbuf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				c.ReleasePacket(p)
				continue
			} else {
				c.done <- err
				c.ReleasePacket(p)
				return
			}
		} else {
			c.Stats.PacketRecvSize = c.Stats.PacketRecvSize + uint64(n)
			c.Stats.PacketRecvCount++
		}

		p.Data.Write(rdbuf[:n])
		c.InputStream <- p
	}
	fmt.Println("receiveData thread done")
}

func (c *UDPClient) SendData(packet Serialize) error {
	var p *Packet
	var err error
	if p, err = c.NewPacket(); err != nil {
		return err
	}

	err = packet.Write(&p.Data)

	if err != nil {
		c.ReleasePacket(p)
		return err
	}

	// Let Go
	c.OutputStream <- p
	return nil
}

func (c *UDPClient) ReceiveData(packet Deserialize) error {
	var err error
	select {
	case p := <-c.InputStream:
		err = packet.Read(&p.Data)
		c.ReleasePacket(p)
	case err = <-c.done:
	}
	return err
}
