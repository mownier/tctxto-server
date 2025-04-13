package server2

import (
	"fmt"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
)

func (s *Server) signUp(clientId string, in *SignUpRequest) error {
	if _, exists := s.playerNameId.get(in.Name); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignUpReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player with name already exists",
		}))
		return nil
	}

	id := uuid.New().String()

	if _, exists := s.players.get(id); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignUpReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "unable to generate player",
		}))
		return nil
	}

	player := &models.Player{
		Id:          uuid.New().String(),
		Name:        in.Name,
		Pass:        in.Pass,
		DisplayName: fmt.Sprintf("user%s", s.generateRandomString(12)),
	}

	s.players.set(player.Id, player)
	s.playerNameId.set(player.Name, player.Id)
	s.playerClient.set(player.Id, clientId)

	s.clientPlayer.set(clientId, player.Id)

	s.queueServerUpdatesAndSignal(clientId,
		s.createSignUpReply(&Outcome{Ok: true}),
		s.createPlayerDisplayNameUpdate(player.DisplayName),
		s.createNavigationUpdate(NavigationPath_HOME, true),
	)

	return nil
}
