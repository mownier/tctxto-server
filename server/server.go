package server

import (
	context "context"
	"log"
	"sync"
	"time"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type Server struct {
	players             map[string]*models.Player
	games               map[string]*models.Game
	lobbies             map[string]*models.Lobby
	gameUpdateStreams   map[string]map[string]TicTacToe_SubscribeGameUpdatesServer
	gameCreationStreams map[string]map[string]TicTacToe_SubscribeToGameCreationServer
	mu                  sync.RWMutex
	UnimplementedTicTacToeServer
}

func NewServer() *Server {
	return &Server{
		players:             make(map[string]*models.Player),
		games:               make(map[string]*models.Game),
		lobbies:             make(map[string]*models.Lobby),
		gameUpdateStreams:   make(map[string]map[string]TicTacToe_SubscribeGameUpdatesServer),
		gameCreationStreams: make(map[string]map[string]TicTacToe_SubscribeToGameCreationServer),
	}
}

func (s *Server) CreateLobby(ctx context.Context, in *CreateLobbyRequest) (*CreateLobbyReply, error) {
	player := s.addPlayer(in.PlayerName)
	lobby := s.addLobby(player)
	return &CreateLobbyReply{LobbyId: lobby.ID, PlayerId: player.ID}, nil
}

func (s *Server) JoinLobby(ctx context.Context, in *JoinLobbyRequest) (*JoinLobbyReply, error) {
	lobby, err := s.checkIfLobbyExistsWithID(in.LobbyId)
	if err != nil {
		return nil, err
	}
	player := s.addPlayer(in.PlayerName)
	s.mu.Lock()
	lobby.Players[player.ID] = player
	s.mu.Unlock()
	return &JoinLobbyReply{PlayerId: player.ID}, nil
}

func (s *Server) CreateGame(ctx context.Context, in *CreateGameRequest) (*Empty, error) {
	lobby, err := s.checkIfLobbyExistsWithID(in.LobbyId)
	if err != nil {
		return nil, err
	}
	player1, err := s.checkIfPlayerExistsInTheLobby(lobby.ID, in.Player1Id)
	if err != nil {
		return nil, err
	}
	player2, err := s.checkIfPlayerExistsInTheLobby(lobby.ID, in.Player2Id)
	if err != nil {
		return nil, err
	}
	game, err := s.addGame(lobby, player1, player2)
	if err != nil {
		return nil, err
	}
	update := &GameCreatedUpdate{
		LobbydId:  lobby.ID,
		GameId:    game.ID,
		Player1Id: player1.ID,
		Player2Id: player2.ID,
	}
	s.sendGameCreatedUpdate(update)
	return &Empty{}, nil
}

func (s *Server) MakeMoke(ctx context.Context, in *MoveRequest) (*Empty, error) {
	game, err := s.checkGameIfExistsWithID(in.GameId)
	if err != nil {
		return nil, err
	}
	if in.Position < 0 || int(in.Position) >= len(game.Board) {
		return nil, status.Errorf(codes.InvalidArgument, "position is out-of-range")
	}
	if game.Board[in.Position] != "" {
		return nil, status.Errorf(codes.InvalidArgument, "board's position is already occupied")
	}
	if game.Result == models.GAMERESULT_DRAW || game.Result == models.GAMERESULT_WIN {
		return nil, status.Errorf(codes.InvalidArgument, "game has ended already")
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
		return nil, status.Errorf(codes.InvalidArgument, "player is not allowed in the game")
	}
	if game.Mover.ID != in.PlayerId {
		return nil, status.Errorf(codes.InvalidArgument, "wait for the other player to make a move")
	}
	game.Board[in.Position] = in.PlayerId
	s.switchMover(game, index)
	update := &GameUpdate{
		GameId: game.ID,
		Board:  game.Board[:],
		Mover:  game.Mover.ID,
	}
	err = nil
	if s.checkWin(game) {
		game.Winner = mover
		game.Result = models.GAMERESULT_WIN
		update.Winner = mover.ID
		update.Result = int32(models.GAMERESULT_WIN)
		err = s.sendGameUpdate(game.ID, update)
	} else if s.checkDraw(game) {
		game.Result = models.GAMERESULT_DRAW
		update.Result = int32(models.GAMERESULT_DRAW)
		err = s.sendGameUpdate(game.ID, update)
	} else {
		err = s.sendGameUpdate(game.ID, update)
	}
	if err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

func (s *Server) SubscribeGameUpdates(in *GameUpdateSubscription, stream TicTacToe_SubscribeGameUpdatesServer) error {
	s.mu.Lock()
	if _, ok := s.gameUpdateStreams[in.GameId]; !ok {
		s.gameUpdateStreams[in.GameId] = make(map[string]TicTacToe_SubscribeGameUpdatesServer)
	}
	s.gameUpdateStreams[in.GameId][in.PlayerId] = stream
	s.mu.Unlock()
	for {
		time.Sleep(time.Hour)
	}
}

func (s *Server) SubscribeToGameCreation(in *LobbySubscription, stream TicTacToe_SubscribeToGameCreationServer) error {
	s.mu.Lock()
	if _, ok := s.gameCreationStreams[in.LobbyId]; !ok {
		s.gameCreationStreams[in.LobbyId] = make(map[string]TicTacToe_SubscribeToGameCreationServer)
	}
	s.gameCreationStreams[in.LobbyId][in.PlayerId] = stream
	s.mu.Unlock()
	for {
		time.Sleep(time.Hour)
	}
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
		Players: [2]*models.Player{player1, player2},
		Board:   [9]string{"", "", "", "", "", "", "", "", ""},
		Mover:   player1,
		Winner:  nil,
		Result:  models.GAMERESULT_ONGOING,
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

func (s *Server) sendGameUpdate(gameId string, update *GameUpdate) error {
	s.mu.Lock()
	streams, ok := s.gameUpdateStreams[gameId]
	s.mu.Unlock()
	if !ok {
		return status.Errorf(codes.NotFound, "there is no game update streams for the game")
	}
	for _, stream := range streams {
		if err := stream.Send(update); err != nil {
			log.Printf("failed to send game update: %v", err)
		}
	}
	return nil
}

func (s *Server) sendGameCreatedUpdate(update *GameCreatedUpdate) error {
	s.mu.Lock()
	streams, ok := s.gameCreationStreams[update.LobbydId]
	s.mu.Unlock()
	if !ok {
		return status.Errorf(codes.NotFound, "there is no game creation streams for the lobby")
	}
	for _, stream := range streams {
		if err := stream.Send(update); err != nil {
			log.Printf("failed to send game created update: %v", err)
		}
	}
	return nil
}

func (s *Server) switchMover(game *models.Game, index int) {
	if index == 0 {
		game.Mover = game.Players[1]
	} else {
		game.Mover = game.Players[0]
	}
}
