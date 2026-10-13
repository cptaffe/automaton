package hotkey

import (
	"slices"
	"strconv"
)

type Keyboard []string

var US = us()

func us() Keyboard {
	var kb Keyboard
	for c := 'a'; c <= 'z'; c++ {
		kb = append(kb, string(c))
	}
	for c := '0'; c <= '9'; c++ {
		kb = append(kb, string(c))
	}
	for _, c := range "-=[]\\;',./`" {
		kb = append(kb, string(c))
	}
	kb = append(kb, "return", "tab", "space", "delete", "escape", "forwarddelete",
		"home", "end", "pageup", "pagedown", "left", "right", "down", "up", "help", "clear")
	for i := 1; i <= 16; i++ {
		kb = append(kb, "f"+strconv.Itoa(i))
	}
	return kb
}

func (kb Keyboard) Has(name string) bool { return slices.Contains(kb, name) }

func (kb Keyboard) chords() []string {
	out := make([]string, 0, len(kb)*(1<<len(modifiers)))
	for mask := range 1 << len(modifiers) {
		var mods []string
		for i, m := range modifiers {
			if mask&(1<<i) != 0 {
				mods = append(mods, m)
			}
		}
		for _, k := range kb {
			out = append(out, Key(mods, k))
		}
	}
	return out
}
