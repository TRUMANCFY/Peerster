package util

type StringSet struct {
	strMap map[string]bool
}

func GenerateStringSet(vals []string) *StringSet {
	ss := StringSet{make(map[string]bool)}
	for _, val := range vals {
		ss.Add(val)
	}
	return &ss
}

func GenerateStringSetSingleton(val string) *StringSet {
	ss := StringSet{make(map[string]bool)}
	ss.Add(val)
	return &ss
}

func (ss *StringSet) Add(s string) {
	ss.strMap[s] = true
}

func (ss *StringSet) Has(s string) bool {
	_, ok := ss.strMap[s]
	return ok
}

func (ss *StringSet) ToArray() []string {
	keys := make([]string, 0, len(ss.strMap))
	for k := range ss.strMap {
		keys = append(keys, k)
	}
	return keys
}
