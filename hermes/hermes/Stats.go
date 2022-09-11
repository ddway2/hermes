package hermes

import "fmt"

type StatsClient struct {
	PacketRecvCount uint64
	PacketRecvSize  uint64
	PacketSentCount uint64
	PacketSentSize  uint64
}

func (s *StatsClient) Reset() {
	s.PacketRecvCount = 0
	s.PacketRecvSize = 0
	s.PacketSentCount = 0
	s.PacketSentSize = 0
}

func (s *StatsClient) String() string {
	return fmt.Sprintf("Packet Receive count=%v size=%v|Packet sent count=%v size=%v",
		s.PacketRecvCount,
		s.PacketRecvSize,
		s.PacketSentCount,
		s.PacketSentSize,
	)
}

func (s *StatsClient) BiterateRecv(deltaTimeSec float64) float64 {
	if deltaTimeSec > 0.0 {
		return float64(s.PacketRecvSize) / deltaTimeSec
	}
	return 0.0
}

func (s *StatsClient) PacketsRecv(deltaTimeSec float64) float64 {
	if deltaTimeSec > 0.0 {
		return float64(s.PacketRecvCount) / deltaTimeSec
	}
	return 0.0
}

func (s *StatsClient) BiterateSent(deltaTimeSec float64) float64 {
	if deltaTimeSec > 0.0 {
		return float64(s.PacketSentSize) / deltaTimeSec
	}
	return 0.0
}

func (s *StatsClient) PacketsSent(deltaTimeSec float64) float64 {
	if deltaTimeSec > 0.0 {
		return float64(s.PacketSentCount) / deltaTimeSec
	}
	return 0.0
}
