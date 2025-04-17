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

func (s *Server) createGameStartUpdate(game *models.Game, you *models.Player) *ServerUpdate {
	if game.MoverX.Id == you.Id {
		return &ServerUpdate{
			Type: &ServerUpdate_GameStartUpdate{
				GameStartUpdate: &GameStartUpdate{You: Mover_X},
			},
		}
	}

	if game.MoverO.Id == you.Id {
		return &ServerUpdate{
			Type: &ServerUpdate_GameStartUpdate{
				GameStartUpdate: &GameStartUpdate{You: Mover_O},
			},
		}
	}

	return s.createPing()
}

func (s *Server) createNextMoverUpdate(you bool) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_NextMoverUpdate{
			NextMoverUpdate: &NextMoverUpdate{You: you},
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
	if playerId == game.MoverX.Id {
		return &ServerUpdate{
			Type: &ServerUpdate_MoveUpdate{
				MoveUpdate: &MoveUpdate{Move: &Move{Position: position, Mover: Mover_X}},
			},
		}
	}

	if playerId == game.MoverO.Id {
		return &ServerUpdate{
			Type: &ServerUpdate_MoveUpdate{
				MoveUpdate: &MoveUpdate{Move: &Move{Position: position, Mover: Mover_O}},
			},
		}
	}

	return s.createPing()
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

func (s *Server) createWinnerUpdate(you bool, technicality Technicality) *ServerUpdate {
	return &ServerUpdate{
		Type: &ServerUpdate_WinnerUpdate{
			WinnerUpdate: &WinnerUpdate{
				You:          you,
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
