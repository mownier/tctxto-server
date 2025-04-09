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

	var navigationUpdate *SubscriptionUpdate
	var otherUpdates []*SubscriptionUpdate
	updates := []*SubscriptionUpdate{}

	if playerId, ok := s.clientPlayerMap[clientId]; ok {
		playerFound := false

		if _, ok := s.players[playerId]; ok {
			playerFound = true
		}

		var lobby *models.Lobby
		lobbyExists := false

		if lobbyId, ok := s.playerLobbyMap[playerId]; ok {
			if l, ok := s.lobbies[lobbyId]; ok {
				lobby = l
				lobbyExists = true
			}
		}

		var game *models.Game
		gameExists := false

		if gameId, ok := s.playerGameMap[playerId]; ok {
			if g, ok := s.games[gameId]; ok {
				game = g
				gameExists = true
			}
		}

		if playerFound {
			if !lobbyExists && gameExists {
				navigationUpdate = s.createNavigationUpdate(NavigationPath_HOME)

			} else if lobbyExists && !gameExists {
				navigationUpdate = s.createNavigationUpdate(NavigationPath_MY_LOBBY)

				if lobby != nil {
					otherUpdates = append(otherUpdates, s.createMyLobbyDetails(lobby))
				}

			} else if !lobbyExists && !gameExists {
				navigationUpdate = s.createNavigationUpdate(NavigationPath_HOME)

			} else if lobbyExists && gameExists {
				navigationUpdate = s.createNavigationUpdate(NavigationPath_GAME)

				if game != nil {
					otherUpdates = append(otherUpdates, s.createMoveUpdates(game)...)
				}

				if lobby != nil {
					otherUpdates = append(otherUpdates, s.createMyLobbyDetails(lobby))
				}

			} else {
				navigationUpdate = s.createNavigationUpdate(NavigationPath_HOME)
			}
		}
	}

	if navigationUpdate != nil {
		updates = append(updates, navigationUpdate)
		updates = append(updates, otherUpdates...)

	} else {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_LOGIN))
	}

	return updates
}
