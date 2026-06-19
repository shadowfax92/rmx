<div align="center">

<img src="assets/rmx.svg" width="120" alt="rmx icon">

# rmx

**An fzf-powered manager for rmux sessions.**

*List, attach, exit, remove, and print rmux session output without remembering the flags.*

</div>

`rmx` keeps the rmux command surface close by: it lists sessions with last activity, opens fzf for interactive selection, removes selected sessions, and prints clearly separated output from one or more sessions.

- **Session inventory** - `rmx ls` shows every rmux session, window count, exact last-active time, relative age, created time, and attach state.
- **Fast attach** - `rmx attach` opens an fzf picker with a live capture preview, then attaches to the selected session.
- **Exit current session** - `rmx exit` closes the current rmux session from inside its pane and removes it from the list.
- **Multi-remove** - `rmx rm` opens multi-select fzf when no session names are provided.
- **Multi-cat** - `rmx cat` selects sessions with fzf and prints each output under a colored separator.
- **Multi-tail** - `rmx tail` selects sessions with fzf and follows newly appended output with colored prefixes.
- **Send input** - `rmx send text` writes literal text, and `rmx send enter` presses Enter in a session.
- **Line limits** - `rmx cat -l 20` prints the last 20 lines from each selected session.
- **Plain delegation** - rmx calls rmux directly; it does not keep state of its own.

---

## Install

Requires Go 1.21+, `rmux`, and [fzf](https://github.com/junegunn/fzf). [fish](https://fishshell.com) is optional, for the `rmx` shortcut.

```sh
git clone https://github.com/shadowfax92/rmx.git
cd rmx
make install
```

`make install` builds `rmx`, installs it to `~/bin/rmx`, signs it on macOS, and installs the fish `rmx` helper to `~/.config/fish/functions/rmx.fish`.

## Quick Start

```sh
rmx ls
rmx attach
rmx exit
rmx cat -l 20
rmx tail
rmx send text -t codex/feat-example 'echo hello from rmx'
rmx send enter -t codex/feat-example
rmx rm
```

## Fish shortcuts

The optional fish function adds short verbs on top of the `rmx` binary. `make install` drops it at `~/.config/fish/functions/rmx.fish` (or run `make fish` on its own). Bare `rmx` lists; anything it doesn't recognize forwards straight to the binary.

```fish
rmx              # list sessions (same as rmx ls)
rmx l            # list     (l / ls / list)
rmx a            # attach   (a / attach)
rmx e            # exit     (e / exit / quit)
rmx c            # cat      (c / cat / cap / capture)
rmx t            # tail     (t / tail / follow)
rmx s            # send     (s / send)
rmx text         # send text
rmx enter        # send enter
rmx k            # remove   (rm / k / kill / remove)
rmx c -l 20      # flags and session names pass through
rmx --help       # forwarded to the binary
```

## Commands

### List

```sh
rmx ls
rmx list
rmx l
```

Shows all rmux sessions, sorted by most recent `#{session_activity}` first.

### Attach

```sh
rmx attach
rmx attach codex/feat-example
rmx a
```

Without a session name, `rmx attach` opens fzf. The first hidden fzf field is the rmux session name, while the visible row includes the name, window count, last activity, and attached state.

### Exit

```sh
rmx exit
rmx e
rmx quit
```

Run from inside an rmux pane to close the current session. The command resolves the current rmux session name and removes that exact session from the list.

### Remove

```sh
rmx rm
rmx rm codex/old-task claude/old-task
rmx remove
rmx kill
```

Without names, `rmx rm` opens fzf in multi-select mode. Selected sessions are removed with `rmux kill-session -t =<session>`.

### Cat

```sh
rmx cat
rmx cat codex/feat-example claude/review
rmx cat -l 20
rmx capture
rmx cap
```

Without names, `rmx cat` opens fzf in multi-select mode. Each session is captured with:

```sh
rmux capture-pane -p -t <session> -S -<lines> -E -1
```

The default line count is 80. Use `-l 20` or `--lines 20` to override it.

### Tail

```sh
rmx tail
rmx tail codex/feat-example claude/review
rmx follow
```

Without names, `rmx tail` opens fzf in multi-select mode. It treats the first capture as the baseline, then polls every 5 seconds and prints newly appended output with a colored `[session]` prefix for each selected rmux session.

### Send

```sh
rmx send text -t codex/feat-example 'echo hello from rmx'
rmx send enter -t codex/feat-example
rmx text -t codex/feat-example 'echo shortcut'
rmx enter -t codex/feat-example
```

`rmx send text` sends literal text with:

```sh
rmux send-keys -l -t <session> <text>
```

`rmx send enter` sends the Enter key with:

```sh
rmux send-keys -t <session> Enter
```

Omit `-t/--target` to pick a session with fzf.

## Make Targets

```sh
make build
make test
make install      # binary + rmx fish helper
make uninstall    # removes both
make fish         # install just the rmx fish helper
make clean
```

## Notes

`rmx` gets last activity from rmux's `#{session_activity}` format field. It does not infer activity from captured output or maintain a sidecar timestamp database.
