package game

import (
	pb "game_server/api/v1"
	"game_server/internal/database"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type SessionsManager struct {
	db                   *database.DbConnector
	maxPlayers           int
	pendingSessions      []int32
	pendingSessionsMutex sync.Mutex
}

func NewSessionsManager(db *database.DbConnector) *SessionsManager {
	return &SessionsManager{
		db:         db,
		maxPlayers: 4,
	}
}

func (sm *SessionsManager) FindSessionForUser(userId int32) (*pb.Session, error) {
	user := createUser(userId)

	pendingSession, err := sm.getPendingSession()
	if err != nil {
		return nil, err
	}

	if pendingSession == nil {
		session := createSession()

		session.Users = append(session.Users, user)

		if err := sm.db.AddSession(session); err != nil {
			return nil, err
		}

		sm.addPendingSession(session.Id)

		return session, nil
	}

	pendingSession.Users = append(pendingSession.Users, user)

	if len(pendingSession.Users) == sm.maxPlayers {
		pendingSession.Status = pb.SessionStatus_ACTIVE
		pendingSession.StartTime = timestamppb.New(time.Now().Add(time.Minute))
	}

	if err := sm.db.UpdateSession(pendingSession); err != nil {
		return nil, err
	}

	return pendingSession, nil
}

func createSession() *pb.Session {
	return nil
}

func createUser(int32) *pb.User {
	return nil
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

	if len(session.Users) >= sm.maxPlayers-1 {
		sm.pendingSessions = sm.pendingSessions[1:]
	}

	return session, nil
}

func (sm *SessionsManager) AddTransport(session *pb.Session, userId int32, from *pb.Coordintates, to *pb.Coordintates, transport pb.Transport) error {
	// blockFrom := session.Map[from.Y+from.X]
	// blockTo := session.Map[to.Y+to.X]
	// blockFrom.Connectors

	return nil
}

func (sm *SessionsManager) ExtendLicense(session *pb.Session, userId int32, blocks []*pb.Coordintates) error {
	return nil
}
