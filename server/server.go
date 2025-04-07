package server

import (
	context "context"
	"fmt"
	"sync"
	"time"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type Server struct {
	players                    map[string]*models.Player
	games                      map[string]*models.Game
	lobbies                    map[string]*models.Lobby
	createdGamesQueueLastIndex map[string]map[string]int
	createdGamesQueue          map[string][]string
	gameUpdatesQueue           map[string][]*GameUpdate
	gameUpdatesQueueLastIndex  map[string]map[string]int
	mu                         sync.RWMutex
	UnimplementedTicTacToeServer
}

func NewServer() *Server {
	return &Server{
		players:                    make(map[string]*models.Player),
		games:                      make(map[string]*models.Game),
		lobbies:                    make(map[string]*models.Lobby),
		createdGamesQueue:          make(map[string][]string),
		createdGamesQueueLastIndex: make(map[string]map[string]int),
		gameUpdatesQueue:           make(map[string][]*GameUpdate),
		gameUpdatesQueueLastIndex:  make(map[string]map[string]int),
	}
}

func (s *Server) CreateLobby(ctx context.Context, in *CreateLobbyRequest) (*CreateLobbyReply, error) {
	fmt.Printf("CreateLobby in = %v\n", in)
	player := s.addPlayer(in.PlayerName)
	s.mu.Lock()
	for _, lobby := range s.lobbies {
		for _, player := range lobby.Players {
			if player.Name == in.PlayerName {
				s.mu.Unlock()
				fmt.Printf("CreateLobby player has already a lobby")
				return &CreateLobbyReply{LobbyId: lobby.ID, PlayerId: player.ID}, nil
			}
		}
	}
	s.mu.Unlock()
	lobby := s.addLobby(player)
	return &CreateLobbyReply{LobbyId: lobby.ID, PlayerId: player.ID}, nil
}

func (s *Server) JoinLobby(ctx context.Context, in *JoinLobbyRequest) (*JoinLobbyReply, error) {
	fmt.Printf("JoinLobby in = %v\n", in)
	s.mu.Lock()
	for _, lobby := range s.lobbies {
		for _, player := range lobby.Players {
			if player.Name == in.PlayerName {
				s.mu.Unlock()
				fmt.Printf("JoinLobby player has already a lobby")
				return &JoinLobbyReply{PlayerId: player.ID}, nil
			}
		}
	}
	s.mu.Unlock()
	lobby, err := s.checkIfLobbyExistsWithID(in.LobbyId)
	if err != nil {
		fmt.Printf("JoinLobby err 1 = %v\n", err)
		return nil, err
	}
	player := s.addPlayer(in.PlayerName)
	s.mu.Lock()
	lobby.Players[player.ID] = player
	s.mu.Unlock()
	return &JoinLobbyReply{PlayerId: player.ID}, nil
}

func (s *Server) CreateGame(ctx context.Context, in *CreateGameRequest) (*Empty, error) {
	fmt.Printf("CreateGame in = %v\n", in)
	lobby, err := s.checkIfLobbyExistsWithID(in.LobbyId)
	if err != nil {
		fmt.Printf("CreateGame err 1 = %v\n", err)
		return nil, err
	}
	player1, err := s.checkIfPlayerExistsInTheLobby(lobby.ID, in.Player1Id)
	if err != nil {
		fmt.Printf("CreateGame err 2 = %v\n", err)
		return nil, err
	}
	player2, err := s.checkIfPlayerExistsInTheLobby(lobby.ID, in.Player2Id)
	if err != nil {
		fmt.Printf("CreateGame err 3 = %v\n", err)
		return nil, err
	}
	game, err := s.addGame(lobby, player1, player2)
	if err != nil {
		fmt.Printf("CreateGame err 4 = %v\n", err)
		return nil, err
	}
	s.mu.Lock()
	_, exists := s.createdGamesQueue[in.LobbyId]
	if !exists {
		s.createdGamesQueue[in.LobbyId] = []string{}
	}
	s.createdGamesQueue[in.LobbyId] = append(s.createdGamesQueue[in.LobbyId], game.ID)
	s.mu.Unlock()
	return &Empty{}, nil
}

func (s *Server) MakeMoke(ctx context.Context, in *MoveRequest) (*Empty, error) {
	fmt.Printf("MakeMoke in = %v\n", in)
	update, err := s.toGameUpdate(in)
	if err != nil {
		return nil, err
	}
	fmt.Printf("MakeMoke game update = %v\n", update)
	s.mu.Lock()
	_, ok := s.gameUpdatesQueue[in.GameId]
	if !ok {
		s.gameUpdatesQueue[in.GameId] = []*GameUpdate{}
	}
	s.gameUpdatesQueue[in.GameId] = append(s.gameUpdatesQueue[in.GameId], update)
	s.mu.Unlock()
	return &Empty{}, nil
}

func (s *Server) SubscribeGameUpdates(in *GameUpdateSubscription, stream TicTacToe_SubscribeGameUpdatesServer) error {
	fmt.Printf("SubscribeGameUpdates in = %v\n", in)
	s.mu.Lock()
	game, exists := s.games[in.GameId]
	s.mu.Unlock()

	if !exists {
		return status.Errorf(codes.InvalidArgument, "SubscribeGameUpdates game %s does not exist", in.GameId)
	}

	s.mu.Lock()
	_, ok := s.gameUpdatesQueueLastIndex[in.GameId]
	if !ok {
		s.gameUpdatesQueueLastIndex[in.GameId] = make(map[string]int)
	}
	_, ok = s.gameUpdatesQueue[in.GameId]
	if !ok {
		s.gameUpdatesQueue[in.GameId] = []*GameUpdate{}
	}
	gameUpdate := &GameUpdate{
		GameId: game.ID,
		Board:  game.Board[:],
		Mover:  game.Mover.ID,
		Result: int32(game.Result),
	}
	fmt.Printf("SubscribeGameUpdates game update = %v\n", gameUpdate)
	s.gameUpdatesQueue[in.GameId] = append(s.gameUpdatesQueue[in.GameId], gameUpdate)
	s.mu.Unlock()

	ticker := time.NewTicker(1 * time.Second) // Send every 1 second
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			lastIndex, ok := s.gameUpdatesQueueLastIndex[in.GameId][in.PlayerId]
			if !ok {
				lastIndex = -1
				s.gameUpdatesQueueLastIndex[in.GameId][in.PlayerId] = lastIndex
			}
			var list []*GameUpdate
			if lastIndex == -1 {
				list = s.gameUpdatesQueue[in.GameId]
			} else {
				for i, j := range s.gameUpdatesQueue[in.GameId] {
					if i > lastIndex {
						list = append(list, j)
					}
				}
			}
			lastIndex = lastIndex + len(list)
			s.gameUpdatesQueueLastIndex[in.GameId][in.PlayerId] = lastIndex
			s.mu.Unlock()
			if len(list) == 0 {
				if err := stream.Send(&GameUpdate{}); err != nil {
					fmt.Printf("SubscribeGameUpdates sending ping err = %v\n", err)
					return err
				}
				continue
			}
			for i, update := range list {
				if err := stream.Send(update); err != nil {
					fmt.Printf("SubscribeGameUpdates [%d] err 2 = %v\n", i, err)
					return err
				}
			}
		case <-stream.Context().Done():
			fmt.Println("SubscribeGameUpdates Client disconnected")
			return nil
		}
	}
}

