package server

import (
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Subscribe(emp *Empty, stream TicTacToe_SubscribeServer) error {
	clientId, err := s.extractClientId(stream.Context())
	if err != nil {
		return err
	}

	if !s.isClientKnown(clientId) {
		return status.Error(codes.NotFound, "unknown client")
	}

	s.setupClientSubscriptionChannels(clientId)
	s.sendInitialClientUpdates(clientId, stream)

	signal := s.getClientSignalChannel(clientId)

	defer s.cleanupClientSubscription(clientId)

	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "subscribe was cancelled")
		case <-signal:
			if err := s.sendClientUpdates(clientId, stream); err != nil {
				return err
			}
		}
	}
}

func (s *Server) isClientKnown(clientId string) bool {
	s.playerDataMu.RLock()
	defer s.playerDataMu.RUnlock()
	_, exists := s.clients[clientId]
	return exists
}

func (s *Server) setupClientSubscriptionChannels(clientId string) {
	s.clientSubscriptionMu.Lock()
	defer s.clientSubscriptionMu.Unlock()
	if _, exists := s.clientSignalMap[clientId]; !exists {
		s.clientSignalMap[clientId] = make(chan struct{}, 1)
	}
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}
}

func (s *Server) getClientSignalChannel(clientId string) <-chan struct{} {
	s.clientSubscriptionMu.RLock()
	defer s.clientSubscriptionMu.RUnlock()
	return s.clientSignalMap[clientId]
}

func (s *Server) sendInitialClientUpdates(clientId string, stream TicTacToe_SubscribeServer) {
	initialUpdates := s.clientInitialUpdates(clientId)
	s.clientSubscriptionMu.Lock()
	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId], initialUpdates...)
	s.clientSubscriptionMu.Unlock()
	s.sendClientUpdates(clientId, stream)
}

func (s *Server) sendClientUpdates(clientId string, stream TicTacToe_SubscribeServer) error {
	s.clientSubscriptionMu.Lock()
	defer s.clientSubscriptionMu.Unlock()

	lastIndex, exists := s.clientLastIndexUpdate[clientId]
	if !exists {
		lastIndex = -1
	}

	startIndex := 0
	if lastIndex != -1 {
		startIndex = lastIndex + 1
	}

	updatesToSend := s.clientUpdatesMap[clientId][startIndex:]
	sentCount := 0
	var err error

	for _, update := range updatesToSend {
		if e := stream.Send(update); e != nil {
			err = e
			break
		}
		sentCount++
	}

	if sentCount > 0 {
		s.clientLastIndexUpdate[clientId] += sentCount
	}

	return err
}

func (s *Server) clientInitialUpdates(clientId string) []*SubscriptionUpdate {
	s.playerDataMu.Lock()
	defer s.playerDataMu.Unlock()

	playerId, playerExists := s.clientPlayerMap[clientId]

	if !playerExists || !s.isPlayerActive(playerId) {
		return []*SubscriptionUpdate{s.createNavigationUpdate(NavigationPath_LOGIN)}
	}

	if gameUpdates := s.getGameInitialUpdates(playerId); len(gameUpdates) > 0 {
		return gameUpdates
	}

	if lobbyUpdates := s.getLobbyInitialUpdates(playerId); len(lobbyUpdates) > 0 {
		return lobbyUpdates
	}

	return []*SubscriptionUpdate{s.createNavigationUpdate(NavigationPath_HOME)}
}

func (s *Server) isPlayerActive(playerId string) bool {
	s.playerDataMu.RLock()
	defer s.playerDataMu.RUnlock()
	_, ok := s.players[playerId]
	return ok
}

func (s *Server) getGameInitialUpdates(playerId string) []*SubscriptionUpdate {
	s.lobbyGameMu.Lock()
	defer s.lobbyGameMu.Unlock()

	if gameId, ok := s.playerGameMap[playerId]; ok {
		if game, ok := s.games[gameId]; ok && game.Result < models.GameResult_DRAW {
			you, other := s.getGamePlayers(game, playerId)
			updates := []*SubscriptionUpdate{
				s.createNavigationUpdate(NavigationPath_GAME),
				s.createGameStartUpdate(game, you, other),
				s.createNextMoverUpdate(game),
			}
			updates = append(updates, s.createMoveUpdates(game)...)
			return updates
		} else {
			delete(s.playerGameMap, playerId)
		}
	}
	return nil
}

func (s *Server) getGamePlayers(game *models.Game, playerId string) (*models.Player, *models.Player) {
	var you *models.Player
	var other *models.Player
	if game.MoverX.Id == playerId {
		you = game.MoverX
		other = game.MoverO
	} else if game.MoverO.Id == playerId {
		you = game.MoverO
		other = game.MoverX
	}
	return you, other
}

func (s *Server) getLobbyInitialUpdates(playerId string) []*SubscriptionUpdate {
	s.lobbyGameMu.Lock()
	defer s.lobbyGameMu.Unlock()

	if lobbyId, ok := s.playerLobbyMap[playerId]; ok {
		if lobby, ok := s.lobbies[lobbyId]; ok {
			return []*SubscriptionUpdate{
				s.createNavigationUpdate(NavigationPath_MY_LOBBY),
				s.createMyLobbyDetails(lobby),
			}
		} else {
			delete(s.playerLobbyMap, playerId)
		}
	}
	return nil
}

func (s *Server) cleanupClientSubscription(clientId string) {
	s.clientSubscriptionMu.Lock()
	defer s.clientSubscriptionMu.Unlock()
	delete(s.clientSignalMap, clientId)
}
