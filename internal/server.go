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
	"google.golang.org/protobuf/types/known/durationpb"
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

func (s *Server) GetSetup(context.Context, *emptypb.Empty) (*pb.Setup, error) {
	return &pb.Setup{
		TimeLimitMin: int32(game.TimeLimitMin),
		LicenseCost:  game.LicenseCost,
		OnpPenalty:   game.OnpPenalty,

		CostBus:   game.Cost_BUS,
		CostMetro: game.Cost_METRO,
		CostTaxi:  game.Cost_TAXI,
		CostTram:  game.Cost_TRAM,

		DurationBus:   durationpb.New(game.Duration_BUS),
		DurationMetro: durationpb.New(game.Duration_METRO),
		DurationTaxi:  durationpb.New(game.Duration_TAXI),
		DurationTram:  durationpb.New(game.Duration_TRAM),
	}, nil
}

func (s *Server) GetSession(_ context.Context, r *pb.UserId) (*pb.Session, error) {
	log.Printf("get session req, user: %d\n", r.Id)

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
	log.Printf("found exists session %d for user %d\n", session.Id, r.Id)

	return session, nil
}

func (s *Server) NewTransport(_ context.Context, r *pb.NewTransportReq) (*emptypb.Empty, error) {
	log.Printf("new transport req, user: %d, transport: %s, from: %s, to: %s\n", r.UserId, r.Transport.String(), r.From.String(), r.To.String())

	err := s.sessionsManager.AddTransport(r.UserId, r.From, r.To, r.Transport)
	if err != nil {
		return nil, InternalError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) ExtendLicense(_ context.Context, r *pb.ExtendLicenseReq) (*emptypb.Empty, error) {
	log.Printf("extend license req, user: %d, license:\n", r.UserId)
	for _, block := range r.Blocks {
		log.Printf("%s\n", block.String())
	}

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
