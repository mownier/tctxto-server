package server2

import (
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

func (s *Server) rematch(youClientId string, in *RematchRequest) error {
	you, outcome := s.validatePlayer(youClientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(youClientId, s.createRematchReply(outcome))
		return nil
	}

	rematchId, exists := s.playerRematch.get(you.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(youClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "rematch not found",
		}))
		return nil
	}

	rematch, exists := s.rematches.get(rematchId)
	if !exists {
		s.queueServerUpdatesAndSignal(youClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "rematch details not found",
		}))
		return nil
	}

	_, exists = rematch.GetPlayerDecision(you.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(youClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "you are not one of the participants for the rematch",
		}))
		return nil
	}

	if in.Yes {
		rematch.SetPlayerDecision(you.Id, models.Decision_YES)
	} else {
		rematch.SetPlayerDecision(you.Id, models.Decision_NO)
	}

	var other *models.Player
	if rematch.PlayerDecisions[0].Player.Id == you.Id {
		other = rematch.PlayerDecisions[1].Player
	} else {
		other = rematch.PlayerDecisions[0].Player
	}

	otherClientId, exists := s.playerClient.get(other.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(youClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "the other player has no client",
		}))
		return nil
	}

	game, updates := s.evaluateRematch(rematch)

	s.queueServerUpdatesAndSignal(youClientId, s.createRematchReply(&Outcome{Ok: true}))
	s.queueServerUpdatesAndSignal(youClientId, updates...)
	s.queueServerUpdatesAndSignal(otherClientId, updates...)

	if game != nil {
		s.queueServerUpdatesAndSignal(youClientId,
			s.createNavigationUpdate(NavigationPath_GAME),
			s.createGameStartUpdate(game, you),
			s.createNextMoverUpdate(s.areYouTheMover(game, you)),
		)

		s.queueServerUpdatesAndSignal(otherClientId,
			s.createNavigationUpdate(NavigationPath_GAME),
			s.createGameStartUpdate(game, other),
			s.createNextMoverUpdate(s.areYouTheMover(game, other)),
		)

		return nil
	}

	if rematch.Pending() {
		s.queueServerUpdatesAndSignal(youClientId, s.createNavigationUpdate(NavigationPath_REMATCH))

		return nil
	}

	if rematch.Cancelled() {
		s.queueServerUpdatesAndSignal(youClientId, s.initialServerUpdates(youClientId)...)
		s.queueServerUpdatesAndSignal(otherClientId, s.initialServerUpdates(otherClientId)...)

		return nil
	}

	return nil
}

func (s *Server) evaluateRematch(rematch *models.Rematch) (*models.Game, []*ServerUpdate) {
	if rematch.Cancelled() {
		for _, pd := range rematch.PlayerDecisions {
			s.playerRematch.delete(pd.Player.Id)
			s.playerGame.delete(pd.Player.Id)
		}
		s.rematches.delete(rematch.Id)

		return nil, []*ServerUpdate{s.createRematchDenied()}
	}

	if rematch.Confirmed() {
		for _, pd := range rematch.PlayerDecisions {
			s.playerRematch.delete(pd.Player.Id)
			s.playerGame.delete(pd.Player.Id)
		}
		s.rematches.delete(rematch.Id)
		game, outcome := s.setupGame(
			rematch.PlayerDecisions[0].Player,
			rematch.PlayerDecisions[0].Player,
			rematch.PlayerDecisions[1].Player,
		)
		if !outcome.Ok {
			rematch.SetPlayerDecision(rematch.PlayerDecisions[0].Player.Id, models.Decision_NO)
			rematch.SetPlayerDecision(rematch.PlayerDecisions[1].Player.Id, models.Decision_NO)
			return nil, []*ServerUpdate{s.createRematchDenied()}
		}
		return game, []*ServerUpdate{s.createRematchApproved()}
	}

	if rematch.Pending() {
		return nil, []*ServerUpdate{s.createRematchPending()}
	}

	return nil, []*ServerUpdate{}
}

func (s *Server) setupRematch(you, other *models.Player) (*models.Rematch, *Outcome) {
	rematchId := uuid.New().String()

	if _, exists := s.rematches.get(rematchId); exists {
		return nil, &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "unable to create rematch",
		}
	}

	rematch := &models.Rematch{
		Id:              rematchId,
		PlayerDecisions: [2]*models.PlayerDecision{},
	}

	rematch.PlayerDecisions[0] = &models.PlayerDecision{
		Player:   you,
		Decision: models.Decision_UNDECIDED,
	}

	rematch.PlayerDecisions[1] = &models.PlayerDecision{
		Player:   other,
		Decision: models.Decision_UNDECIDED,
	}

	s.playerRematch.set(you.Id, rematch.Id)
	s.playerRematch.set(other.Id, rematch.Id)
	s.rematches.set(rematch.Id, rematch)

	return rematch, &Outcome{Ok: true}
}