func (s *Server) SubscribeToGameCreation(in *LobbySubscription, stream TicTacToe_SubscribeToGameCreationServer) error {
	fmt.Printf("SubscribeToGameCreation in = %v, datetime = %v\n", in, time.Now())

	s.mu.Lock()
	_, ok := s.createdGamesQueue[in.LobbyId]
	if !ok {
		s.createdGamesQueue[in.LobbyId] = []string{}
	}
	_, ok = s.createdGamesQueueLastIndex[in.LobbyId]
	if !ok {
		s.createdGamesQueueLastIndex[in.LobbyId] = make(map[string]int)
	}
	var ongoingGame *models.Game = nil
	for _, game := range s.games {
		if game.Players[0].ID == in.PlayerId || game.Players[1].ID == in.PlayerId {
			if game.Result == models.GAMERESULT_ONGOING {
				ongoingGame = game
				break
			}
		}
	}
	s.mu.Unlock()

	if ongoingGame != nil {
		fmt.Printf("SubscribeToGameCreation player %s has game that is ongoing\n", in.PlayerId)
		s.mu.Lock()
		s.createdGamesQueue[in.LobbyId] = append(s.createdGamesQueue[in.LobbyId], ongoingGame.ID)
		s.mu.Unlock()
	}

	ticker := time.NewTicker(1 * time.Second) // Send every 1 second
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			lastIndex, ok := s.createdGamesQueueLastIndex[in.LobbyId][in.PlayerId]
			if !ok {
				lastIndex = -1
				s.createdGamesQueueLastIndex[in.LobbyId][in.PlayerId] = lastIndex
			}
			var list []string
			if lastIndex == -1 {
				list = s.createdGamesQueue[in.LobbyId]
			} else {
				for i, j := range s.createdGamesQueue[in.LobbyId] {
					if i > lastIndex {
						list = append(list, j)
					}
				}
			}
			lastIndex = lastIndex + len(list)
			s.createdGamesQueueLastIndex[in.LobbyId][in.PlayerId] = lastIndex
			s.mu.Unlock()
			if len(list) == 0 {
				if err := stream.Send(&GameCreatedUpdate{}); err != nil {
					fmt.Printf("SubscribeToGameCreation sending ping err = %v\n", err)
					return err
				}
				fmt.Printf("SubscribeToGameCreation sending ping ok, datetime = %v\n", time.Now())
				continue
			}
			for i, id := range list {
				s.mu.Lock()
				game, ok := s.games[id]
				s.mu.Unlock()
				if !ok {
					fmt.Printf("SubscribeToGameCreation [%d] ignored err 1 = game %s not found\n", i, id)
					continue
				}
				var playerFound = false
				for _, p := range game.Players {
					if p.ID == in.PlayerId {
						playerFound = true
						break
					}
				}
				if !playerFound {
					fmt.Printf("SubscribeToGameCreation [%d] ignored err 1 = player %s not found\n", i, in.PlayerId)
					continue
				}
				update := &GameCreatedUpdate{
					GameId:    game.ID,
					LobbydId:  game.LobbyId,
					Player1Id: game.Players[0].ID,
					Player2Id: game.Players[1].ID,
				}
				if err := stream.Send(update); err != nil {
					fmt.Printf("SubscribeToGameCreation [%d] err 2 = %v\n", i, err)
					return err
				}
			}
		case <-stream.Context().Done():
			fmt.Printf("SubscribeToGameCreation Client disconnected, datetime = %v\n", time.Now())
			return nil
		}
	}
}

