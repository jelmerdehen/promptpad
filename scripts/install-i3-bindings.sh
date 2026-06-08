#!/bin/bash
# Append promptpad bindcode block to an i3 config if not already present.
# Idempotent. Usage: install-i3-bindings.sh [/path/to/i3.config]
set -euo pipefail

CFG=${1:-${XDG_CONFIG_HOME:-$HOME/.config}/i3/config}
PROMPTPAD=${PROMPTPAD:-$(command -v promptpad || true)}

[ -f "$CFG" ] || { echo "i3 config not found: $CFG" >&2; exit 1; }
[ -n "$PROMPTPAD" ] || { echo "promptpad not on PATH — run 'make install' first" >&2; exit 1; }

if grep -q "promptpad use 0" "$CFG"; then
    echo "promptpad bindings already present in $CFG"
    exit 0
fi

cat >> "$CFG" <<EOF

# promptpad — added $(date -Iseconds)
# Super+KP_N → paste snippet N (uses xdotool, modifiers suppress KP keysyms so bindcode)
bindcode Mod4+90 exec --no-startup-id $PROMPTPAD use 0
bindcode Mod4+87 exec --no-startup-id $PROMPTPAD use 1
bindcode Mod4+88 exec --no-startup-id $PROMPTPAD use 2
bindcode Mod4+89 exec --no-startup-id $PROMPTPAD use 3
bindcode Mod4+83 exec --no-startup-id $PROMPTPAD use 4
bindcode Mod4+84 exec --no-startup-id $PROMPTPAD use 5
bindcode Mod4+85 exec --no-startup-id $PROMPTPAD use 6
bindcode Mod4+79 exec --no-startup-id $PROMPTPAD use 7
bindcode Mod4+80 exec --no-startup-id $PROMPTPAD use 8
bindcode Mod4+81 exec --no-startup-id $PROMPTPAD use 9
bindcode Mod4+91 exec --no-startup-id $PROMPTPAD pick
EOF

echo "Appended bindings to $CFG"
echo "Reload i3: i3-msg reload"
