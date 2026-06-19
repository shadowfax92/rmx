package rmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const sessionListFormat = "#{session_name}\t#{session_windows}\t#{session_attached}\t#{session_created}\t#{session_activity}"

type Session struct {
	Name         string
	Windows      int
	Attached     bool
	CreatedAt    time.Time
	LastActiveAt time.Time
}

type Runner interface {
	Run(ctx context.Context, binary string, args ...string) (string, error)
	RunInteractive(ctx context.Context, binary string, args ...string) error
}

type Client struct {
	Binary string
	Runner Runner
}

type ExecRunner struct{}

func DefaultClient() Client {
	return Client{Binary: "rmux", Runner: ExecRunner{}}
}

// ListSessions reads rmux's formatted session inventory and returns newest-active sessions first.
func (c Client) ListSessions(ctx context.Context) ([]Session, error) {
	out, err := c.runner().Run(ctx, c.binary(), "list-sessions", "-F", sessionListFormat)
	if err != nil {
		if isNoSessions(err) {
			return nil, nil
		}
		return nil, err
	}
	sessions, err := parseSessions(out)
	if err != nil {
		return nil, err
	}
	SortSessions(sessions)
	return sessions, nil
}

// CapturePane returns the active pane output for a session, optionally limited to the last N lines.
func (c Client) CapturePane(ctx context.Context, name string, lineLimit int) (string, error) {
	args := []string{"capture-pane", "-p", "-t", name}
	if lineLimit > 0 {
		args = append(args, "-S", "-"+strconv.Itoa(lineLimit), "-E", "-1")
	}
	return c.runner().Run(ctx, c.binary(), args...)
}

func (c Client) AttachSession(ctx context.Context, name string) error {
	return c.runner().RunInteractive(ctx, c.binary(), "attach-session", "-t", name)
}

func (c Client) KillSession(ctx context.Context, name string) error {
	_, err := c.runner().Run(ctx, c.binary(), "kill-session", "-t", "="+name)
	return err
}

// SendText writes literal text to the target session's active pane.
func (c Client) SendText(ctx context.Context, name string, text string) error {
	_, err := c.runner().Run(ctx, c.binary(), "send-keys", "-l", "-t", name, text)
	return err
}

// SendEnter presses Enter in the target session's active pane.
func (c Client) SendEnter(ctx context.Context, name string) error {
	_, err := c.runner().Run(ctx, c.binary(), "send-keys", "-t", name, "Enter")
	return err
}

func (ExecRunner) Run(ctx context.Context, binary string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimRight(string(out), "\n")
	if err != nil {
		return "", fmt.Errorf("%s %s: %s (%w)", binary, strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return text, nil
}

func (ExecRunner) RunInteractive(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SortSessions(sessions []Session) {
	sort.Slice(sessions, func(i, j int) bool {
		left := sessions[i]
		right := sessions[j]
		if !left.LastActiveAt.Equal(right.LastActiveAt) {
			return left.LastActiveAt.After(right.LastActiveAt)
		}
		return left.Name < right.Name
	})
}

func parseSessions(out string) ([]Session, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	sessions := make([]Session, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			return nil, fmt.Errorf("unexpected rmux list-sessions row %q", line)
		}

		windows, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parse window count for %q: %w", parts[0], err)
		}
		created, err := parseUnix(parts[3])
		if err != nil {
			return nil, fmt.Errorf("parse created time for %q: %w", parts[0], err)
		}
		activity, err := parseUnix(parts[4])
		if err != nil {
			return nil, fmt.Errorf("parse activity time for %q: %w", parts[0], err)
		}

		sessions = append(sessions, Session{
			Name:         parts[0],
			Windows:      windows,
			Attached:     parts[2] != "0",
			CreatedAt:    created,
			LastActiveAt: activity,
		})
	}
	return sessions, nil
}

func parseUnix(raw string) (time.Time, error) {
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(seconds, 0), nil
}

func isNoSessions(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no sessions") || strings.Contains(msg, "no server running")
}

func (c Client) binary() string {
	if c.Binary == "" {
		return "rmux"
	}
	return c.Binary
}

func (c Client) runner() Runner {
	if c.Runner == nil {
		return ExecRunner{}
	}
	return c.Runner
}
