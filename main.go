package main

import (
	"fmt"
	"log"
	"net"
	"txtcto/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = ":3232"
)

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	reflection.Register(s)
	server.RegisterTicTacToeServer(s, server.NewServer())
	fmt.Printf("listening on tcp://localhost%s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
