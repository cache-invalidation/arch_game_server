package game

import (
	"fmt"
	pb "game_server/api/v1"
)

type Coords struct {
	X int32
	Y int32
}

type Destination struct {
	To        Coords
	Transport pb.Transport
	UserId    int32
}

type TransportNetwork struct {
	blocks map[Coords][]*Destination
}

func NewTransportNetwork() *TransportNetwork {
	return &TransportNetwork{
		blocks: map[Coords][]*Destination{},
	}
}

func (tn *TransportNetwork) ConnectBlocks(userId int32, p1 Coords, p2 Coords, transport pb.Transport) error {
	if tn.isPathExists(p1, p2) || tn.isPathExists(p2, p1) {
		return fmt.Errorf("path between %v and %v already exists", p1, p2)
	}

	path1to2 := &Destination{
		To:        p2,
		Transport: transport,
		UserId:    userId,
	}
	path2to1 := &Destination{
		To:        p1,
		Transport: transport,
		UserId:    userId,
	}

	tn.blocks[p1] = append(tn.blocks[p1], path1to2)
	tn.blocks[p2] = append(tn.blocks[p2], path2to1)

	return nil
}

func (tn *TransportNetwork) isPathExists(from Coords, to Coords) bool {
	for _, point := range tn.blocks[from] {
		if point.To == to {
			return true
		}
	}

	return false
}
