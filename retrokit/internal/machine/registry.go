package machine

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]Machine{}
)

// Register adds a machine to the registry. It is called from each machine
// package's init function.
func Register(m Machine) {
	mu.Lock()
	defer mu.Unlock()
	registry[m.Name()] = m
}

// Lookup returns the machine with the given name.
func Lookup(name string) (Machine, error) {
	mu.RLock()
	defer mu.RUnlock()
	m, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown machine %q (available: %v)", name, Names())
	}
	return m, nil
}

// MustLookup returns the machine with the given name, panicking if absent.
// Intended for tests and package initialization.
func MustLookup(name string) Machine {
	m, err := Lookup(name)
	if err != nil {
		panic(err)
	}
	return m
}

// Names returns the sorted names of all registered machines.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Detect returns the first registered machine whose Detect method matches the
// given data, or nil if none match.
func Detect(data []byte) Machine {
	for _, name := range Names() {
		mu.RLock()
		m := registry[name]
		mu.RUnlock()
		if m.Detect(data) {
			return m
		}
	}
	return nil
}
