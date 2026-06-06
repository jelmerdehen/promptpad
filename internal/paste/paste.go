package paste

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Paste types content via X clipboard + shift+Insert, then forces
// keyup on Super_L/Super_R so a still-held Super after Mod4+KP_N
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

	if err := xdotool("key", "--clearmodifiers", "shift+Insert"); err != nil {
		return fmt.Errorf("xdotool key: %w", err)
	}

	if err := xdotool("keyup", "Super_L", "Super_R"); err != nil {
		return fmt.Errorf("xdotool keyup: %w", err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		_ = xclipIn("clipboard", prevClip)
		_ = xclipIn("primary", prevPrim)
	}()
	return nil
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
	return exec.Command("xdotool", args...).Run()
}
