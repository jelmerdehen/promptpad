package snippets

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Store struct{ Dir string }

func DefaultDir() string {
	if d := os.Getenv("PROMPTPAD_SNIPPETS"); d != "" {
		return d
	}
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		// bin/promptpad → ../snippets
		return filepath.Join(filepath.Dir(filepath.Dir(exe)), "snippets")
	}
	return "snippets"
}

func (s Store) Path(slot int) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%d.txt", slot))
}

func (s Store) IndexPath() string { return filepath.Join(s.Dir, "index.txt") }

func (s Store) Read(slot int) (string, error) {
	b, err := os.ReadFile(s.Path(slot))
	return string(b), err
}

func (s Store) Hash(slot int) string {
	b, err := os.ReadFile(s.Path(slot))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:6])
}

func (s Store) Titles() map[int]string {
	m := map[int]string{}
	f, err := os.Open(s.IndexPath())
	if err != nil {
		return m
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[:i]))
		if err != nil {
			continue
		}
		m[n] = strings.TrimSpace(line[i+1:])
	}
	return m
}

func (s Store) SetTitle(slot int, title string) error {
	titles := s.Titles()
	titles[slot] = title
	keys := make([]int, 0, len(titles))
	for k := range titles {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%d: %s\n", k, titles[k])
	}
	return os.WriteFile(s.IndexPath(), []byte(b.String()), 0o644)
}
