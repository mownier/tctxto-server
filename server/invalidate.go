package server

import (
	context "context"
)

func (s *Server) Invalidate(ctx context.Context, emp *Empty) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "invalidate was cancelled")
	if err != nil {
		return nil, err
	}
	return s.invalidateInternal(clientId)
}

func (s *Server) invalidateInternal(clientId string) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueSignalUpdatesOnInvalidate(clientId, outcome)
		return &Empty{}, nil
	}

	s.cleanupInvalidatedClient(clientId, player.Id)
	s.queueSignalUpdatesOnInvalidate(clientId, &Outcome{Ok: true})

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
	delete(s.playerClientMap, playerId)
}
