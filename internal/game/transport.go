package game

import (
	pb "game_server/api/v1"
	"time"
)

func transportReward(t pb.Transport) int {
	switch t {
	case pb.Transport_BUS:
		return Reward_BUS
	case pb.Transport_METRO:
		return Reward_METRO
	case pb.Transport_TAXI:
		return Reward_TAXI
	case pb.Transport_TRAM:
		return Reward_TRAM
	default:
		return 0
	}
}

func transportDuration(t pb.Transport) time.Duration {
	switch t {
	case pb.Transport_BUS:
		return Duration_BUS
	case pb.Transport_METRO:
		return Duration_METRO
	case pb.Transport_TAXI:
		return Duration_TAXI
	case pb.Transport_TRAM:
		return Duration_TRAM
	default:
		return 0
	}
}

func transportCost(t pb.Transport) int32 {
	switch t {
	case pb.Transport_BUS:
		return Cost_BUS
	case pb.Transport_METRO:
		return Cost_METRO
	case pb.Transport_TAXI:
		return Cost_TAXI
	case pb.Transport_TRAM:
		return Cost_TRAM
	default:
		return 0
	}
}
