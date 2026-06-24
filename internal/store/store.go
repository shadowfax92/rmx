// Package store is rmx's sidecar record of rmux sessions that have exited.
// rmux forgets a session the moment it is killed, so rmx persists a small
// per-session file (metadata + captured output) to keep showing the session as
// "exited" in `rmx ls` and to replay it with `rmx cat` for a while afterwards.
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Retention is how long an exited session stays visible before it is pruned.
const Retention = 24 * time.Hour

// ExitedSession is the metadata recorded when a session exits. The captured
// pane output is stored alongside it (see Store.Output), not inlined here.
type ExitedSession struct {
	Name      string    `json:"name"`
	Windows   int       `json:"windows"`
	CreatedAt time.Time `json:"created_at"`
	ExitedAt  time.Time `json:"exited_at"`
}

// Store reads and writes exited-session records under Dir. A zero Dir disables
// the store: reads return empty and writes return a soft error, so the rest of
// rmx keeps working even when no state directory is available.
type Store struct {
	Dir string
}

// Default resolves the state directory, honoring XDG_STATE_HOME and falling back
// to ~/.local/state/rmx.
func Default() Store {
	return Store{Dir: defaultDir()}
}

func defaultDir() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "rmx")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "rmx")
}

func (s Store) exitedDir() string { return filepath.Join(s.Dir, "exited") }

// recordPaths returns the metadata and log paths for a session. The name is
// url-encoded because session names contain '/' and other path-unsafe runes.
func (s Store) recordPaths(name string) (jsonPath, logPath string) {
	base := filepath.Join(s.exitedDir(), url.QueryEscape(name))
	return base + ".json", base + ".log"
}

// Record persists an exited session and its captured output. The log is written
// before the JSON so a crash mid-write leaves an orphan log (ignored by List)
// rather than a record pointing at missing output.
func (s Store) Record(rec ExitedSession, output string) error {
	if s.Dir == "" {
		return errors.New("rmx: no state directory available")
	}
	if err := os.MkdirAll(s.exitedDir(), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	jsonPath, logPath := s.recordPaths(rec.Name)
	if err := os.WriteFile(logPath, []byte(output), 0o644); err != nil {
		return err
	}
	return os.WriteFile(jsonPath, data, 0o644)
}

// List returns all recorded exited sessions, newest exit first. Unreadable or
// malformed records are skipped so one bad file never breaks the listing.
func (s Store) List() ([]ExitedSession, error) {
	if s.Dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(s.exitedDir())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var out []ExitedSession
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.exitedDir(), entry.Name()))
		if err != nil {
			continue
		}
		var rec ExitedSession
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}

	sort.Slice(out, func(i, j int) bool {
		if !out[i].ExitedAt.Equal(out[j].ExitedAt) {
			return out[i].ExitedAt.After(out[j].ExitedAt)
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Output returns the captured pane output recorded for a session. ok is false
// when no log exists for the name.
func (s Store) Output(name string) (string, bool, error) {
	if s.Dir == "" {
		return "", false, nil
	}
	_, logPath := s.recordPaths(name)
	data, err := os.ReadFile(logPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

// Remove deletes a session's record and log, tolerating already-absent files.
func (s Store) Remove(name string) error {
	if s.Dir == "" {
		return nil
	}
	jsonPath, logPath := s.recordPaths(name)
	for _, path := range []string{logPath, jsonPath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Clear removes every recorded exited session and returns how many were removed.
func (s Store) Clear() (int, error) {
	recs, err := s.List()
	if err != nil {
		return 0, err
	}
	for _, rec := range recs {
		if err := s.Remove(rec.Name); err != nil {
			return 0, err
		}
	}
	return len(recs), nil
}

// Prune removes records whose exit time is at or before cutoff.
func (s Store) Prune(cutoff time.Time) error {
	recs, err := s.List()
	if err != nil {
		return err
	}
	for _, rec := range recs {
		if !rec.ExitedAt.After(cutoff) {
			if err := s.Remove(rec.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
