package paste

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// PasteKey returns the xdotool keysequence used by Paste.
// Default: shift+Insert (legacy + Alacritty PasteSelection).
// Override with PROMPTPAD_KEY, e.g. "ctrl+shift+v".
func PasteKey() string {
	if k := os.Getenv("PROMPTPAD_KEY"); k != "" {
		return k
	}
	return "shift+Insert"
}

// Paste types content via X clipboard + the configured paste key, then
// forces keyup on Super_L/Super_R so a still-held Super after Mod4+KP_N
// doesn't combine with the user's next Enter into $mod+Return.
func Paste(content string) error {
	prevClip := xclipOut("clipboard")
	prevPrim := xclipOut("primary")

	if err := xclipIn("clipboard", content); err != nil {
		return fmt.Errorf("xclip clipboard: %w", err)
	}
	if err := xclipIn("primary", content); err != nil {
		return fmt.Errorf("xclip primary: %w", err)
	}

	time.Sleep(50 * time.Millisecond)

	if err := xdotool("key", "--clearmodifiers", PasteKey()); err != nil {
		return fmt.Errorf("xdotool key %s: %w", PasteKey(), err)
	}

	if err := xdotool("keyup", "Super_L", "Super_R"); err != nil {
		return fmt.Errorf("xdotool keyup: %w", err)
	}

	// Restore old selections via a detached subprocess that survives
	// after main() returns. A bare goroutine would die when the
	// process exits.
	if err := scheduleRestore(prevClip, prevPrim, 1*time.Second); err != nil {
		// Restore is best-effort; don't fail the paste over it.
		fmt.Fprintf(os.Stderr, "promptpad: restore schedule: %v\n", err)
	}
	return nil
}

func scheduleRestore(clip, prim string, after time.Duration) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "_restore", fmt.Sprintf("%d", int(after.Milliseconds())))
	cmd.Stdin = strings.NewReader(clip + "\x00" + prim)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

// RunRestore is the hidden subcommand invoked by scheduleRestore.
// Reads "<clip>\x00<prim>" from stdin, sleeps, restores.
func RunRestore(delayMS int, clip, prim string) {
	time.Sleep(time.Duration(delayMS) * time.Millisecond)
	_ = xclipIn("clipboard", clip)
	_ = xclipIn("primary", prim)
}

func xclipOut(sel string) string {
	out, err := exec.Command("xclip", "-selection", sel, "-o").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func xclipIn(sel, content string) error {
	cmd := exec.Command("xclip", "-selection", sel)
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func xdotool(args ...string) error {
	out, err := exec.Command("xdotool", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
