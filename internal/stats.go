package internal

import (
	"encoding/json"
	"os"
	"sort"
	"sync"
)

type StatStore struct {
	mu    sync.Mutex
	File  string
	Stats map[int64]map[int64]*UserStats // chatID → userID → статистика
}

type UserStats struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	ChatWins  int    `json:"chat_wins"`
	TotalWins int    `json:"total_wins"`
}

// Глобальная переменная хранилища
var statStore = NewStatStore("stats.json")

// Создаёт новое хранилище статистики
func NewStatStore(filename string) *StatStore {
	store := &StatStore{
		File:  filename,
		Stats: make(map[int64]map[int64]*UserStats),
	}
	store.load()
	return store
}

// Добавляет победу игроку
func (s *StatStore) AddWin(chatID, userID int64, username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.Stats[chatID]; !ok {
		s.Stats[chatID] = make(map[int64]*UserStats)
	}

	userStats, ok := s.Stats[chatID][userID]
	if !ok {
		userStats = &UserStats{UserID: userID, Username: username}
		s.Stats[chatID][userID] = userStats
	}

	userStats.ChatWins++
	userStats.TotalWins++
	s.save()
}

// Получает топ игроков в чате
func (s *StatStore) GetTopPlayers(chatID int64, limit int) []*UserStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	var top []*UserStats
	for _, stats := range s.Stats[chatID] {
		top = append(top, stats)
	}

	sort.Slice(top, func(i, j int) bool {
		return top[i].ChatWins > top[j].ChatWins
	})

	if len(top) > limit {
		top = top[:limit]
	}
	return top
}

// Получает глобальный топ игроков
func (s *StatStore) GetGlobalTopPlayers(limit int) []*UserStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	globalMap := make(map[int64]*UserStats)
	for _, chatStats := range s.Stats {
		for uid, stat := range chatStats {
			if existing, ok := globalMap[uid]; ok {
				existing.TotalWins += stat.ChatWins
			} else {
				globalMap[uid] = &UserStats{
					UserID:    uid,
					Username:  stat.Username,
					TotalWins: stat.ChatWins,
				}
			}
		}
	}

	var top []*UserStats
	for _, stat := range globalMap {
		top = append(top, stat)
	}

	sort.Slice(top, func(i, j int) bool {
		return top[i].TotalWins > top[j].TotalWins
	})

	if len(top) > limit {
		top = top[:limit]
	}
	return top
}

// Получает статистику конкретного пользователя
func (s *StatStore) GetUserStats(chatID, userID int64) *UserStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	userStats := &UserStats{UserID: userID}

	if chatStats, ok := s.Stats[chatID]; ok {
		if stat, ok := chatStats[userID]; ok {
			userStats.ChatWins = stat.ChatWins
		}
	}

	for _, chatStats := range s.Stats {
		if stat, ok := chatStats[userID]; ok {
			userStats.TotalWins += stat.ChatWins
			userStats.Username = stat.Username
		}
	}

	return userStats
}

// Сохраняет статистику в файл
func (s *StatStore) save() {
	file, err := os.Create(s.File)
	if err != nil {
		return
	}
	defer file.Close()

	_ = json.NewEncoder(file).Encode(s.Stats)
}

// Загружает статистику из файла
func (s *StatStore) load() {
	file, err := os.Open(s.File)
	if err != nil {
		return
	}
	defer file.Close()

	_ = json.NewDecoder(file).Decode(&s.Stats)
}
