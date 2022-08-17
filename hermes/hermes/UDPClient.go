package hermes

import (
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
	DNS              string
	OutputTimeoutMs  uint32
	OutputPacketSize int
	OutputMaxSize    int
	InputTimeoutMs   uint32
	InputPacketSize  int
	InputMaxSize     int

	OutputPacket []*Packet
	InputPacket  []*Packet

	InputStream  chan *Packet
	OutputStream chan *Packet

	currentInputBuffer int
	currentInputPacket int

	currentOutputBuffer int
	currentOutputPacket int

	conn *net.UDPConn
	done chan error
}

func NewUDPClient() *UDPClient {
	return &UDPClient{
		done: make(chan error, 1),

		OutputTimeoutMs:  CLIENTUDP_DEFAULT_OutputTimeoutMs,
		OutputPacketSize: CLIENTUDP_DEFAULT_OutputPacketSize,
		OutputMaxSize:    CLIENTUDP_DEFAULT_OutputMaxSize,
		InputTimeoutMs:   CLIENTUDP_DEFAULT_InputTimeoutMs,
		InputPacketSize:  CLIENTUDP_DEFAULT_InputPacketSize,
		InputMaxSize:     CLIENTUDP_DEFAULT_InputMaxSize,
	}
}

func (c *UDPClient) init() error {
	c.InputPacket = make([]*Packet, c.InputPacketSize)
	for i := 0; i < c.InputPacketSize; i++ {
		p := &Packet{}
		p.Data.Grow(c.InputMaxSize)
		c.InputPacket[i] = p
	}
	c.OutputPacket = make([]*Packet, c.OutputPacketSize)
	for i := 0; i < c.OutputPacketSize; i++ {
		p := &Packet{}
		p.Data.Grow(c.OutputMaxSize)
		c.OutputPacket[i] = p
	}

	c.InputStream = make(chan *Packet, c.InputPacketSize)
	c.OutputStream = make(chan *Packet, c.OutputPacketSize)
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
			_, err := c.conn.Write(p.Data.Bytes())
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					//timeout = true
				} else {
					c.done <- err
				}
			}
		case <-c.done:
			return
		}
	}

}

func (c *UDPClient) receiveData() {
	rdbuf := make([]byte, c.InputMaxSize)
	for {

		deadline := time.Now().Add(time.Duration(c.InputTimeoutMs) * time.Millisecond)
		c.conn.SetReadDeadline(deadline)

		p := c.InputPacket[c.currentInputPacket]
		p.Data.Reset()

		n, err := c.conn.Read(rdbuf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				c.done <- err
				return
			}
		}

		p.Data.Write(rdbuf[:n])
		c.currentInputPacket = (c.currentInputPacket + 1) % c.InputPacketSize
		c.InputStream <- p
	}
}

func (c *UDPClient) SendData(packet Serialize) error {
	p := c.OutputPacket[c.currentOutputPacket]
	p.Data.Reset()
	c.currentOutputPacket = (c.currentOutputPacket + 1) % c.OutputPacketSize

	var err error
	err = packet.Write(&p.Data)

	if err != nil {
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
	case err = <-c.done:
	}
	return err
}
