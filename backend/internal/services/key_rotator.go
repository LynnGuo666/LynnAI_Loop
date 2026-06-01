package services

import (
	"sync"
	"sync/atomic"

	"loop/internal/config"
	"loop/internal/models"
	"loop/internal/repo"
)

type KeyRotator struct {
	keyRepo  *repo.APIKeyRepo
	cfg      config.Config
	counters map[int64]*atomic.Int64 // channelID -> counter
	mu       sync.Mutex
}

func NewKeyRotator(keyRepo *repo.APIKeyRepo, cfg config.Config) *KeyRotator {
	return &KeyRotator{
		keyRepo:  keyRepo,
		cfg:      cfg,
		counters: make(map[int64]*atomic.Int64),
	}
}

func (kr *KeyRotator) Select(channelID int64) (*models.APIKey, error) {
	keys, err := kr.keyRepo.ListActiveByChannel(channelID)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, ErrNoActiveKeys
	}

	idx := kr.NextIndex(channelID)
	return &keys[int(idx)%len(keys)], nil
}

// NextIndex returns the next round-robin start index for a channel, advancing
// the channel's atomic counter. Callers that hold a key slice select
// keys[(NextIndex()+attempt) % len(keys)] to spread load while retrying.
func (kr *KeyRotator) NextIndex(channelID int64) int64 {
	kr.mu.Lock()
	counter, ok := kr.counters[channelID]
	if !ok {
		counter = &atomic.Int64{}
		kr.counters[channelID] = counter
	}
	kr.mu.Unlock()

	return counter.Add(1) - 1
}

// ActiveKeys returns the active keys for a channel in a single query, letting
// the caller perform rotation/retry without re-querying per attempt.
func (kr *KeyRotator) ActiveKeys(channelID int64) ([]models.APIKey, error) {
	return kr.keyRepo.ListActiveByChannel(channelID)
}

func (kr *KeyRotator) ReportSuccess(keyID int64) {
	kr.keyRepo.RecordSuccess(keyID)
}

func (kr *KeyRotator) ReportFailure(keyID int64) {
	kr.keyRepo.RecordFailure(keyID, kr.cfg.DisableThreshold)
}

func (kr *KeyRotator) DeleteKey(keyID int64) {
	kr.keyRepo.Delete(keyID)
}
