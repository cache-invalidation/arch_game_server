package database

import (
	pb "game_server/api/v1"

	"github.com/tarantool/go-tarantool"
)

type DbConnector struct {
	conn *tarantool.Connection
}

func NewDbConnector(host, user, pass string) (*DbConnector, error) {
	conn, err := tarantool.Connect(host, tarantool.Opts{
		User: user,
		Pass: pass,
	})
	if err != nil {
		return nil, err
	}

	return &DbConnector{
		conn: conn,
	}, nil
}

func (db *DbConnector) Close() error {
	return db.conn.Close()
}

func (db *DbConnector) AddSession(session *pb.Session) error {
	return nil
}

func (db *DbConnector) UpdateSession(session *pb.Session) error {
	return nil
}

func (db *DbConnector) GetSession(id int32) (*pb.Session, error) {
	return nil, nil
}

func (db *DbConnector) GetAliveSessionByUser(userId int32) (*pb.Session, error) {
	return nil, nil
}
