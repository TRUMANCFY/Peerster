package util

import "sync"

type StringSet struct {
	strMap map[string]bool
	Mux    *sync.Mutex
}

func GenerateStringSet(vals []string) *StringSet {
	ss := StringSet{
		strMap: make(map[string]bool),
		Mux:    &sync.Mutex{},
	}
	for _, val := range vals {
		ss.Add(val)
	}
	return &ss
}

func GenerateStringSetSingleton(val string) *StringSet {
	ss := StringSet{
		strMap: make(map[string]bool),
		Mux:    &sync.Mutex{},
	}

	ss.Add(val)
	return &ss
}

func (ss *StringSet) Add(s string) {
	ss.Mux.Lock()
	defer ss.Mux.Unlock()
	ss.strMap[s] = true
}

func (ss *StringSet) Has(s string) bool {
	ss.Mux.Lock()
	defer ss.Mux.Unlock()

	_, ok := ss.strMap[s]
	return ok
}

func (ss *StringSet) ToArray() []string {
	ss.Mux.Lock()
	defer ss.Mux.Unlock()

	keys := make([]string, 0, len(ss.strMap))
	for k := range ss.strMap {
		keys = append(keys, k)
	}
	return keys
}
