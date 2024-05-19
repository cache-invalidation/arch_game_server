package game

import (
	"container/heap"
	"context"
	pb "game_server/api/v1"
	"game_server/internal/database"
	"log"
	"reflect"
	"sync"
	"time"
)

type GameRunner struct {
	sessionId        int32
	db               *database.DbConnector
	ctx              context.Context
	ctxCancel        context.CancelFunc
	connections      []pb.Api_StateStreamServer
	network          TransportNetwork
	networkMutex     sync.Mutex
	rewardQueue      *RewardQueue
	lastSessionState *pb.Session
}

func NewGameRunner(sessionId int32, db *database.DbConnector, initSessionState *pb.Session) *GameRunner {
	ctx, cxtCancel := context.WithCancel(context.Background())
	rewatdQueue := &RewardQueue{}
	heap.Init(rewatdQueue)

	return &GameRunner{
		ctx:              ctx,
		db:               db,
		ctxCancel:        cxtCancel,
		connections:      []pb.Api_StateStreamServer{},
		network:          TransportNetwork{},
		rewardQueue:      rewatdQueue,
		lastSessionState: initSessionState,
	}
}

func (gr *GameRunner) addConnection(srv pb.Api_StateStreamServer) context.Context {
	gr.connections = append(gr.connections, srv)
	return gr.ctx
}

func (gr *GameRunner) startGameComputation() {
	go func() {
		var session *pb.Session
		var err error
		for {
			session, err = gr.db.GetSession(gr.sessionId)
			if err != nil {
				log.Printf("game loop for session %d, get session from db error: %v", gr.sessionId, err)
				gr.ctxCancel()
			}

			if session.StartTime.AsTime().Add(time.Duration(timeLimitMin) * time.Minute).After(time.Now()) {
				break
			}

			state, err := gr.computeState(session)
			if err != nil {
				log.Printf("game loop for session %d, compute state error: %v", gr.sessionId, err)
				session.Status = pb.SessionStatus_FINISHED
				gr.ctxCancel()
			}

			if err := gr.db.UpdateSession(session); err != nil {
				log.Printf("game loop for session %d, update session in db error: %v", gr.sessionId, err)
				gr.ctxCancel()
			}

			for _, srv := range gr.connections {
				if err := srv.Send(state); err != nil {
					log.Printf("game loop for session %d, send state error: %v", gr.sessionId, err)
					// gr.ctxCancel()
				}
			}

			time.Sleep(200)
		}

		session.Status = pb.SessionStatus_FINISHED
		if err := gr.db.UpdateSession(session); err != nil {
			log.Printf("game end for session %d, update session in db error: %v", gr.sessionId, err)
		}

		gr.ctxCancel()
	}()
}

func (gr *GameRunner) extendNetwork(userId int32, p1 *pb.Coordintates, p2 *pb.Coordintates, transport pb.Transport) error {
	gr.networkMutex.Lock()
	defer gr.networkMutex.Unlock()

	coords1 := Coords{
		X: p1.X,
		Y: p1.Y,
	}
	coords2 := Coords{
		X: p2.X,
		Y: p2.Y,
	}

	return gr.network.ConnectBlocks(userId, coords1, coords2, transport)
}

func (gr *GameRunner) computeState(session *pb.Session) (*pb.State, error) {
	changedBlocks := []*pb.Block{}
	for i := range session.Map {
		if !reflect.DeepEqual(session.Map[i], gr.lastSessionState.Map[i]) {
			changedBlocks = append(changedBlocks, session.Map[i])
		}
	}

	state := &pb.State{
		Users:                session.Users,
		NewEvents:            []*pb.Event{},
		ChangedBlocks:        changedBlocks,
		OutNetworkPassengers: []*pb.OutNetworkPassenger{},
	}

	gr.lastSessionState = session

	return state, nil
}
