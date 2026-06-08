# promptpad

Numpad-triggered prompt snippets with usage tracking.

Bind `Super+KP_0..9` in your WM to `promptpad use N`. Each use is logged to
SQLite so you can see which slots are dead weight and reclaim them.

## Build & install

```
make install
```

Builds the binary (~6 MB static) and symlinks `~/.local/bin/promptpad →
$(pwd)/bin/promptpad`. The symlink (not a copy) is load-bearing — the
binary resolves `snippets/` via `EvalSymlinks(os.Executable())/../snippets`
back to this repo's `snippets/` dir.

Pure-Go SQLite (`modernc.org/sqlite`), no cgo. Runtime deps (exec'd):
`xdotool`, `xclip`, `notify-send`, `rofi` (or `dmenu`).

i3 bindings: add to your `~/.config/i3/config` (or run
`scripts/install-i3-bindings.sh` for an idempotent append), then
`i3-msg reload`.

i3 binding (KP_* keysyms suppressed by Mod, so use bindcode):

```
bindcode Mod4+90 $exec promptpad use 0
bindcode Mod4+87 $exec promptpad use 1
bindcode Mod4+88 $exec promptpad use 2
bindcode Mod4+89 $exec promptpad use 3
bindcode Mod4+83 $exec promptpad use 4
bindcode Mod4+84 $exec promptpad use 5
bindcode Mod4+85 $exec promptpad use 6
bindcode Mod4+79 $exec promptpad use 7
bindcode Mod4+80 $exec promptpad use 8
bindcode Mod4+81 $exec promptpad use 9
# KP_Decimal (91): rofi picker → snippet to clipboard, no auto-paste
bindcode Mod4+91 $exec promptpad pick
```

## CLI

```
promptpad list             # all slots: title, count, last-used
promptpad stats            # sorted by use count (ascending — easy ditch candidates)
promptpad use N            # paste snippet N, log to db
promptpad pick             # rofi/dmenu picker → copy snippet to clipboard
promptpad edit N           # open snippet N in $EDITOR
promptpad title N "..."    # set/replace title in index.txt
promptpad reset [N]        # zero counters (all, or one slot)
promptpad path             # print snippet dir
promptpad show N           # cat snippet N
promptpad doctor           # check deps, active window, paste key
```

## Troubleshooting

If `use N` fires (db logs it) but nothing pastes, the focused app may
not handle `shift+Insert`. Override:

```
PROMPTPAD_KEY=ctrl+shift+v promptpad use 0
```

Set per-binding in i3:

```
bindcode Mod4+90 exec --no-startup-id env PROMPTPAD_KEY=ctrl+shift+v /data/p/promptpad/bin/promptpad use 0
```

Run `promptpad doctor` to see active window + paste key + deps.

## Layout

```
cmd/promptpad/             # main package
internal/db/               # SQLite (modernc.org/sqlite)
internal/snippets/         # snippet files + index.txt
internal/paste/            # xdotool/xclip orchestration
snippets/
  0.txt … 9.txt            # one file per slot
  index.txt                # "N: title" lines, one per slot
bin/promptpad              # built binary
```

DB at `~/.local/share/promptpad/usage.db`. Schema:

```sql
CREATE TABLE uses (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  slot  INTEGER NOT NULL,
  ts    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  hash  TEXT
);
```

## Mod-state fix

After `xdotool key --clearmodifiers shift+Insert`, the script also issues
`xdotool keyup Super_L Super_R`. Without this, holding Super through the
numpad press leaves Super "down" in X's view — the next Enter then triggers
`$mod+Return` (terminal launch in i3) instead of submitting the prompt.
