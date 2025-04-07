package models

type Player struct {
	ID   string
	Name string
}

type Lobby struct {
	ID      string
	Creator *Player
	Players map[string]*Player
	Games   map[string]*Game
}

type Game struct {
	ID      string
	LobbyId string
	Players [2]*Player
	Board   [9]string
	Mover   *Player
	Winner  *Player
	Result  GameResult
}

type GameResult int32

var (
	GAMERESULT_ONGOING GameResult = 0
	GAMERESULT_WIN     GameResult = 1
	GAMERESULT_DRAW    GameResult = 2
	GAMERESULT_INITIAL GameResult = 3
)
