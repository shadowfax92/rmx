<div align="center">

<img src="assets/wrapux.svg" width="120" alt="wrapux icon">

# wrapux

**An fzf-powered wrapper for rmux sessions.**

*List, attach, remove, and capture rmux sessions without remembering the flags.*

</div>

`wrapux` keeps the rmux command surface close by: it lists sessions with last activity, opens fzf for interactive selection, removes selected sessions, and prints clearly separated captures from one or more sessions.

- **Session inventory** - `wrapux ls` shows every rmux session, window count, exact last-active time, relative age, created time, and attach state.
- **Fast attach** - `wrapux attach` opens an fzf picker with a live capture preview, then attaches to the selected session.
- **Multi-remove** - `wrapux rm` opens multi-select fzf when no session names are provided.
- **Multi-capture** - `wrapux capture` selects sessions with fzf and prints each output under a colored separator.
- **Line limits** - `wrapux capture -l 20` captures the last 20 lines from each selected session.
- **Plain delegation** - wrapux calls rmux directly; it does not keep state of its own.

---

## Install

Requires Go 1.21+, `rmux`, and [fzf](https://github.com/junegunn/fzf).

```sh
git clone <repo-url> wrapux
cd wrapux
make install
```

`make install` builds `wrapux`, installs it to `~/bin/wrapux`, and signs it on macOS.

## Quick Start

```sh
wrapux ls
wrapux attach
wrapux capture -l 20
wrapux rm
```

## Commands

### List

```sh
wrapux ls
wrapux list
wrapux l
```

Shows all rmux sessions, sorted by most recent `#{session_activity}` first.

### Attach

```sh
wrapux attach
wrapux attach codex/feat-example
wrapux a
```

Without a session name, `wrapux attach` opens fzf. The first hidden fzf field is the rmux session name, while the visible row includes the name, window count, last activity, and attached state.

### Remove

```sh
wrapux rm
wrapux rm codex/old-task claude/old-task
wrapux remove
wrapux kill
```

Without names, `wrapux rm` opens fzf in multi-select mode. Selected sessions are removed with `rmux kill-session -t =<session>`.

### Capture

```sh
wrapux capture
wrapux capture codex/feat-example claude/review
wrapux capture -l 20
wrapux cap
```

Without names, `wrapux capture` opens fzf in multi-select mode. Each session is captured with:

```sh
rmux capture-pane -p -t <session> -S -<lines> -E -1
```

The default line count is 80. Use `-l 20` or `--lines 20` to override it.

## Make Targets

```sh
make build
make test
make install
make uninstall
make clean
```

## Notes

`wrapux` gets last activity from rmux's `#{session_activity}` format field. It does not infer activity from captured output or maintain a sidecar timestamp database.
