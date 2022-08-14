package hermes

import (
	"io"
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
	OutputBufferSize uint32
	OutputBuffer     []byte
	OutputPacketSize uint32
	OutputMaxSize    uint32
	InputTimeoutMs   uint32
	InputBufferSize  uint32
	InputBuffer      []byte
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
		OutputBufferSize: CLIENTUDP_DEFAULT_OutputBufferSize,
		OutputPacketSize: CLIENTUDP_DEFAULT_OutputPacketSize,
		OutputMaxSize:    CLIENTUDP_DEFAULT_OutputMaxSize,
		InputTimeoutMs:   CLIENTUDP_DEFAULT_InputTimeoutMs,
		InputBufferSize:  CLIENTUDP_DEFAULT_InputBufferSize,
		InputPacketSize:  CLIENTUDP_DEFAULT_InputPacketSize,
		InputMaxSize:     CLIENTUDP_DEFAULT_InputMaxSize,
	}
}

func (c *UDPClient) init() error {
	c.InputBuffer = make([]byte, c.InputBufferSize)
	c.InputPacket = make([]*Packet, c.InputPacketSize)
	c.OutputBuffer = make([]byte, c.OutputBufferSize)
	c.OutputPacket = make([]*Packet, c.OutputPacketSize)

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
			_, err := c.conn.Write(c.OutputBuffer[p.Ptr : p.Ptr+uint32(p.Size)])
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

		if c.currentInputBuffer+c.InputMaxSize >= c.InputBufferSize {
			c.currentInputBuffer = 0
		}

		n, err := c.conn.Read(c.InputBuffer[c.currentInputBuffer:])
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				c.done <- err
				return
			}
		}

		p := c.InputPacket[c.currentInputPacket]
		c.currentInputPacket = (c.currentInputPacket + 1) % c.InputPacketSize
		c.currentInputBuffer = c.currentInputBuffer + uint32(n)
		p.Ptr = c.currentInputBuffer
		p.Size = n
		c.InputStream <- p
	}
}

func (c *UDPClient) SendData(packet io.Writer) error {
	p := c.OutputPacket[c.currentOutputPacket]
	c.currentOutputPacket = (c.currentOutputPacket + 1) % c.OutputPacketSize

	if c.currentInputBuffer+c.OutputMaxSize >= c.OutputBufferSize {
		c.currentInputBuffer = 0
	}

	var err error
	p.Size, err = packet.Write(c.OutputBuffer[c.currentOutputBuffer:])
	if err != nil {
		return err
	}
	p.Ptr = c.currentOutputBuffer
	c.currentOutputBuffer = c.currentOutputBuffer + uint32(p.Size)

	// Let Go
	c.OutputStream <- p
	return nil
}

func (c *UDPClient) ReceiveData(packet io.Reader) error {
	var err error
	select {
	case p := <-c.InputStream:
		_, err = packet.Read(c.InputBuffer[p.Ptr:p.Size])
	case err = <-c.done:
	}
	return err
}
