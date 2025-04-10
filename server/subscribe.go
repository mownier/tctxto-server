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

	s.mu.Lock()

	if _, exists := s.clients[clientId]; !exists {
		s.mu.Unlock()

		return status.Error(codes.NotFound, "unknown client")
	}

	if _, exists := s.clientSignalMap[clientId]; !exists {
		s.clientSignalMap[clientId] = make(chan struct{}, 1)
	}

	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId], s.clientInitialUpdates(clientId)...)

	signal := s.clientSignalMap[clientId]

	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clientSignalMap, clientId)
		s.mu.Unlock()
	}()

	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "subscribe was cancelled")

		case <-signal:
			s.mu.Lock()
			err := s.sendClientUpdates(clientId, stream)
			s.mu.Unlock()

			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) sendClientUpdates(clientId string, stream TicTacToe_SubscribeServer) error {
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

	var e error

	for _, update := range updatesToSend {
		if err := stream.Send(update); err != nil {
			e = err
			break
		}

		sentCount++
	}

	if sentCount > 0 {
		s.clientLastIndexUpdate[clientId] += sentCount
	}

	return e
}

func (s *Server) clientInitialUpdates(clientId string) []*SubscriptionUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerId, playerExists := s.clientPlayerMap[clientId]

	if !playerExists || !s.isPlayerActive(playerId) {
		return []*SubscriptionUpdate{
			s.createNavigationUpdate(NavigationPath_LOGIN),
		}
	}

	if gameUpdates := s.getGameInitialUpdates(playerId); len(gameUpdates) > 0 {
		return gameUpdates
	}

	if lobbyUpdates := s.getLobbyInitialUpdates(playerId); len(lobbyUpdates) > 0 {
		return lobbyUpdates
	}

	return []*SubscriptionUpdate{
		s.createNavigationUpdate(NavigationPath_HOME),
	}
}

func (s *Server) isPlayerActive(playerId string) bool {
	_, ok := s.players[playerId]

	return ok
}

func (s *Server) getGameInitialUpdates(playerId string) []*SubscriptionUpdate {
	if gameId, ok := s.playerGameMap[playerId]; ok {
		if game, ok := s.games[gameId]; ok && game.Result != models.GameResult_DRAW && game.Result != models.GameResult_WIN {
			var you *models.Player
			var other *models.Player

			if game.MoverX.Id == playerId {
				you = game.MoverX
				other = game.MoverO
			}

			if game.MoverO.Id == playerId {
				you = game.MoverO
				other = game.MoverX
			}

			updates := []*SubscriptionUpdate{
				s.createNavigationUpdate(NavigationPath_GAME),
				s.createGameStartUpdate(game, you, other),
				s.createNextMoverUpdate(game),
			}

			updates = append(updates,
				s.createMoveUpdates(game)...,
			)

			return updates

		} else {
			delete(s.playerGameMap, playerId)
		}
	}

	return nil
}

func (s *Server) getLobbyInitialUpdates(playerId string) []*SubscriptionUpdate {
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
