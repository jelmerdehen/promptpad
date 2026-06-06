package paste

// Copy puts content on both X selections without typing it.
// Does NOT restore the previous clipboard — pick semantics: user
// explicitly asked for this snippet to land in clipboard.
func Copy(content string) error {
	if err := xclipIn("clipboard", content); err != nil {
		return err
	}
	return xclipIn("primary", content)
}

// ReleaseSuper tells X to forget any held Super_L/Super_R so a
// follow-up Enter doesn't combine with the still-held mod key.
func ReleaseSuper() error {
	return xdotool("keyup", "Super_L", "Super_R")
}
