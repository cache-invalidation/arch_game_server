package internal

import (
	"context"
	"errors"
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
		if errors.Is(err, database.ErrSessionNotFound) {
			session, err = s.sessionsManager.FindSessionForUser(r.Id)
			if err != nil {
				return nil, InternalError(err)
			}
		} else {
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

func (s *Server) StateStream(r *pb.StateStreamReq, srv pb.Api_StateStreamServer) error {
	log.Printf("start session %d state stream for user: %d\n", r.SessionId.Id, r.UserId.Id)

	return s.sessionsManager.StreamState(r.SessionId.Id, r.UserId.Id, srv)
}

func InternalError(err error) error {
	log.Printf("internal error: %v\n", err)
	return status.Errorf(codes.Internal, "internal error: %v", err)
}
