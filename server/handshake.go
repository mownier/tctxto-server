package server

import (
	context "context"
	"fmt"
	"log"
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
	s.playerDataMu.Lock()

	if _, exists := s.clients[clientId]; !exists {
		s.playerDataMu.Unlock()
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	updates := []*SubscriptionUpdate{s.createHandshakeReply(&Outcome{})}

	player, outcome := s.addPlayer(in)
	updates[0] = s.createHandshakeReply(outcome)

	if outcome.Ok {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_HOME))

		if oldClientId, exists := s.playerClientMap[player.Id]; exists {
			if _, exists := s.clients[oldClientId]; exists {
				oldClientIdUpdates := []*SubscriptionUpdate{
					s.createPlayerClientUpdate("player is using another client"),
					s.createNavigationUpdate(NavigationPath_LOGIN),
				}
				s.queueUpdatesAndSignal(oldClientId, append(updates, oldClientIdUpdates...))
			}
		}

		s.clientPlayerMap[clientId] = player.Id
		s.playerClientMap[player.Id] = clientId

		s.playerDataMu.Unlock() // Release playerDataMu before accessing lobby data

		s.lobbyGameMu.RLock()
		if id, exists := s.playerLobbyMap[player.Id]; exists {
			if lobby, exists := s.lobbies[id]; exists {
				updates = append(updates, s.createMyLobbyDetails(lobby))
			}
		}
		s.lobbyGameMu.RUnlock()
	} else {
		s.playerDataMu.Unlock()
	}

	log.Printf("handshakeInternal clientId = %s, updates = %d\n", clientId, len(updates))

	s.queueUpdatesAndSignal(clientId, updates)
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
