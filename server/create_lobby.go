package server

import (
	context "context"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) CreateLobby(ctx context.Context, in *CreateLobbyRequest) (*Empty, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "create lobby was cancelled")

	default:
		clientId, err := s.extractClientId(ctx)

		if err != nil {
			return nil, err
		}

		return s.createLobby(clientId, in)
	}
}

func (s *Server) createLobby(clientId string, in *CreateLobbyRequest) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	player, outcome := s.checkPlayer(clientId)

	if !outcome.Ok {
		s.queueSignalUpdatesOnCreateLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	if _, exists := s.playerLobbyMap[player.Id]; exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.AlreadyExists)
		outcome.ErrorMessage = "player has already a lobby"

		s.queueSignalUpdatesOnCreateLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	const maxAttempt = 10

	for i := 0; i < maxAttempt; i++ {
		id := uuid.New().String()

		if _, exists := s.lobbies[id]; !exists {
			lobby := &models.Lobby{
				Id:      id,
				Name:    in.Name,
				Creator: player,
				Players: make(map[string]*models.Player),
			}

			lobby.Players[player.Id] = player

			s.lobbies[id] = lobby
			s.playerLobbyMap[player.Id] = id

			outcome.Ok = true

			s.queueSignalUpdatesOnCreateLobby(clientId, outcome, lobby)

			return &Empty{}, nil
		}
	}

	outcome.Ok = false
	outcome.ErrorCode = int32(codes.Internal)
	outcome.ErrorMessage = "unable to create lobby"

	s.queueSignalUpdatesOnCreateLobby(clientId, outcome, nil)

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnCreateLobby(clientId string, outcome *Outcome, lobby *models.Lobby) {
	updates := []*SubscriptionUpdate{
		s.createCreateLobbyReply(outcome),
	}

	if lobby != nil {
		updates = append(updates,
			s.createMyLobbyDetails(lobby),
			s.createNavigationUpdate(NavigationPath_MY_LOBBY),
		)
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
