package database

import (
	"context"
	"errors"
	"fmt"
	pb "game_server/api/v1"
	"time"

	"github.com/spf13/cast"

	jsoniter "github.com/json-iterator/go"

	"github.com/tarantool/go-tarantool/v2"
)

var ErrSessionNotFound = errors.New("session not found")

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type DbConnector struct {
	conn *tarantool.Connection
}

func NewDbConnector(host, user, pass string) (*DbConnector, error) {
	ctx := context.Background()

	dialer := tarantool.NetDialer{
		Address:  host,
		User:     user,
		Password: pass,
	}

	opts := tarantool.Opts{
		Reconnect: time.Second,
	}

	conn, err := tarantool.Connect(ctx, dialer, opts)
	if err != nil {
		return nil, err
	}

	return &DbConnector{
		conn: conn,
	}, nil
}

func sessionToTntTuple(session *pb.Session) []interface{} {
	return []interface{}{
		uint64(session.Id),
		session.Users,
		session.Map,
		session.TimeLimit,
		session.Status,
		session.StartTime,
	}
}

func tntTupleToSession(tuple []interface{}) (*pb.Session, error) {
	b, err := json.Marshal(map[string]interface{}{
		"id":        tuple[0],
		"users":     tuple[1],
		"map":       tuple[2],
		"timeLimit": tuple[3],
		"status":    tuple[4],
		"startTime": tuple[5],
	})

	if err != nil {
		return nil, err
	}

	var s *pb.Session
	err = json.Unmarshal(b, &s)
	return s, err
}

func (db *DbConnector) Close() error {
	return db.conn.Close()
}

func (db *DbConnector) AddSession(session *pb.Session) error {
	req := tarantool.NewInsertRequest("sessions").Tuple(sessionToTntTuple(session))
	_, err := db.conn.Do(req).Get()

	return err
}

func (db *DbConnector) UpdateSession(session *pb.Session) error {
	req := tarantool.NewReplaceRequest("sessions").Tuple(sessionToTntTuple(session))
	_, err := db.conn.Do(req).Get()

	return err
}

func (db *DbConnector) GetSession(id int32) (*pb.Session, error) {
	req := tarantool.NewSelectRequest("sessions").Index("id").Iterator(tarantool.IterEq).Key([]interface{}{uint64(id)})
	resp, err := db.conn.Do(req).GetResponse()
	if err != nil {
		return nil, fmt.Errorf("can't get session with id %d: %w", id, err)
	}
	selResp, ok := resp.(*tarantool.SelectResponse)
	if !ok {
		return nil, errors.New("wrong response type")
	}

	data, err := selResp.Decode()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, ErrSessionNotFound
	}

	return tntTupleToSession(data[0].([]interface{}))
}

func (db *DbConnector) GetAliveSessionByUser(userId int32) (*pb.Session, error) {
	// First step is to get the session id for the user
	req := tarantool.NewSelectRequest("joinedusers")
	resp, err := db.conn.Do(req).GetResponse()
	if err != nil {
		return nil, fmt.Errorf("can't find session for user id %d: %w", userId, err)
	}
	selResp, ok := resp.(*tarantool.SelectResponse)
	if !ok {
		return nil, errors.New("wrong response type")
	}

	data, err := selResp.Decode()
	if err != nil {
		return nil, err
	}

	for _, tuple := range data {
		tuple2 := tuple.([]interface{})
		if cast.ToInt32(tuple2[0]) == userId {
			sessionId := cast.ToInt32(tuple2[1])
			return db.GetSession(sessionId)
		}
	}

	return nil, ErrSessionNotFound
}
