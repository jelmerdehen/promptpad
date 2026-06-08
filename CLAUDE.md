# promptpad — repo guide

Go CLI that pastes prompt snippets via Super+numpad and tracks usage in
SQLite. Bound from i3 in `/.../linux/ui/c/i3.config`. The numpad
slots 0–9 each hold one snippet; KP_Decimal opens an interactive picker.

## What problem this solves

1. Memorising which numpad slot holds which prompt is unsustainable past
   ~4 slots. Usage counts + the `pick` picker make it tractable.
2. Identifying which slot to repurpose. Before this repo, snippets were
   bash files with no telemetry — `stats` (lowest-count first) is the
   ditch-candidate list.
3. The "stuck-Super" bug after `Mod4+KP_N` (see Mod-state section).

## Layout

| Path | Purpose |
|------|---------|
| `cmd/promptpad/main.go` | Entry point. Subcommand dispatch, all UI text. |
| `internal/db/db.go` | SQLite open/log/stat/reset. `modernc.org/sqlite` (pure Go, no cgo). |
| `internal/snippets/snippets.go` | Snippet file IO + `index.txt` parsing/writing. |
| `internal/paste/paste.go` | `xdotool`/`xclip` orchestration for the `use` flow. |
| `internal/paste/clipboard.go` | Clipboard-only flow used by `pick`, plus `ReleaseSuper`. |
| `snippets/N.txt` (N=0..9) | Snippet content, one file per slot. Edit raw — no escaping. |
| `snippets/index.txt` | `N: title` per line. Read by `list`/`stats`/`pick`. Sorted by slot. |
| `bin/promptpad` | Built binary. Gitignored. |

## Build & install

```
make install
```

Targets: `build`, `install` (symlinks `~/.local/bin/promptpad` →
`$(CURDIR)/bin/promptpad`), `uninstall`, `reinstall`, `doctor`,
`clean`, `run ARGS=...`, `fmt`, `vet`, `test`. Build is single static
binary, ~6 MB. Module is in the parent `/data/p/go.work`.

**Symlink install is load-bearing.** `internal/snippets.DefaultDir`
runs `EvalSymlinks(os.Executable())` then `<dir>/../snippets`. If install
copied the binary into `~/.local/bin` instead, that resolves to
`~/.local/snippets` and the CLI fails to find anything. Don't change to
copy-install without also wiring an explicit snippet path (env, build
tag, or wrapper script).

