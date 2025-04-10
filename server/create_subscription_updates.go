package server

import "txtcto/models"

func (s *Server) createNavigationUpdate(path NavigationPath) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_NavigationUpdate{
				NavigationUpdate: &NavigationUpdate{Path: path},
			},
		},
	}
}

func (s *Server) createMyLobbyDetails(lobby *models.Lobby) *SubscriptionUpdate {
	players := make([]*Player, 0, len(lobby.Players))
	for _, player := range lobby.Players {
		if player != nil {
			players = append(players, &Player{Id: player.Id, Name: player.Name})
		}
	}
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MyLobbyDetails{
				MyLobbyDetails: &MyLobbyDetails{
					Lobby: &Lobby{Name: lobby.Name, Players: players},
				},
			},
		},
	}
}

func (s *Server) createHandshakeReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_HandshakeReply{
				HandshakeReply: &HandshakeReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createInvalidateReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_InvalidateReply{
				InvalidateReply: &InvalidateReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createMoveUpdates(game *models.Game) []*SubscriptionUpdate {
	updates := make([]*SubscriptionUpdate, 0, len(game.Board))
	for position, playerId := range game.Board {
		if len(playerId) > 0 {
			updates = append(updates, s.createMoveUpdate(game, playerId, int32(position)))
		}
	}
	return updates
}

func (s *Server) createMoveUpdate(game *models.Game, playerId string, position int32) *SubscriptionUpdate {
	mover := Mover_UNSPECIFIED
	if playerId == game.MoverX.Id {
		mover = Mover_X
	} else if playerId == game.MoverO.Id {
		mover = Mover_O
	}
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MoveUpdate{
				MoveUpdate: &MoveUpdate{Move: &Move{Position: position, Mover: mover}}},
		},
	}
}

func (s *Server) createCreateLobbyReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_CreateLobbyReply{
				CreateLobbyReply: &CreateLobbyReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createJoinLobbyReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_JoinLobbyReply{
				JoinLobbyReply: &JoinLobbyReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createMyLobbyJoinerUpdate(player *models.Player) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MyLobbyJoinerUpdate{
				MyLobbyJoinerUpdate: &MyLobbyJoinerUpdate{
					Player: &Player{Id: player.Id, Name: player.Name},
				},
			},
		},
	}
}

func (s *Server) createLeaveMyLobbyReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_LeaveMyLobbyReply{
				LeaveMyLobbyReply: &LeaveMyLobbyReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createMyLobbyLeaverUpdate(player *models.Player) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MyLobbyLeaverUpdate{
				MyLobbyLeaverUpdate: &MyLobbyLeaverUpdate{
					Player: &Player{Id: player.Id, Name: player.Name},
				},
			},
		},
	}
}

func (s *Server) createCreateGameReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_CreateGameReply{
				CreateGameReply: &CreateGameReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createGameStartUpdate(game *models.Game, you, other *models.Player) *SubscriptionUpdate {
	moverYou := Mover_UNSPECIFIED
	moverOther := Mover_UNSPECIFIED

	if game.MoverX.Id == you.Id {
		moverYou = Mover_X
	} else if game.MoverO.Id == you.Id {
		moverYou = Mover_O
	}

	if game.MoverX.Id == other.Id {
		moverOther = Mover_X
	} else if game.MoverO.Id == other.Id {
		moverOther = Mover_O
	}

	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_GameStartUpdate{
				GameStartUpdate: &GameStartUpdate{You: moverYou, Other: moverOther},
			},
		},
	}
}

func (s *Server) createNextMoverUpdate(game *models.Game) *SubscriptionUpdate {
	mover := Mover_UNSPECIFIED
	if game.Mover.Id == game.MoverX.Id {
		mover = Mover_X
	} else if game.Mover.Id == game.MoverO.Id {
		mover = Mover_O
	}
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_NextMoverUpdate{
				NextMoverUpdate: &NextMoverUpdate{Mover: mover},
			},
		},
	}
}

func (s *Server) createPlayerClientUpdate(message string) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_PlayerClientUpdate{
				PlayerClientUpdate: &PlayerClientUpdate{Message: message},
			},
		},
	}
}

func (s *Server) createMakeMoveReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MakeMoveReply{
				MakeMoveReply: &MakeMoveReply{Outcome: outcome},
			},
		},
	}
}

func (s *Server) createWinnerUpdate(winner Winner, mover Mover) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_WinnerUpdate{
				WinnerUpdate: &WinnerUpdate{Winner: winner, Mover: mover},
			},
		},
	}
}

func (s *Server) createDrawupdate() *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_DrawUpdate{
				DrawUpdate: &DrawUpdate{},
			},
		},
	}
}
