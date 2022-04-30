package flagextra

import (
	"encoding/json"
)

func NewStringSliceValue(val []string, p *[]string) *stringSliceValue {
	*p = val
	return &stringSliceValue{
		val: p,
		def: true,
	}
}

type stringSliceValue struct {
	val *[]string
	def bool
}

func (ss *stringSliceValue) Set(v string) error {
	if ss.def {
		ss.def = false
		*ss.val = []string{v}
	} else {
		*ss.val = append(*ss.val, v)
	}
	return nil
}

func (ss *stringSliceValue) String() string {
	b, err := json.Marshal(ss.val)
	if err != nil {
		return ""
	}
	return string(b)
}