Runtime deps (exec'd, not linked): `xdotool`, `xclip`, `notify-send`,
`rofi` (or `dmenu` as fallback).

## CLI surface

| Command | Effect |
|---------|--------|
| `list` | All 10 slots: count, last-used, title |
| `stats` | Same, sorted by count ascending — ditch candidates on top |
| `use N` | Paste snippet N (xdotool shift+Insert), log to db, release Super |
| `pick` | rofi/dmenu picker → copy to clipboard (no auto-paste), log to db |
| `show N` | Cat snippet N to stdout |
| `edit N` | Open snippet N in `$EDITOR` |
| `title N "..."` | Set/replace title in `index.txt`, keep file sorted |
| `reset [N]` | Zero usage counters (all, or one slot) |
| `path` | Print snippet dir and db path |
| `doctor` | Print dep paths, DISPLAY, paste key, active window, db path. First stop when binding "doesn't work". |
| `_restore <ms>` | **Hidden.** Reads `<clip>\x00<prim>` from stdin, sleeps, writes them back to both selections. Spawned detached by `use`. |
| `help` | Help text |

### Env

| Var | Default | Purpose |
|-----|---------|---------|
| `PROMPTPAD_KEY` | `shift+Insert` | xdotool keysequence used by `use`. Try `ctrl+shift+v` if the focused app doesn't paste on shift+Insert. |
| `PROMPTPAD_SNIPPETS` | `<exe>/../snippets` (via `EvalSymlinks`) | Override snippet dir. |
| `PROMPTPAD_DB` | `${XDG_DATA_HOME:-~/.local/share}/promptpad/usage.db` | Override db path. |

## Storage

DB at `${XDG_DATA_HOME:-~/.local/share}/promptpad/usage.db`. Override
with `PROMPTPAD_DB`. Schema (auto-init on open):

```sql
CREATE TABLE uses (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  slot  INTEGER NOT NULL,
  ts    TEXT    NOT NULL DEFAULT (datetime('now')),
  hash  TEXT
);
CREATE INDEX idx_uses_slot ON uses(slot);
```

`hash` = first 12 hex chars of `sha256(snippet)` at use time. Lets you
spot when a slot got repurposed mid-history (count vs distinct hashes).

Snippet dir resolution: `PROMPTPAD_SNIPPETS` env → `<exe>/../snippets`
(via `EvalSymlinks`, so `~/.local/bin/promptpad` symlink still finds
the canonical `/data/p/promptpad/snippets/`).

## i3 wiring

Binds live in `/.../linux/ui/c/i3.config`. KP_* keysyms get
suppressed by Mod4 (yields `KP_Insert`/`KP_End` instead of `KP_0`/`KP_1`),
so we use **bindcode** not **bindsym**. Keycodes:

| Keycode | KP key | Bind |
|---------|--------|------|
| 90 | KP_0 | `promptpad use 0` |
| 87 | KP_1 | `promptpad use 1` |
| 88 | KP_2 | `promptpad use 2` |
| 89 | KP_3 | `promptpad use 3` |
| 83 | KP_4 | `promptpad use 4` |
| 84 | KP_5 | `promptpad use 5` |
| 85 | KP_6 | `promptpad use 6` |
| 79 | KP_7 | `promptpad use 7` |
| 80 | KP_8 | `promptpad use 8` |
| 81 | KP_9 | `promptpad use 9` |
| 91 | KP_Decimal/KP_Delete | `promptpad pick` |

Cheatsheet (`Mod+Shift+/`) reads `snippets/index.txt` and rewrites
`promptpad use N` → `paste #N: <title>` in `/.../linux/ui/c/i3-cheatsheet.sh`.

## Clipboard restore: why a detached subprocess

After `use` writes the snippet to clipboard + primary, it must restore
the user's prior selections so their copy buffer isn't permanently
clobbered. Naïve `go func() { time.Sleep(1s); xclip(restore) }()`
**dies** when `main` returns — the goroutine never wakes.

Fix: `scheduleRestore` in `internal/paste/paste.go` spawns the binary as
a detached subprocess (`Setsid: true`) with the hidden `_restore <ms>`
subcommand, piping `<clip>\x00<prim>` to its stdin. That subprocess
sleeps then restores, surviving `use`'s exit. Single-binary, no
external shell call required.

If you change the restore protocol, update both `scheduleRestore`
(producer) and `cmd/promptpad/main.go`'s `_restore` case (consumer).

## Mod-state bug — why `keyup Super_L Super_R` is needed

`use` chain in `internal/paste/paste.go`:

1. Save existing clipboard + primary selections.
2. Write snippet content to both selections.
3. `xdotool key --clearmodifiers shift+Insert` to paste.
4. `xdotool keyup Super_L Super_R` — **the fix**.
5. After 1 s, restore previous selections (background goroutine).

Without step 4: the user physically still holds Super (their finger is
on it from the `Mod4+KP_N` press). `--clearmodifiers` released-and-
restored Super around the synthesised shift+Insert, so X's modifier
state is back to "Super down". The next Enter the user types combines
to `$mod+Return`, which i3 binds to `i3-sensible-terminal` (i3.config
~line 295) — a terminal opens instead of the prompt being submitted.

Step 4 forces an `XTestFakeKeyEvent` keyup for both Super keys. X then
treats Super as released even though the hardware key is still down.
When the user finally releases the physical key, X emits an extra
release event that has no listeners and gets dropped. Harmless.

`pick` reuses `ReleaseSuper` for the same reason — rofi grabs the
keyboard so the bug doesn't manifest during selection, but Super is
still "held" once rofi closes.

## Conventions

- One snippet per slot file, raw text. No shell escaping, no templating.
  Newlines preserved exactly. Trailing newline is part of the paste.
- `index.txt`: `N: title` per line, sorted by N. `title` is a one-line
  human label, no markup. Edit via `promptpad title N "..."` to keep it
  sorted; manual edits are fine if you re-sort.
- Slot 0..9 are the canonical range; `parseSlot` rejects anything else.
  No plan to extend — numpad has 10 digit keys.
- Commits: present tense, lowercase subject, one-line body explaining
  the *why* if it isn't obvious from the diff.

## When to update what

| You changed... | Also update |
|----------------|-------------|
| Snippet content | Nothing — `use`/`pick` reads file directly. |
| Snippet title | `promptpad title N "..."` (writes `index.txt`, sorts). |
| Added/removed a subcommand | `main.go` switch, `help` string, README CLI section, this file. |
| Binding (keycode/mapping) | `/.../linux/ui/c/i3.config`, README, table above. |
| DB schema | `internal/db/db.go` init SQL + this file's schema block. Migration path is "drop the file" — usage logs are not load-bearing. |

## Non-goals

- Snippets longer than what `shift+Insert` paste comfortably handles
  (multi-MB). If needed, switch `use` to `xdotool type --delay` (slow).
- Wayland support. Hard-coded to X11 (xdotool + xclip). Re-evaluate when
  the lux desktop moves off Xorg.
- Cross-host snippet sync. Snippets are in this git repo; sync via git.
  DB is per-host telemetry by design — don't push `usage.db`.

## Debugging "binding doesn't work"

Decision tree:

1. **Run `promptpad doctor`** — confirms deps present and shows active
   window class. If active window is what you expect, deps are fine,
   skip to step 3.
2. **Check db log**: `sqlite3 ~/.local/share/promptpad/usage.db "SELECT
   * FROM uses ORDER BY id DESC LIMIT 5"`. If your KP_N press logged a
   row, the i3 binding fired and `use N` ran to completion — meaning
   xdotool returned exit 0. Problem is paste **delivery**, not firing.
3. **Paste key mismatch**: focused app may not paste on `shift+Insert`.
   Modern terminals (Alacritty, Kitty, Foot, WezTerm) vary. Test with
   `PROMPTPAD_KEY=ctrl+shift+v promptpad use 0` while focused on the
   same app. If that lands, rebind in i3 by wrapping with `env`.
4. **No db row but key was pressed**: i3 didn't fire. Check
   `i3-msg -t get_config | grep promptpad` shows the bindcode lines.
   `i3-msg reload` if you edited config. Verify keycode with
   `xev | grep keycode` while pressing the numpad key.
5. **Wrong focus at press time**: the snippet content lands in
   clipboard + primary regardless of focus. Click the target window,
   then `shift+Insert` manually. If that works, the issue is purely
   "focus moved between press and paste delivery."

## See also

- `/.../linux/ui/c/i3.config` — the bindcode lines.
- `/.../linux/ui/c/i3-cheatsheet.sh` — reads `snippets/index.txt`.
- `/.../doc/CLAUDE.md` — lux desktop state (where this runs).
