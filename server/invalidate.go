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
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createInvalidateReply(outcome), s.createNavigationUpdate(NavigationPath_LOGIN)})
		return &Empty{}, nil
	}

	s.cleanupInvalidatedClient(clientId, player.Id)
	s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createInvalidateReply(&Outcome{Ok: true}), s.createNavigationUpdate(NavigationPath_LOGIN)})

	return &Empty{}, nil
}

func (s *Server) cleanupInvalidatedClient(clientId, playerId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clientPlayerMap, clientId)
	delete(s.playerClientMap, playerId)
}
