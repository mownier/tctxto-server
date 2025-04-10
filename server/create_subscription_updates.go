package server

import "txtcto/models"

func (s *Server) createNavigationUpdate(path NavigationPath) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_NavigationUpdate{
				NavigationUpdate: &NavigationUpdate{
					Path: path,
				},
			},
		},
	}
}

func (s *Server) createMyLobbyDetails(lobby *models.Lobby) *SubscriptionUpdate {
	details := &MyLobbyDetails{Lobby: &Lobby{Players: []*Player{}}}

	details.Lobby.Name = lobby.Name

	for _, player := range lobby.Players {
		if player != nil {
			p := &Player{Id: player.Id, Name: player.Name}
			details.Lobby.Players = append(details.Lobby.Players, p)
		}
	}

	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MyLobbyDetails{
				MyLobbyDetails: details,
			},
		},
	}
}

func (s *Server) createHandshakeReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_HandshakeReply{
				HandshakeReply: &HandshakeReply{
					Outcome: outcome,
				},
			},
		},
	}
}

func (s *Server) createInvalidateReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_InvalidateReply{
				InvalidateReply: &InvalidateReply{
					Outcome: outcome,
				},
			},
		},
	}
}

func (s *Server) createMoveUpdates(game *models.Game) []*SubscriptionUpdate {
	updates := []*SubscriptionUpdate{}

	for position, playerId := range game.Board {
		if len(playerId) > 0 {
			updates = append(updates, s.createMoveUpdate(game, playerId, int32(position)))
		}
	}

	return updates
}

func (s *Server) createMoveUpdate(game *models.Game, playerId string, position int32) *SubscriptionUpdate {
	move := &Move{Position: int32(position)}

	if playerId == game.MoverX.Id {
		move.Mover = Mover_X

	} else if playerId == game.MoverO.Id {
		move.Mover = Mover_O
	}

	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MoveUpdate{
				MoveUpdate: &MoveUpdate{
					Move: move,
				},
			},
		},
	}
}

func (s *Server) createCreateLobbyReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_CreateLobbyReply{
				CreateLobbyReply: &CreateLobbyReply{
					Outcome: outcome,
				},
			},
		},
	}
}

func (s *Server) createJoinLobbyReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_JoinLobbyReply{
				JoinLobbyReply: &JoinLobbyReply{
					Outcome: outcome,
				},
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
				LeaveMyLobbyReply: &LeaveMyLobbyReply{
					Outcome: outcome,
				},
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
				CreateGameReply: &CreateGameReply{
					Outcome: outcome,
				},
			},
		},
	}
}

func (s *Server) createGameStartUpdate(game *models.Game, you, other *models.Player) *SubscriptionUpdate {
	var moverYou Mover
	var moverOther Mover

	if game.MoverX.Id == you.Id {
		moverYou = Mover_X
	}

	if game.MoverX.Id == other.Id {
		moverOther = Mover_X
	}

	if game.MoverO.Id == you.Id {
		moverYou = Mover_O
	}

	if game.MoverO.Id == other.Id {
		moverOther = Mover_O
	}

	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_GameStartUpdate{
				GameStartUpdate: &GameStartUpdate{
					You:   moverYou,
					Other: moverOther,
				},
			},
		},
	}
}

func (s *Server) createNextMoverUpdate(game *models.Game) *SubscriptionUpdate {
	var mover Mover = Mover_O

	if game.Mover.Id == game.MoverO.Id {
		mover = Mover_O
	}

	if game.Mover.Id == game.MoverX.Id {
		mover = Mover_X
	}

	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_NextMoverUpdate{
				NextMoverUpdate: &NextMoverUpdate{
					Mover: mover,
				},
			},
		},
	}
}

func (s *Server) createPlayerClientUpdate(message string) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_PlayerClientUpdate{
				PlayerClientUpdate: &PlayerClientUpdate{
					Message: message,
				},
			},
		},
	}
}

func (s *Server) createMakeMoveReply(outcome *Outcome) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_MakeMoveReply{
				MakeMoveReply: &MakeMoveReply{
					Outcome: outcome,
				},
			},
		},
	}
}

func (s *Server) createWinnerUpdate(winner Winner, mover Mover) *SubscriptionUpdate {
	return &SubscriptionUpdate{
		Data: &SubscriptionUpdateData{
			SubscriptionUpdateDataType: &SubscriptionUpdateData_WinnerUpdate{
				WinnerUpdate: &WinnerUpdate{
					Winner: winner,
					Mover:  mover,
				},
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
