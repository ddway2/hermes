package hermes

import (
	"fmt"
	"net"
	"time"
)

const (
	SERVERUDP_DEFAULT_InputTimeoutMs  = 10
	SERVERUDP_DEFAULT_OutputTimeoutMs = 10
	SERVERUDP_DEFAULT_InputBufferSize = 1024 * 1024
	SERVERUDP_DEFAULT_InputPacketSize = 50000
	SERVERUDP_DEFAULT_InputMaxSize    = 1300

	SERVERUDP_DEFAULT_MaxRemoteClient = 1500
)

type UDPServerConn interface {
	OnNewConnect(s *UDPServer, addr *net.UDPAddr) error
	OnReceiveData(s *UDPServer, p *Packet) error
}

type NewServerConnCallback func() (UDPServerConn, error)

type UDPServer struct {
	DNS            string
	InputTimeoutMs uint32
	BufferSize     int
	Packets        []*Packet
	PacketSize     int
	MaxSize        int

	OutputTimeoutMs uint32

	MaxRemoteClient       uint32
	NewConnectionCallback NewServerConnCallback

	OutputStream chan *Packet

	currentInputPacket int

	conn          *net.UDPConn
	done          chan error
	udpConnection map[string]UDPServerConn

	queue chan int
}

func NewUDPServer(cb NewServerConnCallback) *UDPServer {
	return &UDPServer{
		done: make(chan error, 1),

		InputTimeoutMs:        SERVERUDP_DEFAULT_InputTimeoutMs,
		OutputTimeoutMs:       SERVERUDP_DEFAULT_OutputTimeoutMs,
		BufferSize:            SERVERUDP_DEFAULT_InputBufferSize,
		PacketSize:            SERVERUDP_DEFAULT_InputPacketSize,
		MaxSize:               SERVERUDP_DEFAULT_InputMaxSize,
		MaxRemoteClient:       SERVERUDP_DEFAULT_MaxRemoteClient,
		NewConnectionCallback: cb,
	}
}

func (s *UDPServer) init() error {
	s.udpConnection = make(map[string]UDPServerConn)
	s.Packets = make([]*Packet, s.PacketSize)
	s.OutputStream = make(chan *Packet, s.PacketSize)
	s.queue = make(chan int, s.PacketSize)
	for i := 0; i < s.PacketSize; i++ {
		s.Packets[i] = &Packet{Index: i}
		s.Packets[i].Data.Grow(s.MaxSize)
		s.queue <- i
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
	go s.sendData()

	return nil
}

func (s *UDPServer) Close() error {
	s.conn.Close()
	return nil
}

func (s *UDPServer) NewPacket() (*Packet, error) {
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

func (s *UDPServer) ReleasePacket(p *Packet) {
	s.queue <- p.Index
}

func (s *UDPServer) SendDataToClient(addr *net.UDPAddr, packet Serialize) error {
	var p *Packet
	var err error
	if p, err = s.NewPacket(); err != nil {
		return err
	}

	err = packet.Write(&p.Data)

	if err != nil {
		s.ReleasePacket(p)
		return err
	}

	// Let Go
	s.OutputStream <- p
	return nil
}

func (s *UDPServer) SendRawDataToClient(addr *net.UDPAddr, p *Packet) error {
	// Let Go
	s.OutputStream <- p
	return nil
}

func (s *UDPServer) sendData() {
	fmt.Println("Start sendData goroutine")
	for {
		var p *Packet
		select {
		case p = <-s.OutputStream:
			deadline := time.Now().Add(time.Duration(s.OutputTimeoutMs) * time.Millisecond)
			s.conn.SetWriteDeadline(deadline)

			_, err := s.conn.WriteToUDP(p.Data.Bytes(), p.Addr)
			s.ReleasePacket(p)
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				} else {
					s.done <- err
				}
			}
		}
	}
	fmt.Println("End sendData goroutine")
}

func (s *UDPServer) receiveData() {
	rdbuf := make([]byte, s.MaxSize)
	fmt.Println("Start receiveData goroutine")
	for {
		var p *Packet
		var err error
		if p, err = s.NewPacket(); err != nil {
			continue
		}

		deadline := time.Now().Add(time.Duration(s.InputTimeoutMs) * time.Millisecond)
		s.conn.SetReadDeadline(deadline)

		n, addr, err := s.conn.ReadFromUDP(rdbuf)

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				s.ReleasePacket(p)
				continue
			} else {
				s.done <- err
				s.ReleasePacket(p)
				return
			}
		}

		p.Data.Write(rdbuf[:n])
		p.Addr = addr

		if v, ok := s.udpConnection[addr.String()]; !ok {
			if conn, err := s.NewConnectionCallback(); err == nil {
				s.udpConnection[addr.String()] = conn
				conn.OnNewConnect(s, addr)
				conn.OnReceiveData(s, p)
			}
		} else {
			v.OnReceiveData(s, p)
		}
	}
	fmt.Println("End receiveData goroutine")
}
