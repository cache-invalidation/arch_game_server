package game

import pb "game_server/api/v1"

type GameRunner struct {
	connections []pb.Api_StateStreamServer
}

func NewGameRunner() *GameRunner {
	return &GameRunner{
		connections: []pb.Api_StateStreamServer{},
	}
}

func (gr *GameRunner) AddConnection(srv pb.Api_StateStreamServer) {}

func computeState(*pb.Session) (*pb.State, error) {
	return nil, nil
}
