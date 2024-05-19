package game

import (
	pb "game_server/api/v1"
	"math/rand"
)

func generateMap() []*pb.Block {
	gameMap := []*pb.Block{}
	for y := int32(0); y < sideLen; y++ {
		for x := int32(0); x < sideLen; x++ {
			block := &pb.Block{
				Position:   &pb.Coordintates{X: x, Y: y},
				Type:       pb.BlockType(rand.Int31n(int32(len(pb.BlockType_name) - 1))),
				Capacity:   rand.Int31n(5) + 1,
				Connectors: []*pb.Connector{},
			}

			gameMap = append(gameMap, block)
		}
	}

	return gameMap
}
