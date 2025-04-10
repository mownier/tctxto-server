package server

import (
	context "context"
	"fmt"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Handshake(ctx context.Context, in *HandshakeRequest) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "handshake was cancelled")
	if err != nil {
		return nil, err
	}
	return s.handshakeInternal(clientId, in)
}

func (s *Server) handshakeInternal(clientId string, in *HandshakeRequest) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	updates := []*SubscriptionUpdate{}

	player, outcome := s.addPlayer(in)

	updates = append(updates, s.createHandshakeReply(outcome))

	if outcome.Ok {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_HOME))

		if oldClientId, exists := s.playerClientMap[player.Id]; exists {
			if _, exists := s.clients[oldClientId]; exists {
				oldClientIdUpdates := append(updates,
					s.createPlayerClientUpdate("player is using another client"),
					s.createNavigationUpdate(NavigationPath_LOGIN),
				)

				if _, exists := s.clientUpdatesMap[oldClientId]; !exists {
					s.clientUpdatesMap[oldClientId] = []*SubscriptionUpdate{}
				}

				s.clientUpdatesMap[oldClientId] = append(s.clientUpdatesMap[oldClientId], oldClientIdUpdates...)

				if signal, exists := s.clientSignalMap[oldClientId]; exists {
					select {
					case signal <- struct{}{}:
						break

					default:
						break
					}
				}
			}
		}

		s.clientPlayerMap[clientId] = player.Id
		s.playerClientMap[player.Id] = clientId

		if id, exists := s.playerLobbyMap[player.Id]; exists {
			if lobby, exists := s.lobbies[id]; exists {
				updates = append(updates, s.createMyLobbyDetails(lobby))
			}
		}
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

	return &Empty{}, nil
}

func (s *Server) addPlayer(in *HandshakeRequest) (*models.Player, *Outcome) {
	outcome := &Outcome{}

	playerId, exists := s.playerNameIdMap[in.PlayerName]

	if exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.AlreadyExists)
		outcome.ErrorMessage = "player name already exists"

		return nil, outcome
	}

	player, exists := s.players[playerId]

	if exists {
		if player.Name == in.PlayerName && player.Pass == in.PlayerPass {
			outcome.Ok = true

			return player, outcome
		}

		outcome.Ok = false
		outcome.ErrorCode = int32(codes.PermissionDenied)
		outcome.ErrorMessage = "invalid player credentials"

		return nil, outcome
	}

	const maxAttempt = 10

	for i := 0; i < maxAttempt; i++ {
		id := uuid.New().String()

		if _, exists := s.players[id]; !exists {
			outcome.Ok = true

			player := &models.Player{Id: id, Name: in.PlayerName}

			s.players[id] = player
			s.playerNameIdMap[in.PlayerName] = id

			return player, outcome
		}
	}

	outcome.Ok = false
	outcome.ErrorCode = int32(codes.Internal)
	outcome.ErrorMessage = fmt.Sprintf("failed to add player with name %s", in.PlayerName)

	return nil, outcome
}
