package emailcheck

import (
	_ "embed"
	"strings"
	"sync"
)

// Disposable-domain list from github.com/disposable-email-domains (CC0), one domain per line.
//
//go:embed data/disposable.txt
var disposableTxt string

var (
	disposableOnce sync.Once
	disposableSet  map[string]bool
)

// DisposableDomains returns the built-in set of disposable (throwaway) email domains.
// The map is shared; copy it before changing it.
func DisposableDomains() map[string]bool {
	disposableOnce.Do(func() {
		lines := strings.Split(strings.TrimSpace(disposableTxt), "\n")
		disposableSet = make(map[string]bool, len(lines))
		for _, d := range lines {
			disposableSet[d] = true
		}
	})
	return disposableSet
}
