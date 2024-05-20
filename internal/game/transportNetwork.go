package game

import (
	"fmt"
	pb "game_server/api/v1"
	"math"
	"math/rand"
	"time"
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

func NewTransportNetwork() TransportNetwork {
	return TransportNetwork{
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

	if tn.blocks[p1] == nil {
		tn.blocks[p1] = []*Destination{}
	}
	if tn.blocks[p2] == nil {
		tn.blocks[p2] = []*Destination{}
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

type Path struct {
	Start Coords
	Hops  []*Destination
}

// Reward returns mapping of userid to reward they shall get
func (p Path) Reward() map[int32]int {
	rewards := map[int32]float64{}

	prev := p.Start

	for _, point := range p.Hops {
		xdiff := math.Abs(float64(prev.X) - float64(point.To.X))
		ydiff := math.Abs(float64(prev.Y) - float64(point.To.Y))
		distance := math.Sqrt(xdiff*xdiff + ydiff*ydiff)

		oldReward, ok := rewards[point.UserId]
		if !ok {
			oldReward = 0
		}

		rewards[point.UserId] = oldReward + distance*float64(transportReward(point.Transport))

		prev = point.To
	}

	rewardsInt := map[int32]int{}
	for k, v := range rewards {
		rewardsInt[k] = int(v)
	}

	return rewardsInt
}

// Duration returns amount of time needed to achieve the destination
func (p Path) Duration() time.Duration {
	var duration time.Duration
	duration = 0

	prev := p.Start

	for _, point := range p.Hops {
		xdiff := math.Abs(float64(prev.X) - float64(point.To.X))
		ydiff := math.Abs(float64(prev.Y) - float64(point.To.Y))
		distance := math.Sqrt(xdiff*xdiff + ydiff*ydiff)

		duration += time.Duration(distance * float64(transportDuration(point.Transport)))

		prev = point.To
	}

	return duration
}

func (tn *TransportNetwork) RandomPath(from Coords, fuel int) Path {
	hops := []*Destination{}

	// First hop could go literally anywhere
	possibleHops, ok := tn.blocks[from]
	if !ok || len(possibleHops) == 0 {
		return Path{
			Start: from,
			Hops:  hops,
		}
	}
	hop := possibleHops[rand.Intn(len(possibleHops))]
	hops = append(hops, hop)

	prev := from
	cur := hop.To

	for fuel > 0 {
		fuel--
		possibleHops, ok := tn.blocks[cur]

		// oh what a shame no way to go but back
		if !ok || len(possibleHops) <= 1 {
			// luckily theres always an escape :3
			break
		}

		// in other cases sample random direction to go to
		nextIdx := rand.Intn(len(possibleHops) - 1) // -1 cuz we aint gonna go back

		// fun indexing stuff where we skip the shit that's back
		// good thing we're in 2d dis loop is smol
		i := 0

		// in case the idx is zero and the loop ahead ain't gonna run
		if nextIdx == 0 && possibleHops[0].To == prev {
			i = 1
		}

		for nextIdx > 0 {
			i++
			// we ain't counting bitch that's going back
			if possibleHops[i].To != prev {
				nextIdx--
			}
		}

		// Could we overflow cause the edge to prev is last? HELL NAH
		// Cause nextIdx is sampled from [0, len - 1), we ain't ever
		// gonna get such index c:

		hop := possibleHops[i]
		prev = cur
		cur = hop.To
		hops = append(hops, hop)
	}

	return Path{
		Start: from,
		Hops:  hops,
	}
}
