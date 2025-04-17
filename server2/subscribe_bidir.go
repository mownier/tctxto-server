package server2

import (
	"io"
	"time"
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) SubscribeBiDir(stream TicTacToe_SubscribeBiDirServer) error {
	publicKey, err := s.extractPublicKeyWithCancel(stream.Context(), "subscribe was cancelled")
	if err != nil {
		return err
	}

	if _, exists := s.consumers.get(publicKey); !exists {
		return status.Error(codes.PermissionDenied, "rejected")
	}

	clientId, err := s.extractClientId(stream.Context())
	if err != nil {
		clientId = uuid.New().String()
		s.clients.set(clientId, &models.Client{Id: clientId})
	}

	_, exists := s.clients.get(clientId)
	if !exists {
		return status.Error(codes.NotFound, "unknown client")
	}

	if err := stream.Send(s.createClientAssignmentUpdate(clientId)); err != nil {
		return status.Error(codes.Internal, "unable to send client assignment update")
	}

	if _, exists := s.clientServerUpdates.get(clientId); !exists {
		s.clientServerUpdates.set(clientId, []*ServerUpdate{})
	}

	if _, exists := s.clientSignal.get(clientId); !exists {
		s.clientSignal.set(clientId, make(chan struct{}, 1))
	}

	s.sendInitialServerUpdates(clientId, stream)

	signal, _ := s.clientSignal.get(clientId)

	defer s.cleanupClientResources(clientId)

	go func() {
		for {
			clientUpdate, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			s.Notify(stream.Context(), clientUpdate)
		}
	}()

	pingInterval := 100 * time.Millisecond
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "subscribe was done")
		case <-pingTicker.C:
			if err := stream.Send(s.createPing()); err != nil {
				return err
			}
		case <-signal:
			if err := s.sendServerUpdates(stream, clientId); err != nil {
				return err
			}
		}
	}
}
