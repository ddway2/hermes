package hermes

import (
	"net"
	"time"
)

const (
	CLIENTUDP_DEFAULT_OutputTimeoutMs  = 500
	CLIENTUDP_DEFAULT_OutputBufferSize = 1024 * 1024
	CLIENTUDP_DEFAULT_OutputPacketSize = 500
	CLIENTUDP_DEFAULT_OutputMaxSize    = 1300
	CLIENTUDP_DEFAULT_InputTimeoutMs   = 500
	CLIENTUDP_DEFAULT_InputBufferSize  = 1024 * 1024
	CLIENTUDP_DEFAULT_InputPacketSize  = 500
	CLIENTUDP_DEFAULT_InputMaxSize     = 1300
)

type UDPClient struct {
	DNS              string
	OutputTimeoutMs  uint32
	OutputPacketSize uint32
	OutputMaxSize    uint32
	InputTimeoutMs   uint32
	InputPacketSize  uint32
	InputMaxSize     uint32

	OutputPacket []*Packet
	InputPacket  []*Packet

	InputStream  chan *Packet
	OutputStream chan *Packet

	currentInputBuffer uint32
	currentInputPacket uint32

	currentOutputBuffer uint32
	currentOutputPacket uint32

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
	for i := 0; i < int(c.InputPacketSize); i++ {
		p := &Packet{}
		p.Data.Grow(c.InputMaxSize)
		c.InputPacket = append(c.InputPacket, p)
	}
	c.OutputPacket = make([]*Packet, c.OutputPacketSize)
	for i := 0; i < int(c.OutputPacketSize); i++ {
		p := &Packet{}
		p.Data.Grow(c.OutputMaxSize)
		c.OutputPacket = append(c.OutputPacket, p)
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

	for {

		deadline := time.Now().Add(time.Duration(c.InputTimeoutMs) * time.Millisecond)
		c.conn.SetReadDeadline(deadline)

		p := c.InputPacket[c.currentInputPacket]
		p.Data.Reset()

		_, err := c.conn.Read(p.data)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				c.done <- err
				return
			}
		}

		c.currentInputPacket = (c.currentInputPacket + 1) % c.InputPacketSize
		c.InputStream <- p
	}
}

func (c *UDPClient) SendData(packet Serialize) error {
	p := c.OutputPacket[c.currentOutputPacket]
	p.Data.Reset()
	c.currentOutputPacket = (c.currentOutputPacket + 1) % c.OutputPacketSize

	var err error
	err = packet.Write(p.Data)

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
		err = packet.Read(p.Data)
	case err = <-c.done:
	}
	return err
}
