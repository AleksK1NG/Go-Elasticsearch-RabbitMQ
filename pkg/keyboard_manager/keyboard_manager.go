package keyboard_manager

import (
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"strings"
	"sync"
)

type KeyboardLayoutManager interface {
	GetOppositeLayoutWord(originalWord string) string
}

type keyboardLayoutsManager struct {
	log         logger.Logger
	keyMappings map[string]string
	sbPool      *sync.Pool
}

func NewKeyboardLayoutManager(log logger.Logger, keyMappings map[string]string) *keyboardLayoutsManager {
	sbPool := &sync.Pool{New: func() any { return new(strings.Builder) }}
	return &keyboardLayoutsManager{log: log, keyMappings: keyMappings, sbPool: sbPool}
}

func (k *keyboardLayoutsManager) GetOppositeLayoutWord(originalWord string) string {
	originalWord = strings.ReplaceAll(originalWord, "Ё", "Е")
	originalWord = strings.ReplaceAll(originalWord, "ё", "е")

	sb := k.sbPool.Get().(*strings.Builder)
	defer k.sbPool.Put(sb)
	sb.Reset()

	for _, c := range originalWord {
		lowerCasedChar := strings.ToLower(string(c))
		if char, ok := k.keyMappings[lowerCasedChar]; ok {
			sb.WriteString(char)
		} else {
			sb.WriteString(lowerCasedChar)
		}
	}
	return sb.String()
}
