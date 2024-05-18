package game

import (
	"context"
	pb "game_server/api/v1"
	"game_server/internal/database"
	"log"
)

type GameRunner struct {
	sessionId   int32
	db          *database.DbConnector
	ctx         context.Context
	ctxCancel   context.CancelFunc
	connections []pb.Api_StateStreamServer
}

func NewGameRunner(sessionId int32, db *database.DbConnector) *GameRunner {

	ctx, cxtCancel := context.WithCancel(context.Background())
	return &GameRunner{
		ctx:         ctx,
		db:          db,
		ctxCancel:   cxtCancel,
		connections: []pb.Api_StateStreamServer{},
	}
}

func (gr *GameRunner) addConnection(srv pb.Api_StateStreamServer) context.Context {
	gr.connections = append(gr.connections, srv)
	return gr.ctx
}

func (gr *GameRunner) startGameComputation() {
	go func() {
		for {
			session, err := gr.db.GetSession(gr.sessionId)
			if err != nil {
				log.Printf("game loop for session %d, get session from db error: %v", gr.sessionId, err)
				gr.ctxCancel()
			}

			state, err := computeState(session)
			if err != nil {
				log.Printf("game loop for session %d, compute state error: %v", gr.sessionId, err)
				gr.ctxCancel()
			}

			for _, srv := range gr.connections {
				if err := srv.Send(state); err != nil {
					log.Printf("game loop for session %d, send state error: %v", gr.sessionId, err)
					// gr.ctxCancel()
				}
			}
		}
	}()
}

func computeState(*pb.Session) (*pb.State, error) {
	return &pb.State{}, nil
}
