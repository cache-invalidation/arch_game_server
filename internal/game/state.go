package game

import (
	"container/heap"
	"context"
	pb "game_server/api/v1"
	"game_server/internal/database"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
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
	onps             []*pb.OutNetworkPassenger
	lastSessionState *pb.Session
	moneyMutex       *sync.Mutex
}

func NewGameRunner(sessionId int32, db *database.DbConnector, initSessionState *pb.Session, moneyMutex *sync.Mutex) *GameRunner {
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
		onps:             []*pb.OutNetworkPassenger{},
		lastSessionState: initSessionState,
		moneyMutex:       moneyMutex,
	}
}

func (gr *GameRunner) addConnection(srv pb.Api_StateStreamServer) context.Context {
	gr.connections = append(gr.connections, srv)
	return gr.ctx
}

// kF computes value for k (+ jitter) based on game progression
func kF(alpha float64) int {
	// Idea: approx every 20 ticks in the beginning, every 5 ticks at the end
	return 16 - int(alpha*15) + rand.Intn(4)
}

// nF computes value for n (+ jitter) based on game progression
func nF(alpha float64) int {
	// Idea: at most 1 at the beginning, at most 25 at the end
	// Linearly adjust n
	return 1 + rand.Intn(int(alpha*25))
}

func (gr *GameRunner) startGameComputation() {
	go func() {
		// k is a counter managing generation of travellers. Each time it
		// reaches zero, a handful of travellers is produced
		k := 1

		var session *pb.Session
		var err error
		for {
			gr.moneyMutex.Lock()
			k--
			session, err = gr.db.GetSession(gr.sessionId)
			if err != nil {
				log.Printf("game loop for session %d, get session from db error: %v", gr.sessionId, err)
				gr.ctxCancel()
			}

			if session.StartTime.AsTime().Add(time.Duration(timeLimitMin) * time.Minute).Before(time.Now()) {
				break
			}

			to_spawn := 0
			if k == 0 {
				alpha := time.Now().Sub(session.StartTime.AsTime()).Minutes() / float64(timeLimitMin)

				// set the counter to new value
				k = kF(alpha)
				// select the number of travellers spawned
				to_spawn = nF(alpha)
			}

			state, err := gr.computeState(session, to_spawn)
			if err != nil {
				log.Printf("game loop for session %d, compute state error: %v", gr.sessionId, err)
				session.Status = pb.SessionStatus_FINISHED
				gr.ctxCancel()
			}

			if err := gr.db.UpdateSession(session); err != nil {
				log.Printf("game loop for session %d, update session in db error: %v", gr.sessionId, err)
				gr.ctxCancel()
			}

			gr.moneyMutex.Unlock()

			for _, srv := range gr.connections {
				if err := srv.Send(state); err != nil {
					log.Printf("game loop for session %d, send state error: %v", gr.sessionId, err)
					// gr.ctxCancel()
				}
			}

			time.Sleep(300)
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

func (gr *GameRunner) generateTravellers(n int) []Path {
	gr.networkMutex.Lock()
	defer gr.networkMutex.Unlock()

	// Let's filter out the starting edges that have
	starts := []Coords{}

	paths := []Path{}

	for s := range gr.network.blocks {
		starts = append(starts, s)
	}

	if len(starts) > 0 {
		for i := 0; i < n; i++ {
			start := starts[rand.Intn(len(starts))]
			path := gr.network.RandomPath(start, passengerFuel)
			if len(path.Hops) > 0 {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

// ONP means OutNetworkPassenger
func (gr *GameRunner) generateONP(session *pb.Session) []*pb.OutNetworkPassenger {
	nowTime := time.Now()
	points := []*pb.Coordintates{}

	for _, user := range session.Users {
		for _, block := range user.License {
			if _, ok := gr.network.blocks[Coords{X: block.X, Y: block.Y}]; !ok {
				points = append(points, block)
			}
		}
	}

	newOnps := []*pb.OutNetworkPassenger{}

	passengersNum := 3 + rand.Int31n(1)

	for i := range rand.Perm(len(points))[:passengersNum] {
		ttl := time.Duration(30+rand.Int31n(60)) * time.Second

		onp := &pb.OutNetworkPassenger{
			Position:   points[i],
			TimeToBurn: timestamppb.New(nowTime.Add(ttl)),
		}

		newOnps = append(newOnps, onp)
		gr.onps = append(gr.onps, onp)
	}

	return newOnps
}

func (gr *GameRunner) onpsBurnOrGetSendToRoad(session *pb.Session) []*pb.OutNetworkPassenger {
	currentTime := time.Now()
	sendToRoad := []*pb.OutNetworkPassenger{}

	for i, onp := range gr.onps {
		if _, ok := gr.network.blocks[Coords{X: onp.Position.X, Y: onp.Position.Y}]; ok {
			sendToRoad = append(sendToRoad, onp)
			gr.onps = append(gr.onps[:i], gr.onps[i+1:]...)
			continue
		}
		if onp.TimeToBurn.AsTime().Before(currentTime) {
			for _, user := range session.Users {
				for _, block := range user.License {
					if onp.Position.X == block.X && onp.Position.Y == block.Y {
						money := user.Money - onpPenalty
						if money < 0 {
							money = 0
						}

						user.Money = money
						break
					}
				}
			}

			gr.onps = append(gr.onps[:i], gr.onps[i+1:]...)
		}
	}

	return sendToRoad
}

func (gr *GameRunner) computeState(session *pb.Session, to_spawn int) (*pb.State, error) {
	gr.rewardsAccrual(session)
	sendToRoadOnps := gr.onpsBurnOrGetSendToRoad(session)

	changedBlocks := []*pb.Block{}
	for i := range session.Map {
		if !reflect.DeepEqual(session.Map[i], gr.lastSessionState.Map[i]) {
			changedBlocks = append(changedBlocks, session.Map[i])
		}
	}

	now := time.Now()
	paths := gr.generateTravellers(to_spawn)
	for _, onp := range sendToRoadOnps {
		paths = append(paths, gr.network.RandomPath(Coords{X: onp.Position.X, Y: onp.Position.Y}, passengerFuel))
	}

	newOnps := gr.generateONP(session)

	// No we shall reward generously the completers of the path
	for _, path := range paths {
		rewards := path.Reward()

		for user, money := range rewards {
			heap.Push(gr.rewardQueue, &Reward{
				userId:         user,
				money:          int32(money),
				activationTime: now,
			})
		}
	}

	// And we shall send the generated paths to the client
	tracks := []*pb.Path{}
	for _, path := range paths {
		coords := []*pb.Coordintates{{
			X: path.Start.X,
			Y: path.Start.Y,
		}}

		for _, hop := range path.Hops {
			coords = append(coords, &pb.Coordintates{
				X: hop.To.X,
				Y: hop.To.Y,
			})
		}

		tracks = append(tracks, &pb.Path{
			Points: coords,
		})
	}

	state := &pb.State{
		Users:                session.Users,
		NewEvents:            []*pb.Event{},
		ChangedBlocks:        changedBlocks,
		Tracks:               tracks,
		OutNetworkPassengers: newOnps,
	}

	gr.lastSessionState = session

	return state, nil
}

func (gr *GameRunner) rewardsAccrual(session *pb.Session) {
	currentTime := time.Now()

	for gr.rewardQueue.Len() > 0 && gr.rewardQueue.Top().activationTime.After(currentTime) {
		reward := heap.Pop(gr.rewardQueue).(*Reward)

		for _, user := range session.Users {
			if user.Id == reward.userId {
				user.Money += reward.money
				break
			}
		}
	}
}
