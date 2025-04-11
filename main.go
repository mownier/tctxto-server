package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"txtcto/models"
	"txtcto/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("TCTXTO_PORT")
	enableReflectionStr := os.Getenv("TCTXTO_ENABLE_RELECTION")
	consumersPath := os.Getenv("TCTXTO_CONSUMERS")

	if len(port) == 0 {
		port = "3232"
	}

	reflectionEnabled := false
	if enableReflectionStr != "" {
		var err error
		reflectionEnabled, err = strconv.ParseBool(strings.ToLower(enableReflectionStr))
		if err != nil {
			log.Printf("warning: invalid value for TCTXTO_ENABLE_RELECTION: %q, defaulting to false\n", enableReflectionStr)
		}
	}

	if consumersPath == "" {
		log.Fatalln("need to specify the path to the consumers JSON file (TCTXTO_CONSUMERS environment variable)")
		os.Exit(1)
	}

	consumersData, err := os.ReadFile(consumersPath)
	if err != nil {
		log.Fatalf("error reading consumers file at %s: %v\n", consumersPath, err)
		os.Exit(1)
	}

	var consumers []*models.Consumer
	err = json.Unmarshal(consumersData, &consumers)
	if err != nil {
		log.Fatalf("error unmarshalling consumers from %s: %v\n", consumersPath, err)
		os.Exit(1)
	}

	if len(consumers) == 0 {
		log.Fatalf("no consumers found in %s\n", consumersPath)
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	s := grpc.NewServer()

	if reflectionEnabled {
		reflection.Register(s)
	}

	consumersMap := make(map[string]*models.Consumer)
	for _, consumer := range consumers {
		consumersMap[consumer.PublicKey] = consumer
	}

	server.RegisterTicTacToeServer(s, server.NewServer(consumersMap))

	fmt.Printf("listening on tcp://localhost:%s\n", port)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
