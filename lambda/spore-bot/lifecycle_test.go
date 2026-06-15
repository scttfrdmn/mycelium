package main

import (
	"testing"
	"time"
)

func TestComputeExtendedDeadline(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	hour := time.Hour

	cases := []struct {
		name            string
		currentDeadline time.Time
		launchTime      time.Time
		newTTL          time.Duration
		extension       time.Duration
		want            time.Time
	}{
		{
			name:            "future deadline pushed forward",
			currentDeadline: now.Add(3 * hour),
			extension:       2 * hour,
			want:            now.Add(5 * hour),
		},
		{
			name:            "expired deadline floored to now+extension",
			currentDeadline: now.Add(-10 * hour), // already past — would reap immediately
			extension:       2 * hour,
			want:            now.Add(2 * hour),
		},
		{
			name:       "no deadline tag, stale launch + zero TTL floored",
			launchTime: now.Add(-30 * hour), // missing/unparseable ttl → newTTL==extension
			newTTL:     2 * hour,
			extension:  2 * hour,
			want:       now.Add(2 * hour), // launch+2h is in the past → floored
		},
		{
			name:       "no deadline tag, launch + ttl in future kept",
			launchTime: now.Add(-1 * hour),
			newTTL:     5 * hour, // launch+5h = now+4h
			extension:  2 * hour,
			want:       now.Add(4 * hour),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeExtendedDeadline(now, tc.currentDeadline, tc.launchTime, tc.newTTL, tc.extension)
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
			// Invariant: never earlier than now+extension.
			if got.Before(now.Add(tc.extension)) {
				t.Errorf("result %v is earlier than floor now+extension %v", got, now.Add(tc.extension))
			}
		})
	}
}
