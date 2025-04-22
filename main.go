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
	"txtcto/server2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"net/http"
	_ "net/http/pprof"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("The .env file is not found. It should have TCTXTO_SERVER_PORT=3232, TCTXTO_ENABLE_RELECTION=false, and TCTXTO_CONSUMERS=?.")
	}

	port := os.Getenv("TCTXTO_SERVER_PORT")
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
	}

	consumersData, err := os.ReadFile(consumersPath)
	if err != nil {
		log.Fatalf("error reading consumers file at %s: %v\n", consumersPath, err)
	}

	var consumers []*models.Consumer
	err = json.Unmarshal(consumersData, &consumers)
	if err != nil {
		log.Fatalf("error unmarshalling consumers from %s: %v\n", consumersPath, err)
	}

	if len(consumers) == 0 {
		log.Fatalf("no consumers found in %s\n", consumersPath)
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

	server2.RegisterTicTacToeServer(s, server2.NewServer(consumersMap))

	// Start a separate HTTP server for pprof (choose a different port)
	go func() {
		pprofPort := ":6060" // Example port
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Fatalf("error getting network interfaces: %v\n", err)
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					localIP := ipnet.IP.String()
					log.Printf("tctxto server pprof running on http://%s%s\n", localIP, pprofPort)
				}
			}
		}
		if err := http.ListenAndServe(pprofPort, nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("tctxto server pprof failed to listen and serve: %v", err)
		}
	}()

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("error getting network interfaces: %v\n", err)
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIP := ipnet.IP.String()
				log.Printf("tctxto server running on tcp://%s:%s\n", localIP, port)
			}
		}
	}

	if err := s.Serve(lis); err != nil {
		log.Fatalf("tctxto server failed to serve: %v\n", err)
	}
}
