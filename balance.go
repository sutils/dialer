package dialer

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/Centny/gwf/log"
	"github.com/Centny/gwf/util"
)

type MapIntSorter struct {
	List  []string
	Data  map[string][]int64
	Index int
}

func NewMapIntSorter(data map[string][]int64, index int) *MapIntSorter {
	sorter := &MapIntSorter{
		Data:  data,
		Index: index,
	}
	for name := range data {
		sorter.List = append(sorter.List, name)
	}
	return sorter
}

func (m *MapIntSorter) Len() int {
	return len(m.List)
}

func (m *MapIntSorter) Less(i, j int) bool {
	return m.Data[m.List[i]][m.Index] < m.Data[m.List[j]][m.Index]
}

func (m *MapIntSorter) Swap(i, j int) {
	m.List[i], m.List[j] = m.List[j], m.List[i]
}

type Policy struct {
	Matcher *regexp.Regexp
	Limit   []int64
}

type BalancedDialer struct {
	ID          string
	dialers     map[string]Dialer
	dialersUsed map[string][]int64 //map key to [begin,used,fail]
	dialersLock sync.RWMutex
	PolicyList  []*Policy
	Delay       int64
	Timeout     int64
	Conf        util.Map
	matcher     *regexp.Regexp
}

func NewBalancedDialer() *BalancedDialer {
	return &BalancedDialer{
		dialers:     map[string]Dialer{},
		dialersUsed: map[string][]int64{},
		dialersLock: sync.RWMutex{},
		Delay:       500,
		Timeout:     60000,
		Conf:        util.Map{},
		matcher:     regexp.MustCompile(".*"),
	}
}

func (b *BalancedDialer) sortedDialer(index int) []string {
	sorter := NewMapIntSorter(b.dialersUsed, index)
	sort.Sort(sorter)
	return sorter.List
}

func (b *BalancedDialer) AddPolicy(matcher string, limit []int64) (err error) {
	if len(limit) < 2 {
		err = fmt.Errorf("limit must be [time,limit]")
		return
	}
	reg, err := regexp.Compile(matcher)
	if err == nil {
		b.PolicyList = append(b.PolicyList, &Policy{
			Matcher: reg,
			Limit:   limit,
		})
	}
	return
}

func (b *BalancedDialer) AddDialer(dialers ...Dialer) {
	b.dialersLock.Lock()
	for _, dialer := range dialers {
		name := dialer.Name()
		b.dialers[name] = dialer
		b.dialersUsed[name] = []int64{0, 0, 0}
	}
	b.dialersLock.Unlock()
	return
}

func (b *BalancedDialer) Name() string {
	return b.ID
}

//initial dialer
func (b *BalancedDialer) Bootstrap(options util.Map) (err error) {
	b.Conf = options
	b.ID = options.StrVal("id")
	if len(b.ID) < 1 {
		err = fmt.Errorf("the dialer id is required")
		return
	}
	matcher := options.StrVal("matcher")
	if len(matcher) > 0 {
		b.matcher, err = regexp.Compile(matcher)
	}
	b.Timeout = options.IntValV("timeout", 60000)
	b.Delay = options.IntValV("delay", 500)
	policy := options.AryMapVal("policy")
	for _, p := range policy {
		err = b.AddPolicy(p.StrVal("matcher"), p.AryInt64Val("limit"))
		if err != nil {
			return
		}
	}
	b.dialersLock.Lock()
	defer b.dialersLock.Unlock()
	dialerOptions := options.AryMapVal("dialers")
	for _, option := range dialerOptions {
		dtype := option.StrVal("type")
		dialer := NewDialer(dtype)
		if dialer == nil {
			return fmt.Errorf("create dialer fail by %v", util.S2Json(option))
		}
		err := dialer.Bootstrap(option)
		if err != nil {
			return err
		}
		name := dialer.Name()
		b.dialers[name] = dialer
		b.dialersUsed[name] = []int64{0, 0, 0}
		log.D("BalancedDialer add dialer(%v) to pool success", dialer)
	}
	return nil
}

//Options
func (b *BalancedDialer) Options() util.Map {
	return b.Conf
}

//Matched uri
func (b *BalancedDialer) Matched(uri string) bool {
	return b.matcher.MatchString(uri)
}

func (b *BalancedDialer) Dial(sid uint64, uri string) (r io.ReadWriteCloser, err error) {
	//
	begin := util.Now()
	var showed int64
	for {
		now := util.Now()
		if now-begin >= b.Timeout {
			err = fmt.Errorf("dial to %v timeout", uri)
			break
		}
		var policy *Policy
		var names []string
		b.dialersLock.Lock()
		for _, p := range b.PolicyList {
			if p.Matcher.MatchString(uri) {
				policy = p
			}
		}
		if policy == nil {
			names = b.sortedDialer(1)
		} else {
			for name, used := range b.dialersUsed {
				if now-used[0] > policy.Limit[0] {
					names = append(names, name)
					used[1] = 0
				}
				if used[1] < policy.Limit[1] {
					names = append(names, name)
				}
			}
		}
		for _, name := range names {
			dialer := b.dialers[name]
			if !dialer.Matched(uri) {
				continue
			}
			used := b.dialersUsed[name]
			if used[1] == 0 {
				used[0] = util.Now()
			}
			used[1]++
			b.dialersLock.Unlock()
			r, err = dialer.Dial(sid, uri)
			b.dialersLock.Lock()
			if err == nil {
				used[2] = 0
				b.dialersLock.Unlock()
				return
			}
			used[2]++
			log.D("BalancedDialer using %v and dial to %v fail with %v", dialer, uri, err)
			failRemove := dialer.Options().IntValV("fail_remove", 0)
			if failRemove > 0 && used[2] >= failRemove {
				log.D("BalancedDialer remove dialer(%v) by %v fail count", dialer, used[2])
				delete(b.dialers, name)
				delete(b.dialersUsed, name)
			}
		}
		b.dialersLock.Unlock()
		now = util.Now()
		if now-showed > 3000 {
			log.D("BalancedDialer dial to %v is waiting", uri)
			showed = now
		}
		time.Sleep(time.Duration(b.Delay) * time.Millisecond)
	}
	return
}
