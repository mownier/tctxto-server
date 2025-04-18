package server

import (
	context "context"
	"errors"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
)

func (s *Server) CreateLobby(ctx context.Context, in *CreateLobbyRequest) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "create lobby was cancelled")
	if err != nil {
		return nil, err
	}
	return s.createLobbyInternal(clientId, in)
}

func (s *Server) createLobbyInternal(clientId string, in *CreateLobbyRequest) (*Empty, error) {
	player, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateLobbyReply(outcome)})
		return &Empty{}, nil
	}

	s.lobbyGameMu.Lock()
	defer s.lobbyGameMu.Unlock()

	if _, exists := s.playerLobbyMap[player.Id]; exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player has already a lobby",
		})})
		return &Empty{}, nil
	}

	lobby, err := s.attemptCreateLobby(player, in.Name)
	if err != nil {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.Internal),
			ErrorMessage: err.Error(),
		})})
		return &Empty{}, nil
	}

	outcome = &Outcome{Ok: true}
	updates := []*SubscriptionUpdate{s.createCreateLobbyReply(outcome)}
	if lobby != nil {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_MY_LOBBY), s.createMyLobbyDetails(lobby))
	}
	s.queueUpdatesAndSignal(clientId, updates)
	return &Empty{}, nil
}

func (s *Server) attemptCreateLobby(creator *models.Player, lobbyName string) (*models.Lobby, error) {
	const maxAttempt = 10
	for i := 0; i < maxAttempt; i++ {
		id := uuid.New().String()
		if _, exists := s.lobbies[id]; !exists {
			lobby := &models.Lobby{
				Id:      id,
				Name:    lobbyName,
				Creator: creator,
				Players: make(map[string]*models.Player),
			}
			lobby.Players[creator.Id] = creator
			s.lobbies[id] = lobby
			s.playerLobbyMap[creator.Id] = id
			return lobby, nil
		}
	}
	return nil, errors.New("unable to create lobby after multiple attempts")
}
