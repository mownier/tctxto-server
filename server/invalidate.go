package server

import (
	context "context"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Invalidate(ctx context.Context, emp *Empty) (*Empty, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "invalidate was cancelled")

	default:
		clientId, err := s.extractClientId(ctx)

		if err != nil {
			return nil, err
		}

		return s.invalidate(clientId)
	}
}

func (s *Server) invalidate(clientId string) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	player, outcome := s.checkPlayer(clientId)

	s.cleanupInvalidatedClient(clientId, player.Id)
	s.queueSignalUpdatesOnInvalidate(clientId, outcome)

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnInvalidate(clientId string, outcome *Outcome) {
	updates := []*SubscriptionUpdate{
		s.createInvalidateReply(outcome),
		s.createNavigationUpdate(NavigationPath_LOGIN),
	}

	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId], updates...)

	if signal, exists := s.clientSignalMap[clientId]; exists {
		select {
		case signal <- struct{}{}:

			break
		default:
			break
		}
	}
}

func (s *Server) cleanupInvalidatedClient(clientId, playerId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clientPlayerMap, clientId)
	delete(s.playerClientMap[playerId], clientId)
}
