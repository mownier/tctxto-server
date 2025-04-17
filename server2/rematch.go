package server2

import (
	"txtcto/models"

	"google.golang.org/grpc/codes"
)

func (s *Server) rematch(clientId string, in *RematchRequest) error {
	player, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createRematchReply(outcome))

		return nil
	}

	rematchId, exists := s.playerRematch.get(player.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "rematch not found",
		}))

		return nil
	}

	rematch, exists := s.rematches.get(rematchId)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "rematch details not found",
		}))

		return nil
	}

	pd, exists := rematch.GetPlayerDecision(player.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "you are not one of the participants for the rematch",
		}))

		return nil
	}

	if in.Yes {
		rematch.SetPlayerDecision(pd.Player.Id, models.Decision_YES)
	} else {
		rematch.SetPlayerDecision(pd.Player.Id, models.Decision_NO)
	}

	var you *models.Player
	var other *models.Player
	if rematch.PlayerDecisions[0].Player.Id == player.Id {
		you = rematch.PlayerDecisions[0].Player
		other = rematch.PlayerDecisions[1].Player
	} else {
		you = rematch.PlayerDecisions[1].Player
		other = rematch.PlayerDecisions[0].Player
	}

	game, upd := s.evaluateRematch(rematch)

	updates := []*ServerUpdate{s.createRematchReply(&Outcome{Ok: true})}
	if rematch.Pending() {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_REMATCH))
	}
	updates = append(updates, upd...)
	if game != nil {
		updates = append(updates,
			s.createNavigationUpdate(NavigationPath_GAME),
			s.createGameStartUpdate(game, you, other),
			s.createNextMoverUpdate(game),
		)
	}
	if rematch.Cancelled() {
		updates = append(updates, s.initialServerUpdates(clientId)...)
	}
	s.queueServerUpdatesAndSignal(clientId, updates...)

	otherClientId, exists := s.playerClient.get(other.Id)
	if !exists {
		return nil
	}

	updates = []*ServerUpdate{}
	updates = append(updates, upd...)
	if game != nil {
		updates = append(updates,
			s.createNavigationUpdate(NavigationPath_GAME),
			s.createGameStartUpdate(game, other, you),
			s.createNextMoverUpdate(game),
		)
	}
	if rematch.Cancelled() {
		updates = append(updates, s.initialServerUpdates(otherClientId)...)
	}
	s.queueServerUpdatesAndSignal(otherClientId, updates...)

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
