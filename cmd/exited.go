package cmd

import (
	"fmt"
	"io"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"
)

// loadExited prunes stale records, then returns the surviving exited sessions.
// Store errors are non-fatal: they warn and degrade to "no exited sessions" so
// the live listing still renders.
func loadExited(st store.Store, now time.Time, warn io.Writer) []store.ExitedSession {
	if err := st.Prune(now.Add(-store.Retention)); err != nil {
		fmt.Fprintf(warn, "rmx: prune exited sessions: %v\n", err)
	}
	exited, err := st.List()
	if err != nil {
		fmt.Fprintf(warn, "rmx: list exited sessions: %v\n", err)
		return nil
	}
	return exited
}

// exitedToSession renders a recorded exit as a Session for the list/picker,
// carrying the exit time as its last-active time.
func exitedToSession(rec store.ExitedSession) rmux.Session {
	return rmux.Session{
		Name:         rec.Name,
		Windows:      rec.Windows,
		Attached:     false,
		Exited:       true,
		CreatedAt:    rec.CreatedAt,
		LastActiveAt: rec.ExitedAt,
	}
}

// mergeSessions overlays exited records onto the live rmux list, newest-active
// first. A live session wins over an exited record of the same name (the session
// was recreated after it exited).
func mergeSessions(live []rmux.Session, exited []store.ExitedSession) []rmux.Session {
	liveNames := make(map[string]bool, len(live))
	for _, session := range live {
		liveNames[session.Name] = true
	}

	merged := append([]rmux.Session(nil), live...)
	for _, rec := range exited {
		if liveNames[rec.Name] {
			continue
		}
		merged = append(merged, exitedToSession(rec))
	}
	rmux.SortSessions(merged)
	return merged
}
