package squish

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
)

func New(limit int) Map {
	if limit <= 0 {
		limit = defaultSquishLimit
	}
	return Map{
		limit: limit,
		squish: make(map[string]interface{}),
	}
}

const defaultSquishLimit = 150

// Like a regular map, but stops storing individual values when limit is reached
type Map struct {
	// if the count of distinct values exceeds this limit, distinct values will cease to be collected (but will be counted)
	limit  int
	squish map[string]interface{}
}

// Add this observation to the collection
func (m Map) Observe(o map[string]interface{}) {
	for k, v := range o {
		m.classify(k, v)
	}
}

// Coerce all incoming values to strings or collections of strings
func (m Map) classify(k string, v interface{}) {
	switch v.(type) {
	case nil:
		m.addVal(k, "nil")
	case bool:
		m.addVal(k, strconv.FormatBool(v.(bool)))
	case string:
		m.addVal(k, v.(string))
	case int32:
		m.addVal(k, strconv.FormatInt(int64(v.(int32)), 10))
	case int64:
		m.addVal(k, strconv.FormatInt(v.(int64), 10))
	case map[string]interface{}:
		m.addSquish(k, v.(map[string]interface{}))
	case []interface{}:
		m.addSlice(k, v.([]interface{}))
	default:
		logrus.Debugf("key: %s has value %v with unhandled type %s", k, v, reflect.TypeOf(v))
	}
}

func (m Map) addSlice(k string, slice []interface{}) {
	for _, v := range slice {
		m.classify(k, v)
	}
}

func (m Map) addSquish(kk string, squish map[string]interface{}) {
	if len(squish) == 1 {
		// This is a bit of a hack for avro Union handling
		if v, ok := squish["string"]; ok {
			m.addVal(kk, v.(string))
			return
		}
	}

	var sq Map
	if kv, ok := m.squish[kk]; !ok {
		// This limit value is a heuristic and probably should be more easily tunable
		// Limit is inherited from the parent....
		sq = Map{m.limit, make(map[string]interface{})}
		m.squish[kk] = sq
	} else {
		sq = kv.(Map)
	}

	for k, v := range squish {
		sq.classify(k, v)
	}
}

const count = "*"

// Stores value occurrence counts
// Total count will be stored at '*'
type squishLeaf map[string]int

func (m Map) addVal(k string, v string) {
	// Check for existence
	if kv, ok := m.squish[k]; !ok {
		// Create initial value counter map entry
		sl := make(squishLeaf)
		sl[v] = 1
		sl[count] = 1
		m.squish[k] = sl
	} else {
		switch kv.(type) {
		case squishLeaf:
			sl := kv.(squishLeaf)
			// After m.limit unique items, classify as TooManyTooEnumerate and cease mapping value
			lsl := len(sl)
			if lsl <= m.limit+1 {
				sl[v] = sl[v] + 1
			}
			sl[count] = sl[count] + 1
		case Map:
			sqm := kv.(Map)
			sqm.addVal(k, v)
		}
	}
}

func (m Map) Dump() []string {
	path := []string{"root"}
	rKeys := fmt.Sprintf("%s.keys : %v", path, reflect.ValueOf(m.squish).MapKeys())

	acc := []string{rKeys}
	return m.walk(path, acc, m.squish)
}

const TooManyTooEnumerate = "㟢"

func (m Map) walk(path []string, acc []string, sq map[string]interface{}) []string {
	for kk, kv := range sq {
		slPath := append(path, kk)
		p := strings.Join(slPath, ".")
		switch kv.(type) {
		case squishLeaf:
			sl := kv.(squishLeaf)
			// `count` adds an additional value
			if len(sl) > m.limit+1 {
				msg := fmt.Sprintf("%s : %s : %d", p, TooManyTooEnumerate, sl[count])
				acc = append(acc, msg)
			} else {
				for k, v := range sl {
					msg := fmt.Sprintf("%s : %s : %d", p, k, v)
					acc = append(acc, msg)
				}
			}
		case Map:
			sqkv := kv.(Map)
			sqkeys := fmt.Sprintf("%s.keys : %v", p, reflect.ValueOf(sqkv.squish).MapKeys())
			sqacc := []string{sqkeys}
			sqpath := append(path, kk)
			acc = append(acc, sqkv.walk(sqpath, sqacc, sqkv.squish)...)
		}
	}
	return acc
}
