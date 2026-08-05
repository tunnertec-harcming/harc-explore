package evolution

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/harc/soundscape/apps/api/internal/data"
	"github.com/harc/soundscape/apps/api/internal/models"
)

type EventType string

const (
	EventPlayStart  EventType = "play_start"
	EventHeartbeat  EventType = "heartbeat"
	EventPackSwitch EventType = "pack_switch"
	EventPlayStop   EventType = "play_stop"
)

type Event struct {
	Type        EventType     `json:"type"`
	TS          int64         `json:"ts"`
	Mode        models.ModeID `json:"mode"`
	PackID      string        `json:"pack_id"`
	FromPackID  string        `json:"from_pack_id,omitempty"`
	DurationMin int           `json:"duration_min,omitempty"`
	ListenedSec float64       `json:"listened_sec,omitempty"`
}

type DeviceStats struct {
	ListenSec   map[string]float64 `json:"listen_sec"`
	PlayCount   map[string]int     `json:"play_count"`
	EarlySwitch map[string]int     `json:"early_switch"` // from_pack penalized
}

type Store struct {
	mu    sync.RWMutex
	stats map[string]*DeviceStats
}

func NewStore() *Store {
	return &Store{stats: map[string]*DeviceStats{}}
}

func (s *Store) ensure(deviceID string) *DeviceStats {
	st, ok := s.stats[deviceID]
	if !ok {
		st = &DeviceStats{
			ListenSec:   map[string]float64{},
			PlayCount:   map[string]int{},
			EarlySwitch: map[string]int{},
		}
		s.stats[deviceID] = st
	}
	return st
}

func (s *Store) Ingest(deviceID string, events []Event) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.ensure(deviceID)
	n := 0
	for _, e := range events {
		if e.PackID == "" {
			continue
		}
		switch e.Type {
		case EventPlayStart:
			st.PlayCount[e.PackID]++
		case EventHeartbeat, EventPlayStop:
			if e.ListenedSec > 0 {
				st.ListenSec[e.PackID] += e.ListenedSec
			}
		case EventPackSwitch:
			if e.FromPackID != "" && e.ListenedSec > 0 && e.ListenedSec < 45 {
				st.EarlySwitch[e.FromPackID]++
			} else if e.FromPackID != "" {
				// treat short unspecified switch as mild penalty
				st.EarlySwitch[e.FromPackID]++
			}
			st.PlayCount[e.PackID]++
		default:
			continue
		}
		n++
	}
	return n
}

func (s *Store) Insights(deviceID string) DeviceStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.stats[deviceID]
	if !ok {
		return DeviceStats{
			ListenSec:   map[string]float64{},
			PlayCount:   map[string]int{},
			EarlySwitch: map[string]int{},
		}
	}
	// copy
	out := DeviceStats{
		ListenSec:   map[string]float64{},
		PlayCount:   map[string]int{},
		EarlySwitch: map[string]int{},
	}
	for k, v := range st.ListenSec {
		out.ListenSec[k] = v
	}
	for k, v := range st.PlayCount {
		out.PlayCount[k] = v
	}
	for k, v := range st.EarlySwitch {
		out.EarlySwitch[k] = v
	}
	return out
}

type ProfileDelta struct {
	Energy       float64 `json:"energy"`
	Brightness   float64 `json:"brightness"`
	Masking      float64 `json:"masking"`
	Space        float64 `json:"space"`
	EventDensity float64 `json:"event_density"`
}

type RecommendResult struct {
	PackID string  `json:"pack_id"`
	Score  float64 `json:"score"`
	Source string  `json:"source"`
	Reason string  `json:"reason"`
}

type PersonalizeResult struct {
	PackID       string       `json:"pack_id"`
	ProfileDelta ProfileDelta `json:"profile_delta"`
	Source       string       `json:"source"`
	Reason       string       `json:"reason"`
}

func (s *Store) Recommend(deviceID string, mode models.ModeID, hourLocal int) RecommendResult {
	defaults := data.DefaultPackIDs()
	fallback := defaults[mode]
	if fallback == "" {
		packs := data.PacksByMode(mode)
		if len(packs) > 0 {
			fallback = packs[0].ID
		}
	}

	s.mu.RLock()
	st := s.stats[deviceID]
	s.mu.RUnlock()

	type scored struct {
		id    string
		score float64
	}
	var list []scored
	hasPersonal := false
	for _, p := range data.PacksByMode(mode) {
		sc := 0.15
		if st != nil {
			if st.ListenSec[p.ID] > 0 || st.PlayCount[p.ID] > 0 {
				hasPersonal = true
			}
			sc += math.Log1p(st.ListenSec[p.ID]/60) * 0.45
			sc += float64(st.PlayCount[p.ID]) * 0.05
			sc -= float64(st.EarlySwitch[p.ID]) * 0.12
		}
		if hourLocal >= 22 || hourLocal < 5 {
			if p.ID == "sleep-soft-rain" || p.ID == "relax-still-air" {
				sc += 0.2
			}
		}
		if hourLocal >= 9 && hourLocal < 18 && mode == models.ModeFocus {
			if p.ID == "focus-clear" {
				sc += 0.12
			}
		}
		list = append(list, scored{id: p.ID, score: sc})
	}
	if len(list) == 0 {
		return RecommendResult{PackID: fallback, Score: 0, Source: "default", Reason: "empty_catalog"}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	best := list[0]

	night := hourLocal >= 22 || hourLocal < 5
	if !hasPersonal && !night {
		return RecommendResult{PackID: fallback, Score: 0.1, Source: "default", Reason: "mode_default"}
	}

	reason := "heuristic"
	source := "rules"
	if hasPersonal {
		reason = "dwell_affinity"
		if night {
			reason = "night_affinity+dwell"
		}
	} else if night {
		reason = "night_affinity"
	}

	return RecommendResult{PackID: best.id, Score: round2(best.score), Source: source, Reason: reason}
}

func (s *Store) Personalize(deviceID string, packID string, hourLocal int) PersonalizeResult {
	delta := ProfileDelta{}
	reason := "neutral"
	source := "rules"

	if hourLocal >= 22 || hourLocal < 5 {
		delta = ProfileDelta{
			Energy: -0.04, Brightness: -0.05, Masking: 0.05, Space: 0.05, EventDensity: -0.03,
		}
		reason = "late_night_soften"
	} else if hourLocal >= 9 && hourLocal < 18 {
		if p, ok := data.PackByID(packID); ok && p.Mode == models.ModeFocus {
			delta = ProfileDelta{
				Energy: 0.03, Brightness: 0.02, Masking: 0, Space: -0.02, EventDensity: -0.04,
			}
			reason = "workday_focus_clarity"
		} else {
			reason = "daytime_neutral"
			source = "default"
		}
	} else {
		source = "default"
		reason = "no_adjustment"
	}

	// if user dwells long on a pack, slight comfort bias
	s.mu.RLock()
	st := s.stats[deviceID]
	s.mu.RUnlock()
	if st != nil && st.ListenSec[packID] > 180 {
		delta.Energy -= 0.02
		delta.Space += 0.03
		delta.Brightness -= 0.02
		if source == "default" {
			source = "rules"
		}
		reason = reason + "+long_dwell_comfort"
	}

	return PersonalizeResult{
		PackID:       packID,
		ProfileDelta: delta,
		Source:       source,
		Reason:       reason,
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// HourLocal helper for handlers when client omits hour.
func HourLocal() int {
	return time.Now().Hour()
}
