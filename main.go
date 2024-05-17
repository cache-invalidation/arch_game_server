package main

import (
	pb "game_server/api/v1"
	"game_server/config"
	"game_server/internal"
	"game_server/internal/database"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	config, err := config.ReadConfig()
	if err != nil {
		log.Fatalf("starting server error: %v", err)
	}

	db, err := database.NewDbConnector(config.DbHost, config.DbUser, config.DbPass)
	if err != nil {
		log.Fatalf("failed to connect database (%s): %v", config.DbHost, err)
	}

	lis, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", config.Port, err)
	}

	server := grpc.NewServer()
	pb.RegisterApiServer(server, internal.NewServer(db))
	log.Printf("gRPC server listening at %s\n", config.Port)

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