func (s *Server) toGameUpdate(in *MoveRequest) (*GameUpdate, error) {
	fmt.Printf("toGameUpdate in = %v\n", in)
	game, err := s.checkGameIfExistsWithID(in.GameId)
	if err != nil {
		fmt.Printf("toGameUpdate err 1 = %v\n", err)
		return nil, err
	}
	update := &GameUpdate{
		GameId: game.ID,
		Board:  game.Board[:],
		Mover:  game.Mover.ID,
	}
	if in.Position < 0 || int(in.Position) >= len(game.Board) {
		err := status.Errorf(codes.InvalidArgument, "position is out-of-range")
		fmt.Printf("toGameUpdate err 2 = position is out-of-range, %v\n", err)
		return update, nil
	}
	if game.Board[in.Position] != "" {
		err := status.Errorf(codes.InvalidArgument, "board's position is already occupied")
		fmt.Printf("toGameUpdate err 3 = board's position is already occupied, %v\n", err)
		return update, nil
	}
	if game.Result == models.GAMERESULT_DRAW || game.Result == models.GAMERESULT_WIN {
		err := status.Errorf(codes.InvalidArgument, "game has ended already")
		fmt.Printf("toGameUpdate err 4 = game has ended already, %v\n", err)
		return update, nil
	}
	var index = -1
	var mover *models.Player
	for i, p := range game.Players {
		if p.ID == in.PlayerId {
			mover = p
			index = i
			break
		}
	}
	if index == -1 || mover == nil {
		fmt.Printf("toGameUpdate err 5 = player is not allowed in the game\n")
		return nil, status.Errorf(codes.InvalidArgument, "player is not allowed in the game")
	}
	if game.Mover.ID != in.PlayerId {
		err := status.Errorf(codes.InvalidArgument, "wait for the other player to make a move")
		fmt.Printf("toGameUpdate err 6 = wait for the other player to make a move, %v\n", err)
		return update, nil
	}
	game.Board[in.Position] = in.PlayerId
	s.switchMover(game, index)
	update = &GameUpdate{
		GameId: game.ID,
		Board:  game.Board[:],
		Mover:  game.Mover.ID,
	}
	if s.checkWin(game) {
		game.Winner = mover
		game.Result = models.GAMERESULT_WIN
		update.Winner = mover.ID
		update.Result = int32(models.GAMERESULT_WIN)
	} else if s.checkDraw(game) {
		game.Result = models.GAMERESULT_DRAW
		update.Result = int32(models.GAMERESULT_DRAW)
	}
	return update, nil
}

