# promptpad

Numpad-triggered prompt snippets with usage tracking.

Bind `Super+KP_0..9` in your WM to `promptpad use N`. Each use is logged to
SQLite so you can see which slots are dead weight and reclaim them.

## Install

```
git clone … /data/p/promptpad
ln -s /data/p/promptpad/bin/promptpad ~/.local/bin/promptpad
```

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
```

## CLI

```
promptpad list             # all slots: title, count, last-used
promptpad stats            # sorted by use count (ascending — easy ditch candidates)
promptpad use N            # paste snippet N, log to db
promptpad edit N           # open snippet N in $EDITOR
promptpad title N "..."    # set/replace title in index.txt
promptpad reset [N]        # zero counters (all, or one slot)
promptpad path             # print snippet dir
promptpad show N           # cat snippet N
```

## Layout

```
snippets/
  0.txt … 9.txt    # one file per slot
  index.txt        # "N: title" lines, one per slot
bin/promptpad      # CLI
lib/db.sh          # sqlite helpers
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
