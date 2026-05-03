package builtin

import (
	"flag"
	"sync"
	"time"
)

// FlagType represents the type of a flag for completion purposes.
type FlagType string

const (
	FlagBool     FlagType = "bool"
	FlagString   FlagType = "string"
	FlagInt      FlagType = "int"
	FlagDuration FlagType = "duration"
)

// FlagMeta stores completion-relevant metadata for a single flag.
type FlagMeta struct {
	Long           string       // long name, e.g. "recursive"
	Short          string       // short name, e.g. "r" (without dash)
	Desc           string       // description for completion display
	Type           FlagType     // bool, string, int, duration
	ValueCompleter ArgCompleter // optional: completes the value for this flag
}

// ArgCompleter provides dynamic completion for positional arguments.
// argIndex is the 0-based index of the positional argument being completed.
type ArgCompleter func(cmdName string, argIndex int, prefix string) []string

// CommandMeta stores all flag metadata and optional arg completer for a command.
type CommandMeta struct {
	Flags     []*FlagMeta
	Completer ArgCompleter
}

var (
	metaMu      sync.RWMutex
	commandMeta = make(map[string]*CommandMeta)
)

// RegisterMeta returns the CommandMeta for the given command name.
// Creates a new one if it doesn't exist. Idempotent.
func RegisterMeta(name string) *CommandMeta {
	metaMu.Lock()
	defer metaMu.Unlock()
	if m, ok := commandMeta[name]; ok {
		return m
	}
	m := &CommandMeta{}
	commandMeta[name] = m
	return m
}

// GetMeta returns the CommandMeta for the given command name, or nil.
func GetMeta(name string) *CommandMeta {
	metaMu.RLock()
	defer metaMu.RUnlock()
	return commandMeta[name]
}

// SetArgCompleter sets the positional argument completer for a command.
func SetArgCompleter(name string, c ArgCompleter) {
	m := RegisterMeta(name)
	m.Completer = c
}

// hasFlag checks if a flag with the given long name is already registered.
func hasFlag(m *CommandMeta, long string) bool {
	for _, f := range m.Flags {
		if f.Long == long {
			return true
		}
	}
	return false
}

// RegisterFlag manually registers a flag's metadata.
// Use this for custom flag.Value types where BoolFlag/StringFlag can't be used.
func (m *CommandMeta) RegisterFlag(long, short string, desc string, typ FlagType) {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{
			Long:  long,
			Short: short,
			Desc:  desc,
			Type:  typ,
		})
	}
}

// SetFlagCompleter sets a value completer for the flag with the given long name.
func (m *CommandMeta) SetFlagCompleter(long string, c ArgCompleter) {
	for _, f := range m.Flags {
		if f.Long == long {
			f.ValueCompleter = c
			return
		}
	}
}

// FindFlagByToken returns the FlagMeta matching the given token (with or without dashes).
func (m *CommandMeta) FindFlagByToken(token string) *FlagMeta {
	name := token
	for len(name) > 0 && name[0] == '-' {
		name = name[1:]
	}
	for _, f := range m.Flags {
		if f.Long == name || f.Short == name {
			return f
		}
	}
	return nil
}

// BoolFlag registers a bool flag with both the FlagSet and metadata.
// Returns a pointer to the bool value. Short and long names both bind to the same variable.
func BoolFlag(fs *flag.FlagSet, m *CommandMeta, long, short string, def bool, desc string) *bool {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagBool})
	}
	var target bool
	fs.BoolVar(&target, long, def, desc)
	if short != "" && short != long {
		fs.BoolVar(&target, short, def, desc)
	}
	return &target
}

// StringFlag registers a string flag with both the FlagSet and metadata.
func StringFlag(fs *flag.FlagSet, m *CommandMeta, long, short string, def string, desc string) *string {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagString})
	}
	var target string
	fs.StringVar(&target, long, def, desc)
	if short != "" && short != long {
		fs.StringVar(&target, short, def, desc)
	}
	return &target
}

// IntFlag registers an int flag with both the FlagSet and metadata.
func IntFlag(fs *flag.FlagSet, m *CommandMeta, long, short string, def int, desc string) *int {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagInt})
	}
	var target int
	fs.IntVar(&target, long, def, desc)
	if short != "" && short != long {
		fs.IntVar(&target, short, def, desc)
	}
	return &target
}

// DurationFlag registers a duration flag with both the FlagSet and metadata.
func DurationFlag(fs *flag.FlagSet, m *CommandMeta, long, short string, def time.Duration, desc string) *time.Duration {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagDuration})
	}
	var target time.Duration
	fs.DurationVar(&target, long, def, desc)
	if short != "" && short != long {
		fs.DurationVar(&target, short, def, desc)
	}
	return &target
}

// BoolFlagVar registers a bool flag bound to an existing variable.
// Both short and long names bind to the same target.
func BoolFlagVar(fs *flag.FlagSet, m *CommandMeta, target *bool, long, short string, def bool, desc string) {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagBool})
	}
	fs.BoolVar(target, long, def, desc)
	if short != "" && short != long {
		fs.BoolVar(target, short, def, desc)
	}
}

// StringFlagVar registers a string flag bound to an existing variable.
func StringFlagVar(fs *flag.FlagSet, m *CommandMeta, target *string, long, short string, def string, desc string) {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagString})
	}
	fs.StringVar(target, long, def, desc)
	if short != "" && short != long {
		fs.StringVar(target, short, def, desc)
	}
}

// IntFlagVar registers an int flag bound to an existing variable.
func IntFlagVar(fs *flag.FlagSet, m *CommandMeta, target *int, long, short string, def int, desc string) {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagInt})
	}
	fs.IntVar(target, long, def, desc)
	if short != "" && short != long {
		fs.IntVar(target, short, def, desc)
	}
}

// DurationFlagVar registers a duration flag bound to an existing variable.
func DurationFlagVar(fs *flag.FlagSet, m *CommandMeta, target *time.Duration, long, short string, def time.Duration, desc string) {
	if !hasFlag(m, long) {
		m.Flags = append(m.Flags, &FlagMeta{Long: long, Short: short, Desc: desc, Type: FlagDuration})
	}
	fs.DurationVar(target, long, def, desc)
	if short != "" && short != long {
		fs.DurationVar(target, short, def, desc)
	}
}

// LongFlags returns all long flag names for a command.
func (m *CommandMeta) LongFlags() []string {
	names := make([]string, 0, len(m.Flags))
	for _, f := range m.Flags {
		names = append(names, f.Long)
	}
	return names
}

// FindFlag returns the FlagMeta matching the given token (with or without dashes).
func (m *CommandMeta) FindFlag(token string) *FlagMeta {
	// Strip leading dashes
	name := token
	for len(name) > 0 && name[0] == '-' {
		name = name[1:]
	}
	for _, f := range m.Flags {
		if f.Long == name || f.Short == name {
			return f
		}
	}
	return nil
}
