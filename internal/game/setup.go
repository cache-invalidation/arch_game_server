package game

import "time"

const (
	maxPlayers         int   = 4
	sideLen            int32 = 30
	licenseAreaSideLen int32 = 6
	startMoney         int32 = 1000
	timeLimitMin       int   = 10
	licenseCost        int32 = 100
	passengerFuel      int   = 5
	onpPenalty         int32 = 500
)

// Transport reward
const (
	Reward_BUS   int = 5
	Reward_METRO int = 10
	Reward_TAXI  int = 0
	Reward_TRAM  int = 7
)

// Transport travel duration (per unit of distance)
const (
	Duration_BUS   time.Duration = time.Second
	Duration_METRO time.Duration = time.Second
	Duration_TAXI  time.Duration = time.Second
	Duration_TRAM  time.Duration = time.Second
)

// Transport const
const (
	Cost_BUS   int32 = 300
	Cost_METRO int32 = 3000
	Cost_TAXI  int32 = 1000
	Cost_TRAM  int32 = 1500
)
