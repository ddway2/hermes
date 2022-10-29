package hermes

import "net"

type State uint16

const (
	STATE_CLIENT_INIT State = iota
	STATE_CLIENT_CHALLENGE_SENT
	STATE_CLIENT_ACTIVE
	STATE_CLIENT_DISC_REQ
	STATE_CLIENT_INACTIVE
	STATE_CLIENT_ERROR
)

type UDPServerConnection struct {
	GameId uint32
	Id     uint32

	state State
	addr  *net.UDPAddr
	srv   *net.UDPConn
}

func NewUDPServerConnection(addr *net.UDPAddr, srv *net.UDPConn) *UDPServerConnection {
	return &UDPServerConnection{
		state: STATE_CLIENT_INIT,
		addr:  addr,
		srv:   srv,
	}
}

func (c *UDPServerConnection) ChallengeSent() {
	c.state = STATE_CLIENT_CHALLENGE_SENT
}
