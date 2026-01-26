// Copyright (c) 2025 Simon Lapacek
// SPDX-License-Identifier: MIT

package logging

import (
	"testing"
	"time"
)

func TestLimiter_Allow_IntervalNonPositiveAlwaysAllowsAndNoState(t *testing.T) {
	l := NewLimiter(10)
	now := time.Unix(100, 0)

	if !l.Allow("fp", now, 0) {
		t.Fatalf("expected allow for interval=0")
	}
	if !l.Allow("fp", now, -time.Second) {
		t.Fatalf("expected allow for interval<0")
	}

	// interval<=0 should not store state
	if got := len(l.entries); got != 0 {
		t.Fatalf("expected no state for interval<=0, got entries=%d", got)
	}
}

func TestLimiter_Allow_BlocksWithinInterval(t *testing.T) {
	l := NewLimiter(10)
	t0 := time.Unix(100, 0)
	interval := 10 * time.Second

	if !l.Allow("fp", t0, interval) {
		t.Fatalf("first call should allow")
	}

	// inside interval -> blocked
	t1 := t0.Add(9 * time.Second)
	if l.Allow("fp", t1, interval) {
		t.Fatalf("expected block within interval")
	}

	// exactly at boundary -> allowed (now == nextAllowed)
	t2 := t0.Add(10 * time.Second)
	if !l.Allow("fp", t2, interval) {
		t.Fatalf("expected allow at boundary now==nextAllowed")
	}

	// after allowing at t2, nextAllowed should be t2+interval; within that window -> blocked
	t3 := t2.Add(9 * time.Second)
	if l.Allow("fp", t3, interval) {
		t.Fatalf("expected block after window moved forward")
	}
}

func TestLimiter_Allow_MultipleFingerprintsIndependent(t *testing.T) {
	l := NewLimiter(10)
	now := time.Unix(100, 0)
	interval := 10 * time.Second

	if !l.Allow("a", now, interval) {
		t.Fatalf("a should allow first time")
	}
	if !l.Allow("b", now, interval) {
		t.Fatalf("b should allow first time")
	}

	// "a" blocked, "b" blocked, but they don't interfere
	if l.Allow("a", now.Add(1*time.Second), interval) {
		t.Fatalf("a should be blocked")
	}
	if l.Allow("b", now.Add(1*time.Second), interval) {
		t.Fatalf("b should be blocked")
	}

	// "a" allowed at boundary
	if !l.Allow("a", now.Add(interval), interval) {
		t.Fatalf("a should be allowed at boundary")
	}
}

func TestLimiter_Prune_RemovesExpiredEntriesWhenOversize(t *testing.T) {
	// Force prune by making size small
	l := NewLimiter(1)
	base := time.Unix(100, 0)
	interval := 10 * time.Second

	// Create an expired entry: nextAllowed = base+10s, now later (>= nextAllowed)
	if !l.Allow("expired", base, interval) {
		t.Fatalf("expected allow creating expired entry")
	}

	// Advance time beyond nextAllowed so it's expired
	now := base.Add(20 * time.Second)

	// Add another fingerprint to exceed size and trigger prune
	if !l.Allow("new", now, interval) {
		t.Fatalf("expected allow for new fingerprint")
	}

	// After prune, "expired" should be gone (because now >= nextAllowed for it)
	if _, ok := l.entries["expired"]; ok {
		t.Fatalf("expected expired entry to be pruned")
	}

	// Map should be <= size after prune
	if got := len(l.entries); got > l.size {
		t.Fatalf("expected entries <= size after prune, got %d > %d", got, l.size)
	}
}

func TestLimiter_Prune_ArbitraryEvictionIfStillOversize(t *testing.T) {
	// Set size=1. We'll insert 2 entries with long intervals so none are expired at prune time.
	l := NewLimiter(1)
	now := time.Unix(100, 0)
	interval := time.Hour

	if !l.Allow("a", now, interval) {
		t.Fatalf("expected allow for a")
	}
	if !l.Allow("b", now, interval) {
		t.Fatalf("expected allow for b")
	}

	// Because size=1, prune should have reduced entries to <=1 (by arbitrary eviction).
	if got := len(l.entries); got > 1 {
		t.Fatalf("expected entries to be <= 1 after prune, got %d", got)
	}

	// One of them must remain.
	if _, okA := l.entries["a"]; !okA {
		if _, okB := l.entries["b"]; !okB {
			t.Fatalf("expected at least one entry to remain after eviction")
		}
	}
}
