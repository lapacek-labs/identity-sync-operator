// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package logging

import (
	"sync"
	"time"
)

const defaultSize = 10_000

type Limiter struct {
	size    int
	mutex   sync.Mutex
	entries map[string]time.Time
}

func NewLimiter(size int) *Limiter {
	if size <= 0 {
		size = defaultSize
	}
	return &Limiter{
		size:    size,
		entries: make(map[string]time.Time, min(size, 1024)),
	}
}

func (l *Limiter) Allow(fingerprint string, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return true
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if nextAllowed, ok := l.entries[fingerprint]; ok {
		if now.Before(nextAllowed) {
			return false
		}
	}
	l.entries[fingerprint] = now.Add(interval)

	if len(l.entries) > l.size {
		l.prune(now)
	}
	return true
}

func (l *Limiter) prune(now time.Time) {
	for fp, nextAllowed := range l.entries {
		if !now.Before(nextAllowed) {
			delete(l.entries, fp)
		}
	}
	for len(l.entries) > l.size {
		for fp := range l.entries {
			delete(l.entries, fp)
			break
		}
	}
}
