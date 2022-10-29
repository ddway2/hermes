package hermes

import (
	"fmt"
	"net"
	"sync"
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
	DNS             string
	InputTimeoutMs  uint32
	OutputTimeoutMs uint32

	MaxRemoteClient       uint32
	NewConnectionCallback NewServerConnCallback

	poolPacket    sync.Pool
	MaxPacketSize int

	conn *net.UDPConn
	done chan error
	//udpConnection map[uint32]*UDPServerConnection
	udpClient sync.Map
}

func NewUDPServer(cb NewServerConnCallback) *UDPServer {
	return &UDPServer{
		done: make(chan error, 1),

		InputTimeoutMs:  SERVERUDP_DEFAULT_InputTimeoutMs,
		OutputTimeoutMs: SERVERUDP_DEFAULT_OutputTimeoutMs,

		poolPacket: sync.Pool{
			New: func() any {
				res := new(Packet)
				res.Data.Grow(SERVERUDP_DEFAULT_InputMaxSize)
				return res
			},
		},

		MaxRemoteClient:       SERVERUDP_DEFAULT_MaxRemoteClient,
		NewConnectionCallback: cb,
	}
}

func (s *UDPServer) init() error {
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

func (s *UDPServer) SendDataToClient(addr *net.UDPAddr, packet Serialize) error {
	var p *Packet
	var err error

	p = s.poolPacket.Get().(*Packet)
	if p == nil {
		return fmt.Errorf("Pool Get Error")
	}

	err = packet.Write(&p.Data)

	if err != nil {
		s.poolPacket.Put(p)
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
			s.poolPacket.Put(p)
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
	rdbuf := make([]byte, SERVERUDP_DEFAULT_InputMaxSize)
	fmt.Println("Start receiveData goroutine")
	var header PHeader
	for {
		var p *Packet
		var err error
		p = s.poolPacket.Get().(*Packet)
		if p == nil {
			continue
		}

		deadline := time.Now().Add(time.Duration(s.InputTimeoutMs) * time.Millisecond)
		s.conn.SetReadDeadline(deadline)

		n, addr, err := s.conn.ReadFromUDP(rdbuf)

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				s.poolPacket.Put(p)
				continue
			} else {
				s.done <- err
				s.poolPacket.Put(p)
				return
			}
		}

		p.Data.Write(rdbuf[:n])
		p.Addr = addr

		if err = header.Deserialize(&p.Data); err != nil {
			continue
		}

		s.dispatchData(header, p)

		// if v, ok := s.udpConnection[addr.String()]; !ok {
		// 	if conn, err := s.NewConnectionCallback(); err == nil {
		// 		s.udpConnection[addr.String()] = conn
		// 		conn.OnNewConnect(s, addr)
		// 		conn.OnReceiveData(s, p)
		// 	}
		// } else {
		// 	v.OnReceiveData(s, p)
		// }
	}
	fmt.Println("End receiveData goroutine")
}

func (s *UDPServer) dispatchData(h PHeader, p *Packet) error {
	switch h.Type {
	case PTYPE_INIT:

	default:
		return fmt.Errorf("Unknown Message type")
	}
	return nil
}
