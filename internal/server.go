package internal

import (
	"context"
	pb "game_server/api/v1"
	"game_server/internal/database"
	"game_server/internal/game"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedApiServer

	db              *database.DbConnector
	sessionsManager *game.SessionsManager
}

func NewServer(db *database.DbConnector) *Server {
	return &Server{
		db:              db,
		sessionsManager: game.NewSessionsManager(db),
	}
}

func (s *Server) GetSession(_ context.Context, r *pb.UserId) (*pb.Session, error) {
	session, err := s.db.GetAliveSessionByUser(r.Id)
	if err != nil {
		// добавить проверку not found
		session, err = s.sessionsManager.FindSessionForUser(r.Id)
		if err != nil {
			return nil, InternalError(err)
		}
	}

	return session, nil
}

func (s *Server) NewTransport(_ context.Context, r *pb.NewTransportReq) (*emptypb.Empty, error) {
	err := s.sessionsManager.AddTransport(r.UserId, r.From, r.To, r.Transport)
	if err != nil {
		return nil, InternalError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) ExtendLicense(_ context.Context, r *pb.ExtendLicenseReq) (*emptypb.Empty, error) {
	err := s.sessionsManager.ExtendLicense(r.UserId, r.Blocks)
	if err != nil {
		return nil, InternalError(err)
	}

	return &emptypb.Empty{}, nil
}
func (s *Server) EventStream(r *pb.UserId, srv pb.Api_EventStreamServer) error {
	log.Printf("start event stream for user: %d\n", r.Id)

	return status.Errorf(codes.Unimplemented, "method EventStream not implemented")
}
func (s *Server) StateStream(r *pb.SessionId, srv pb.Api_StateStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method StateStream not implemented")
}

func InternalError(err error) error {
	return status.Errorf(codes.Internal, "internal error: %v", err)
}
