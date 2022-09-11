package hermes

import (
	"net"
	"net/netip"
	"time"
)

const (
	SERVERUDP_DEFAULT_InputTimeoutMs  = 10
	SERVERUDP_DEFAULT_InputBufferSize = 1024 * 1024
	SERVERUDP_DEFAULT_InputPacketSize = 50000
	SERVERUDP_DEFAULT_InputMaxSize    = 1300

	SERVERUDP_DEFAULT_MaxRemoteClient = 1500
)

type UDPServerConn interface {
	OnNewConnect() error
	OnReceiveData(p *Packet) error
}

type NewServerConnCallback func(addr netip.AddrPort) (UDPServerConn, error)

type UDPServer struct {
	DNS                   string
	InputTimeoutMs        uint32
	InputBufferSize       int
	InputPackets          []*Packet
	InputPacketSize       int
	InputMaxSize          int
	MaxRemoteClient       uint32
	NewConnectionCallback NewServerConnCallback

	currentInputPacket int

	conn          *net.UDPConn
	done          chan error
	udpConnection map[netip.AddrPort]UDPServerConn
}

func NewUDPServer(cb NewServerConnCallback) *UDPServer {
	return &UDPServer{
		done: make(chan error, 1),

		InputTimeoutMs:        SERVERUDP_DEFAULT_InputTimeoutMs,
		InputBufferSize:       SERVERUDP_DEFAULT_InputBufferSize,
		InputPacketSize:       SERVERUDP_DEFAULT_InputPacketSize,
		InputMaxSize:          SERVERUDP_DEFAULT_InputMaxSize,
		MaxRemoteClient:       SERVERUDP_DEFAULT_MaxRemoteClient,
		NewConnectionCallback: cb,
	}
}

func (s *UDPServer) init() error {
	s.udpConnection = make(map[string]UDPServerConn)
	s.InputPackets = make([]*Packet, s.InputPacketSize)
	for i := 0; i < s.InputPacketSize; i++ {
		s.InputPackets[i] = &Packet{}
		s.InputPackets[i].Data.Grow(s.InputMaxSize)
	}
	s.currentInputPacket = 0
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

func (s *UDPServer) Close() error {
	s.conn.Close()
	return nil
}

func (s *UDPServer) receiveData() {
	rdbuf := make([]byte, s.InputMaxSize)
	for {
		deadline := time.Now().Add(time.Duration(s.InputTimeoutMs) * time.Millisecond)
		s.conn.SetReadDeadline(deadline)

		p := s.InputPackets[s.currentInputPacket]
		p.Data.Reset()
		n, addr, err := s.conn.ReadFromUDP(rdbuf)

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				s.done <- err
				return
			}
		}

		p.Data.Write(rdbuf[:n])
		s.currentInputPacket = (s.currentInputPacket + 1) % s.InputPacketSize

		if v, ok := s.udpConnection[addr]; !ok {
			if conn, err := s.NewConnectionCallback(addr); err == nil {
				s.udpConnection[addr] = conn
				conn.OnNewConnect()
				conn.OnReceiveData(p)
			}
		} else {
			v.OnReceiveData(p)
		}

		_, err = s.conn.WriteToUDP(p.Data.Bytes(), addr)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else {
				s.done <- err
				return
			}
		}

	}
}