func (s *Server) checkIfLobbyExistsWithID(id string) (*models.Lobby, error) {
	s.mu.Lock()
	lobby, ok := s.lobbies[id]
	s.mu.Unlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "lobby not found")
	}
	return lobby, nil
}

func (s *Server) checkIfPlayerExistsInTheLobby(lobbyId string, playerId string) (*models.Player, error) {
	s.mu.Lock()
	_, ok := s.lobbies[lobbyId]
	s.mu.Unlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "lobby not found when checking existing players")
	}
	var player *models.Player
	s.mu.Lock()
	for _, p := range s.lobbies[lobbyId].Players {
		if p.ID == playerId {
			player = p
		}
	}
	s.mu.Unlock()
	if player == nil {
		return nil, status.Errorf(codes.NotFound, "player not found in the lobby")
	}
	return player, nil
}

func (s *Server) generatePlayerId() string {
	for {
		id := uuid.New().String()
		if _, ok := s.players[id]; !ok {
			return id
		}
	}
}

func (s *Server) generateLobbyId() string {
	for {
		id := uuid.New().String()
		if _, ok := s.lobbies[id]; !ok {
			return id
		}
	}
}

func (s *Server) generateGameId() string {
	for {
		id := uuid.New().String()
		if _, ok := s.games[id]; !ok {
			return id
		}
	}
}

func (s *Server) addPlayer(name string) *models.Player {
	var player *models.Player
	s.mu.Lock()
	for _, p := range s.players {
		if p.Name == name {
			player = p
		}
	}
	s.mu.Unlock()
	if player != nil {
		return player
	}
	id := s.generatePlayerId()
	player = &models.Player{ID: id, Name: name}
	s.mu.Lock()
	s.players[id] = player
	s.mu.Unlock()
	return player
}

func (s *Server) addLobby(player *models.Player) *models.Lobby {
	id := s.generateLobbyId()
	lobby := &models.Lobby{
		ID:      id,
		Creator: player,
		Games:   make(map[string]*models.Game),
		Players: make(map[string]*models.Player),
	}
	s.mu.Lock()
	s.lobbies[id] = lobby
	lobby.Players[player.ID] = player
	s.mu.Unlock()
	return lobby
}

func (s *Server) addGame(lobby *models.Lobby, player1 *models.Player, player2 *models.Player) (*models.Game, error) {
	id := s.generateGameId()
	game := &models.Game{
		ID:      id,
		LobbyId: lobby.ID,
		Players: [2]*models.Player{player1, player2},
		Board:   [9]string{"", "", "", "", "", "", "", "", ""},
		Mover:   player1,
		Winner:  nil,
		Result:  models.GAMERESULT_INITIAL,
	}
	s.mu.Lock()
	s.games[id] = game
	lobby.Games[id] = game
	s.mu.Unlock()
	return game, nil
}

func (s *Server) checkGameIfExistsWithID(id string) (*models.Game, error) {
	s.mu.Lock()
	game, ok := s.games[id]
	s.mu.Unlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "game not found")
	}
	return game, nil
}

func (s *Server) checkDraw(game *models.Game) bool {
	for _, tile := range game.Board {
		if tile == "" {
			return false
		}
	}
	return true
}

func (s *Server) checkWin(game *models.Game) bool {
	wins := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // Rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // Columns
		{0, 4, 8}, {2, 4, 6}, // Diagonals
	}
	for _, win := range wins {
		if game.Board[win[0]] != "" && game.Board[win[0]] == game.Board[win[1]] && game.Board[win[1]] == game.Board[win[2]] {
			return true
		}
	}
	return false
}

func (s *Server) switchMover(game *models.Game, index int) {
	if index == 0 {
		game.Mover = game.Players[1]
	} else {
		game.Mover = game.Players[0]
	}
}
