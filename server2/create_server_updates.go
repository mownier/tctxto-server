package server2

import "txtcto/models"

func (s *Server) createClientAssignmentUpdate(clientId string) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_ClientAssignmentUpdate{
			ClientAssignmentUpdate: &ClientAssignmentUpdate{
				ClientId: clientId,
			},
		},
	}
}

func (s *Server) createPing() *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_Ping{
			Ping: &Ping{},
		},
	}
}

func (s *Server) createNavigationUpdate(path NavigationPath) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_NavigationUpdate{
			NavigationUpdate: &NavigationUpdate{
				Path: path,
			},
		},
	}
}

func (s *Server) createSignUpReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_SignUpReply{
			SignUpReply: &SignUpReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createMyLobbyDetails(lobby *models.Lobby) *ServerUpdate {
	players := make([]*Player, 0, len(lobby.Players))
	for _, player := range lobby.Players {
		if player != nil {
			players = append(players, &Player{Id: player.Id, Name: player.DisplayName})
		}
	}
	return &ServerUpdate{
		Type: &ServerUpdate_MyLobbyDetails{
			MyLobbyDetails: &MyLobbyDetails{
				Lobby: &Lobby{Id: lobby.Id, Name: lobby.Name, Players: players},
			},
		},
	}
}

func (s *Server) createGameStartUpdate(game *models.Game, you, other *models.Player) *ServerUpdate {
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

	return &ServerUpdate{
		Type: &ServerUpdate_GameStartUpdate{
			GameStartUpdate: &GameStartUpdate{You: moverYou, Other: moverOther},
		},
	}
}

func (s *Server) createNextMoverUpdate(game *models.Game) *ServerUpdate {
	mover := Mover_UNSPECIFIED
	if game.Mover.Id == game.MoverX.Id {
		mover = Mover_X
	} else if game.Mover.Id == game.MoverO.Id {
		mover = Mover_O
	}
	return &ServerUpdate{
		Type: &ServerUpdate_NextMoverUpdate{
			NextMoverUpdate: &NextMoverUpdate{Mover: mover},
		},
	}
}

func (s *Server) createMoveUpdates(game *models.Game) []*ServerUpdate {
	updates := make([]*ServerUpdate, 0, len(game.Board))
	for position, playerId := range game.Board {
		if len(playerId) > 0 {
			updates = append(updates, s.createMoveUpdate(game, playerId, int32(position)))
		}
	}
	return updates
}

func (s *Server) createMoveUpdate(game *models.Game, playerId string, position int32) *ServerUpdate {
	mover := Mover_UNSPECIFIED
	if playerId == game.MoverX.Id {
		mover = Mover_X
	} else if playerId == game.MoverO.Id {
		mover = Mover_O
	}
	return &ServerUpdate{
		Type: &ServerUpdate_MoveUpdate{
			MoveUpdate: &MoveUpdate{Move: &Move{Position: position, Mover: mover}},
		},
	}
}

func (s *Server) createSignInReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_SignInReply{
			SignInReply: &SignInReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createPlayerClientUpdate(message string) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_PlayerClientUpdate{
			PlayerClientUpdate: &PlayerClientUpdate{Message: message},
		},
	}
}

func (s *Server) createPlayerDisplayNameUpdate(displayName string) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_PlayerDisplayNameUpdate{
			PlayerDisplayNameUpdate: &PlayerDisplayNameUpdate{DisplayName: displayName},
		},
	}
}

func (s *Server) createSignOutReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_SignOutReply{
			SignOutReply: &SignOutReply{Outcome: outcome},
		},
	}
}

func (s *Server) createCreateLobbyReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_CreateLobbyReply{
			CreateLobbyReply: &CreateLobbyReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createJoinLobbyReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_JoinLobbyReply{
			JoinLobbyReply: &JoinLobbyReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createLeaveMyLobbyReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_LeaveMyLobbyReply{
			LeaveMyLobbyReply: &LeaveMyLobbyReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createMyLobbyLeaverUpdate(id, name string) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_MyLobbyLeaverUpdate{
			MyLobbyLeaverUpdate: &MyLobbyLeaverUpdate{
				Player: &Player{Id: id, Name: name},
			},
		},
	}
}

func (s *Server) createMyLobbyJoinerUpdate(id, name string) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_MyLobbyJoinerUpdate{
			MyLobbyJoinerUpdate: &MyLobbyJoinerUpdate{
				Player: &Player{Id: id, Name: name},
			},
		},
	}
}

func (s *Server) createGameReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_CreateGameReply{
			CreateGameReply: &CreateGameReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createMakeMoveReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_MakeMoveReply{
			MakeMoveReply: &MakeMoveReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createWinnerUpdate(mover Mover, winner Winner, technicality Technicality) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_WinnerUpdate{
			WinnerUpdate: &WinnerUpdate{
				Mover:        mover,
				Winner:       winner,
				Technicality: technicality,
			},
		},
	}
}

func (s *Server) createDrawUpdate() *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_DrawUpdate{
			DrawUpdate: &DrawUpdate{},
		},
	}
}

func (s *Server) createRematchReply(outcome *Outcome) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_RematchReply{
			RematchReply: &RematchReply{
				Outcome: outcome,
			},
		},
	}
}

func (s *Server) createRematchDenied() *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_RematchDenied{
			RematchDenied: &RematchDenied{},
		},
	}
}

func (s *Server) createRematchApproved() *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_RematchApproved{
			RematchApproved: &RematchApproved{},
		},
	}
}

func (s *Server) createRematchPending() *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_RematchPending{
			RematchPending: &RematchPending{},
		},
	}
}
