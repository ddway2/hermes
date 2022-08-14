package hermes

import (
	"net"
	"time"
)

const (
	SERVERUDP_DEFAULT_InputTimeoutMs  = 500
	SERVERUDP_DEFAULT_InputBufferSize = 1024 * 1024
	SERVERUDP_DEFAULT_InputPacketSize = 2048
	SERVERUDP_DEFAULT_InputMaxSize    = 1300

	SERVERUDP_DEFAULT_MaxRemoteClient = 1500
)

type UDPServerConn interface {
	OnNewConnect() error
}

type UDPServer struct {
	DNS             string
	InputTimeoutMs  uint32
	InputBufferSize uint32
	InputBuffer     []byte
	InputPackets    []*Packet
	InputPacketSize uint32
	InputMaxSize    uint32
	MaxRemoteClient uint32

	currentInputBuffer uint32

	conn          *net.UDPConn
	done          chan error
	udpConnection map[string]UDPServerConn
}

func newUDPServer() *UDPServer {
	return &UDPServer{
		done: make(chan error, 1),

		InputTimeoutMs:  SERVERUDP_DEFAULT_InputTimeoutMs,
		InputBufferSize: SERVERUDP_DEFAULT_InputBufferSize,
		InputPacketSize: SERVERUDP_DEFAULT_InputPacketSize,
		InputMaxSize:    SERVERUDP_DEFAULT_InputMaxSize,
		MaxRemoteClient: SERVERUDP_DEFAULT_MaxRemoteClient,
	}
}

func (s *UDPServer) init() error {
	s.InputBuffer = make([]byte, s.InputBufferSize)
	s.udpConnection = make(map[string]UDPServerConn)
	s.InputPackets = make([]*Packet, s.InputPacketSize)
	s.currentInputBuffer = 0
	return nil
}

func (s *UDPServer) Listen(DNS string) error {

	saddr, err := net.ResolveUDPAddr("udp", DNS)
	if err != nil {
		return err
	}

	s.conn, err = net.ListenUDP("udp", saddr)
	if err != nil {
		return err
	}

	if err = s.init(); err != nil {
		return err
	}

	s.DNS = DNS

	go s.receiveData()

	return nil
}

func (s *UDPServer) receiveData() {
	for {
		deadline := time.Now().Add(time.Duration(s.InputTimeoutMs) * time.Millisecond)
		s.conn.SetReadDeadline(deadline)

		if s.currentInputBuffer+s.InputMaxSize >= s.InputBufferSize {
			s.currentInputBuffer = 0
		}

		n, addr, err := s.conn.ReadFromUDP(s.InputBuffer[s.currentInputBuffer:])
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				s.done <- err
				return
			}
		}

		var p Packet
		p.Size = n
		p.Ptr = s.currentInputBuffer

		_, err = s.conn.WriteToUDP(s.InputBuffer[p.Ptr:p.Size], addr)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				s.done <- err
				return
			}
		}

		s.currentInputBuffer = s.currentInputBuffer + uint32(n)

	}
}
