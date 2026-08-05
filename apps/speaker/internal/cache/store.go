package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/harc/soundscape/apps/speaker/internal/cloud"
	"github.com/harc/soundscape/apps/speaker/internal/models"
)

// Store keeps downloaded stems under root with a soft byte budget (2G device).
type Store struct {
	Root      string
	MaxBytes  int64
	mu        sync.Mutex
	cloud     *cloud.Client
}

type metaFile struct {
	PackID  string            `json:"pack_id"`
	Version string            `json:"version"`
	Files   map[string]string `json:"files"` // layer_id -> relative path
	Bytes   int64             `json:"bytes"`
}

func New(root string, maxBytes int64, c *cloud.Client) *Store {
	_ = os.MkdirAll(root, 0o755)
	return &Store{Root: root, MaxBytes: maxBytes, cloud: c}
}

func (s *Store) packDir(packID, version string) string {
	return filepath.Join(s.Root, "packs", packID, version)
}

func (s *Store) EnsurePack(packID string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	manifest, err := s.cloud.Manifest(packID)
	if err != nil {
		return nil, err
	}
	dir := s.packDir(packID, manifest.Version)
	metaPath := filepath.Join(dir, "meta.json")
	if b, err := os.ReadFile(metaPath); err == nil {
		var m metaFile
		if json.Unmarshal(b, &m) == nil && m.Version == manifest.Version && len(m.Files) == len(manifest.Files) {
			abs := map[string]string{}
			ok := true
			for layer, rel := range m.Files {
				p := filepath.Join(s.Root, rel)
				if _, err := os.Stat(p); err != nil {
					ok = false
					break
				}
				abs[layer] = p
			}
			if ok {
				return abs, nil
			}
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	files := map[string]string{}
	var total int64
	for _, f := range manifest.Files {
		data, err := s.cloud.Download(f.Path)
		if err != nil {
			return nil, err
		}
		name := f.LayerID + ".opus"
		dst := filepath.Join(dir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(s.Root, dst)
		files[f.LayerID] = rel
		total += int64(len(data))
	}
	meta := metaFile{PackID: packID, Version: manifest.Version, Files: files, Bytes: total}
	raw, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, raw, 0o644); err != nil {
		return nil, err
	}
	if err := s.evictLocked(); err != nil {
		return nil, err
	}
	abs := map[string]string{}
	for layer, rel := range files {
		abs[layer] = filepath.Join(s.Root, rel)
	}
	return abs, nil
}

func (s *Store) usageLocked() (int64, []string, error) {
	var total int64
	var dirs []string
	root := filepath.Join(s.Root, "packs")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() && filepath.Base(path) != "packs" {
			// collect version dirs that contain meta.json
			if _, err := os.Stat(filepath.Join(path, "meta.json")); err == nil {
				dirs = append(dirs, path)
			}
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, dirs, nil
}

func (s *Store) evictLocked() error {
	if s.MaxBytes <= 0 {
		return nil
	}
	for {
		total, dirs, err := s.usageLocked()
		if err != nil {
			return err
		}
		if total <= s.MaxBytes || len(dirs) == 0 {
			return nil
		}
		// oldest mtime first
		sort.Slice(dirs, func(i, j int) bool {
			ii, _ := os.Stat(dirs[i])
			jj, _ := os.Stat(dirs[j])
			if ii == nil || jj == nil {
				return i < j
			}
			return ii.ModTime().Before(jj.ModTime())
		})
		victim := dirs[0]
		fmt.Printf("cache evict %s\n", victim)
		if err := os.RemoveAll(victim); err != nil {
			return err
		}
	}
}

func ApplyDelta(p models.Pack, d models.ProfileDelta) models.Pack {
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	ep := p.EngineProfile
	ep.Energy = clamp(ep.Energy + d.Energy)
	ep.Brightness = clamp(ep.Brightness + d.Brightness)
	ep.Masking = clamp(ep.Masking + d.Masking)
	ep.Space = clamp(ep.Space + d.Space)
	ep.EventDensity = clamp(ep.EventDensity + d.EventDensity)
	p.EngineProfile = ep
	return p
}
