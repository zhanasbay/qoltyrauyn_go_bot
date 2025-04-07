package internal

import "time"

// GameState представляет состояние игры в одном чате
type GameState struct {
	CurrentWord   string
	HostID        int64
	WordGuesserID int64
	WordTime      time.Time
	LastStartTime time.Time
}

// games — карта: chatID → состояние игры
var games = make(map[int64]*GameState)

// getGame возвращает состояние игры для указанного чата
func GetGame(chatID int64) *GameState {
	if game, ok := games[chatID]; ok {
		return game
	}
	games[chatID] = &GameState{}
	return games[chatID]
}
