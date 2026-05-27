package services

import (
	"sync"
	"sync/atomic"
	"time"

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

	kr.mu.Lock()
	counter, ok := kr.counters[channelID]
	if !ok {
		counter = &atomic.Int64{}
		kr.counters[channelID] = counter
	}
	kr.mu.Unlock()

	idx := counter.Add(1) - 1
	return &keys[int(idx)%len(keys)], nil
}

func (kr *KeyRotator) ReportSuccess(keyID int64) {
	k, err := kr.keyRepo.GetByID(keyID)
	if err != nil {
		return
	}
	k.ConsecutiveFailures = 0
	k.TotalSuccesses++
	now := time.Now()
	k.LastUsedAt = &now
	kr.keyRepo.Update(k)
}

func (kr *KeyRotator) ReportFailure(keyID int64) {
	k, err := kr.keyRepo.GetByID(keyID)
	if err != nil {
		return
	}
	k.ConsecutiveFailures++
	k.TotalFailures++
	now := time.Now()
	k.LastFailureAt = &now
	k.LastUsedAt = &now

	if k.ConsecutiveFailures >= kr.cfg.DisableThreshold {
		k.IsActive = false
		k.DisabledAt = &now
		nextProbe := now.Add(time.Duration(k.ProbeBackoffMin) * time.Minute)
		k.NextProbeAt = &nextProbe
	}
	kr.keyRepo.Update(k)
}

func (kr *KeyRotator) ActiveKeyCount(channelID int64) (int, error) {
	keys, err := kr.keyRepo.ListActiveByChannel(channelID)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}
