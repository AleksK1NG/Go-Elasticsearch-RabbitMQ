package misstype_manager

import (
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"strings"
	"sync"
)

type MissTypeManager interface {
	GetMissTypedWord(originalWord string) string
}

type KeyboardMissTypeManager struct {
	log         logger.Logger
	keyMappings map[string]string
	sbPool      *sync.Pool
}

func NewMissTypeManager(log logger.Logger, keyMappings map[string]string) *KeyboardMissTypeManager {
	sbPool := &sync.Pool{New: func() any { return new(strings.Builder) }}
	return &KeyboardMissTypeManager{log: log, keyMappings: keyMappings, sbPool: sbPool}
}

func (k *KeyboardMissTypeManager) GetMissTypedWord(originalWord string) string {
	sb := k.sbPool.Get().(*strings.Builder)
	defer k.sbPool.Put(sb)
	sb.Reset()

	for _, c := range []rune(originalWord) {
		lowerCasedChar := strings.ToLower(string(c))
		if char, ok := k.keyMappings[lowerCasedChar]; ok {
			sb.WriteString(char)
		} else {
			sb.WriteString(lowerCasedChar)
		}
	}
	return sb.String()
}
