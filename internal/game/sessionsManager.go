package game

import (
	"fmt"
	pb "game_server/api/v1"
	"game_server/internal/database"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SessionsManager struct {
	db                   *database.DbConnector
	pendingSessions      []int32
	gameRuners           map[int32]*GameRunner //key: sessionId
	pendingSessionsMutex sync.Mutex
	transportMutex       sync.Mutex
	moneyMutex           *sync.Mutex
}

func NewSessionsManager(db *database.DbConnector) *SessionsManager {
	return &SessionsManager{
		db:              db,
		pendingSessions: []int32{},
		gameRuners:      map[int32]*GameRunner{},
		moneyMutex:      &sync.Mutex{},
	}
}

func (sm *SessionsManager) FindSessionForUser(userId int32) (*pb.Session, error) {
	pendingSession, err := sm.getPendingSession()
	if err != nil {
		return nil, err
	}

	if pendingSession == nil {
		session := createSession()
		user := createUser(userId, 0)
		session.Users = append(session.Users, user)

		if maxPlayers == 1 {
			if err := sm.db.AddSession(session); err != nil {
				return nil, err
			}

			sm.startSesison(session)

			return session, sm.db.UpdateSession(session)
		}

		sm.addPendingSession(session.Id)

		if err := sm.db.AddSession(session); err != nil {
			return nil, err
		}

		return session, nil
	}

	user := createUser(userId, len(pendingSession.Users))
	pendingSession.Users = append(pendingSession.Users, user)

	if len(pendingSession.Users) == maxPlayers {
		sm.startSesison(pendingSession)
	}

	if err := sm.db.UpdateSession(pendingSession); err != nil {
		return nil, err
	}

	return pendingSession, nil
}

func (sm *SessionsManager) startSesison(session *pb.Session) {
	session.Status = pb.SessionStatus_ACTIVE
	session.StartTime = timestamppb.New(time.Now().Add(time.Second * 30))

	gameRunner := NewGameRunner(session.Id, sm.db, session, sm.moneyMutex)
	gameRunner.startGameComputation()

	sm.gameRuners[session.Id] = gameRunner
}

func createSession() *pb.Session {
	session := &pb.Session{
		Id:        rand.Int31(),
		Users:     []*pb.User{},
		Map:       generateMap(),
		TimeLimit: durationpb.New(time.Duration(TimeLimitMin) * time.Minute),
		Status:    pb.SessionStatus_WAITING,
	}
	return session
}

// Start positions;
// ┌--------┐
// |   0    |
// |3      1|
// |   2    |
// └--------┘
func createUser(userId int32, startPos int) *pb.User {
	license := []*pb.Coordintates{}
	start := sideLen/2 - licenseAreaSideLen/2
	end := sideLen/2 + licenseAreaSideLen/2

	var xStart int32 = 0
	var xEnd int32 = 0
	var yStart int32 = 0
	var yEnd int32 = 0

	switch startPos {
	case 0:
		xStart = start
		xEnd = end
		yEnd = licenseAreaSideLen
	case 1:
		xStart = sideLen - licenseAreaSideLen
		xEnd = sideLen
		yStart = start
		yEnd = end
	case 2:
		xStart = start
		xEnd = end
		yStart = sideLen - licenseAreaSideLen
		yEnd = sideLen
	case 3:
		xEnd = licenseAreaSideLen
		yStart = start
		yEnd = end
	default:
		log.Panicf("unexpected starting pos %d during user creation", startPos)
	}

	for y := yStart; y < yEnd; y++ {
		for x := xStart; x < xEnd; x++ {
			license = append(license, &pb.Coordintates{X: x, Y: y})
		}
	}

	user := pb.User{
		Id:      userId,
		Name:    strconv.Itoa(int(userId)),
		Money:   startMoney,
		License: license,
	}
	return &user
}

func (sm *SessionsManager) addPendingSession(sessionId int32) {
	sm.pendingSessionsMutex.Lock()
	defer sm.pendingSessionsMutex.Unlock()

	sm.pendingSessions = append(sm.pendingSessions, sessionId)
}

func (sm *SessionsManager) getPendingSession() (*pb.Session, error) {
	sm.pendingSessionsMutex.Lock()
	defer sm.pendingSessionsMutex.Unlock()

	if len(sm.pendingSessions) == 0 {
		return nil, nil
	}

	id := sm.pendingSessions[0]
	session, err := sm.db.GetSession(id)
	if err != nil {
		sm.pendingSessions = sm.pendingSessions[1:]
		return nil, err
	}

	if len(session.Users) >= maxPlayers-1 {
		sm.pendingSessions = sm.pendingSessions[1:]
	}

	return session, nil
}

func (sm *SessionsManager) AddTransport(userId int32, from *pb.Coordintates, to *pb.Coordintates, transport pb.Transport) error {
	sm.moneyMutex.Lock()
	sm.transportMutex.Lock()
	defer sm.moneyMutex.Unlock()
	defer sm.transportMutex.Unlock()

	session, err := sm.db.GetAliveSessionByUser(userId)
	if err != nil {
		return err
	}
	log.Printf("found session %d for user %d\n", session.Id, userId)

	fromBlock := session.Map[from.Y*sideLen+from.X]
	toBlock := session.Map[to.Y*sideLen+to.X]

	if len(fromBlock.Connectors) == int(fromBlock.Capacity) || len(toBlock.Connectors) == int(toBlock.Capacity) {
		return fmt.Errorf("connector capacity exceeded")
	}

	fromBlock.Connectors = append(fromBlock.Connectors, &pb.Connector{UserId: userId, Transport: transport, Destination: to})
	toBlock.Connectors = append(toBlock.Connectors, &pb.Connector{UserId: userId, Transport: transport, Destination: from})

	gameRunner, ok := sm.gameRuners[session.Id]
	if !ok {
		return fmt.Errorf("no game runner for session %d with user: %d", session.Id, userId)
	}

	if err := gameRunner.extendNetwork(userId, from, to, transport); err != nil {
		return err
	}

	for _, user := range session.Users {
		if user.Id == userId {
			money := user.Money - transportCost(transport)
			if money < 0 {
				return fmt.Errorf("user %d does not have enough money", userId)
			}
			user.Money = money
			break
		}
	}

	return sm.db.UpdateSession(session)
}

func (sm *SessionsManager) ExtendLicense(userId int32, blocks []*pb.Coordintates) error {
	sm.moneyMutex.Lock()
	defer sm.moneyMutex.Unlock()

	session, err := sm.db.GetAliveSessionByUser(userId)
	if err != nil {
		return err
	}
	log.Printf("found session %d for user %d\n", session.Id, userId)

	for _, user := range session.Users {
		if user.Id == userId {
			money := user.Money - LicenseCost*int32(len(blocks))
			if money < 0 {
				return fmt.Errorf("user %d does not have enough money", userId)
			}

			user.License = append(user.License, blocks...)
			user.Money = money
			break
		}
	}

	return sm.db.UpdateSession(session)
}

func (sm *SessionsManager) StreamState(sessionId, userId int32, srv pb.Api_StateStreamServer) error {
	gameRunner, ok := sm.gameRuners[sessionId]
	if !ok {
		return fmt.Errorf("no game runner for session %d with user: %d", sessionId, userId)
	}

	ctx := gameRunner.addConnection(srv)
	<-ctx.Done()

	return nil
}
