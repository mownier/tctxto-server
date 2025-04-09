package models

type Lobby struct {
	Id      string
	Name    string
	Creator *Player
	Players map[string]*Player
}

type Player struct {
	Id   string
	Name string
	Pass string
}

type Game struct {
	Id      string
	Board   [9]string
	Mover   *Player
	Player1 *Player
	Player2 *Player
	Winner  *Player
	Result  GameResult
}

type GameResult int32

const (
	GameResult_INITIAL GameResult = 0
	GameResult_ONGOING GameResult = 1
	GameResult_WIN     GameResult = 2
	GameResult_DRAW    GameResult = 3
)
