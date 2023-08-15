package waitgroup

import "sync"
import "sort"

type Creator func() Mod

type creatorWrapper struct {
	c     Creator
	order int
	name  string
}

var creators = map[string]*creatorWrapper{}
var mu = sync.Mutex{}

type creatorWrappers []*creatorWrapper

func (s creatorWrappers) Len() int {
	return len(s)
}
func (s creatorWrappers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s creatorWrappers) Less(i, j int) bool {
	return s[i].order < s[j].order
}

func AddModCreator(name string, order int, creator Creator) {
	mu.Lock()
	defer mu.Unlock()

	creators[name] = &creatorWrapper{
		c:     creator,
		order: order,
		name:  name,
	}
}

func ModCreatorsCount() int {
	mu.Lock()
	defer mu.Unlock()

	return len(creators)
}

func sortCreators(asc bool) []*creatorWrapper {
	cws := creatorWrappers{}
	for _, cw := range creators {
		cws = append(cws, cw)
	}

	if asc {
		sort.Sort(cws)
	} else {
		sort.Sort(sort.Reverse(cws))
	}
	return cws
}
