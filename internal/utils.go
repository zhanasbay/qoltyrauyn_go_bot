package internal

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"time"
)

// LoadWordsFromFile загружает слова из текстового файла (по одному на строку)
func LoadWordsFromFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("❌ Failed to open words file:", err)
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		word := scanner.Text()
		if word != "" {
			words = append(words, word)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("❌ Error reading words:", err)
	}

	return words
}

// GetRandomWord возвращает случайное слово из списка
func GetRandomWord(words []string) string {
	rand.Seed(time.Now().UnixNano())
	return words[rand.Intn(len(words))]
}
