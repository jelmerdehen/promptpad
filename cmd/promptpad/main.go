package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/jelmer/promptpad/internal/db"
	"github.com/jelmer/promptpad/internal/paste"
	"github.com/jelmer/promptpad/internal/snippets"
)

const help = `promptpad — numpad prompt snippets with usage tracking

Usage: promptpad <command> [args]

  list              show all slots: count, last-used, title
  stats             same as list but sorted by count ascending
  use N             paste snippet N (logs to db, fixes Super mod-state)
  pick              rofi/dmenu picker → copy snippet to clipboard (no paste)
  show N            print snippet N to stdout
  edit N            open snippet N in $EDITOR
  title N "..."     set title for slot N in index.txt
  reset [N]         zero usage counters (all, or one slot)
  path              print snippet dir + db path
  help              this message
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(help)
		return
	}
	store := snippets.Store{Dir: snippets.DefaultDir()}
	args := os.Args[2:]
	var err error
	switch os.Args[1] {
	case "list":
		err = cmdList(store)
	case "stats":
		err = cmdStats(store)
	case "use":
		err = cmdUse(store, args)
	case "pick":
		err = cmdPick(store)
	case "show":
		err = cmdShow(store, args)
	case "edit":
		err = cmdEdit(store, args)
	case "title":
		err = cmdTitle(store, args)
	case "reset":
		err = cmdReset(args)
	case "path":
		err = cmdPath(store)
	case "help", "-h", "--help":
		fmt.Print(help)
	default:
		die("unknown command: %s (try: help)", os.Args[1])
	}
	if err != nil {
		die("%v", err)
	}
}

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "promptpad: "+format+"\n", a...)
	os.Exit(1)
}

func parseSlot(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 9 {
		return 0, fmt.Errorf("slot must be 0-9, got: %q", s)
	}
	return n, nil
}

func openDB() (*db.DB, error) { return db.Open() }

func printRow(slot, count int, last, title string) {
	fmt.Printf("%-4d %-7d %-21s %s\n", slot, count, last, title)
}

func printHeader() {
	fmt.Printf("%-4s %-7s %-21s %s\n", "SLOT", "COUNT", "LAST USED", "TITLE")
	fmt.Printf("%-4s %-7s %-21s %s\n", "----", "-----", "---------", "-----")
}

func cmdList(s snippets.Store) error {
	d, err := openDB()
	if err != nil {
		return err
	}
	defer d.Close()
	titles := s.Titles()
	printHeader()
	for n := 0; n <= 9; n++ {
		st, err := d.Stat(n)
		if err != nil {
			return err
		}
		printRow(n, st.Count, st.Last, titles[n])
	}
	return nil
}

func cmdStats(s snippets.Store) error {
	d, err := openDB()
	if err != nil {
		return err
	}
	defer d.Close()
	titles := s.Titles()
	stats := make([]db.Stat, 0, 10)
	for n := 0; n <= 9; n++ {
		st, err := d.Stat(n)
		if err != nil {
			return err
		}
		stats = append(stats, st)
	}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].Count < stats[j].Count })
	fmt.Println("Sorted by use count (low first — ditch candidates on top):")
	fmt.Println()
	printHeader()
	for _, st := range stats {
		printRow(st.Slot, st.Count, st.Last, titles[st.Slot])
	}
	return nil
}

func cmdUse(s snippets.Store, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("use needs slot")
	}
	n, err := parseSlot(args[0])
	if err != nil {
		return err
	}
	content, err := s.Read(n)
	if err != nil {
		_ = exec.Command("notify-send", "promptpad", "Missing: "+s.Path(n)).Run()
		return err
	}
	if err := paste.Paste(content); err != nil {
		return err
	}
	d, err := openDB()
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Log(n, s.Hash(n))
}

func cmdPick(s snippets.Store) error {
	d, err := openDB()
	if err != nil {
		return err
	}
	defer d.Close()
	titles := s.Titles()

	var menu bytes.Buffer
	for n := 0; n <= 9; n++ {
		st, _ := d.Stat(n)
		title := titles[n]
		if title == "" {
			title = "(empty)"
		}
		fmt.Fprintf(&menu, "%d  [%d]  %s\n", n, st.Count, title)
	}

	picker, args := pickerCmd()
	cmd := exec.Command(picker, args...)
	cmd.Stdin = &menu
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		// User cancelled (Esc) — no-op, no error message.
		_ = paste.ReleaseSuper()
		return nil
	}
	sel := strings.TrimSpace(string(out))
	if sel == "" {
		return nil
	}
	n, err := strconv.Atoi(strings.Fields(sel)[0])
	if err != nil || n < 0 || n > 9 {
		return fmt.Errorf("bad picker output: %q", sel)
	}
	content, err := s.Read(n)
	if err != nil {
		return err
	}
	if err := paste.Copy(content); err != nil {
		return err
	}
	_ = paste.ReleaseSuper()
	_ = exec.Command("notify-send", "-t", "1500", "promptpad",
		fmt.Sprintf("Copied #%d: %s", n, titles[n])).Run()
	return d.Log(n, s.Hash(n))
}

func pickerCmd() (string, []string) {
	if _, err := exec.LookPath("rofi"); err == nil {
		return "rofi", []string{"-dmenu", "-i", "-p", "promptpad",
			"-theme-str", "window { width: 50%; } listview { lines: 10; }"}
	}
	return "dmenu", []string{"-i", "-p", "promptpad", "-l", "10"}
}

func cmdShow(s snippets.Store, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("show needs slot")
	}
	n, err := parseSlot(args[0])
	if err != nil {
		return err
	}
	c, err := s.Read(n)
	if err != nil {
		return err
	}
	fmt.Print(c)
	return nil
}

func cmdEdit(s snippets.Store, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("edit needs slot")
	}
	n, err := parseSlot(args[0])
	if err != nil {
		return err
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, s.Path(n))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func cmdTitle(s snippets.Store, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("title needs slot and string")
	}
	n, err := parseSlot(args[0])
	if err != nil {
		return err
	}
	return s.SetTitle(n, args[1])
}

func cmdReset(args []string) error {
	d, err := openDB()
	if err != nil {
		return err
	}
	defer d.Close()
	if len(args) >= 1 {
		n, err := parseSlot(args[0])
		if err != nil {
			return err
		}
		if err := d.Reset(&n); err != nil {
			return err
		}
		fmt.Printf("reset slot %d\n", n)
		return nil
	}
	if err := d.Reset(nil); err != nil {
		return err
	}
	fmt.Println("reset all slots")
	return nil
}

func cmdPath(s snippets.Store) error {
	d, err := openDB()
	if err != nil {
		return err
	}
	defer d.Close()
	fmt.Println(s.Dir)
	fmt.Println("db:", d.Path())
	return nil
}
