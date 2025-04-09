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

	if playerId == game.Player1.Id {
		move.Mover = Mover_X

	} else if playerId == game.Player2.Id {
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
