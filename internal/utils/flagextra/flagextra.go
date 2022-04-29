package flagextra

import (
	"encoding/json"
)

func NewStringSliceValue(def []string, p *[]string) *stringSliceValue {
	*p = def
	return (*stringSliceValue)(p)
}

type stringSliceValue []string

func (ss *stringSliceValue) Set(v string) error {
	*ss = append(*ss, v)
	return nil
}

func (ss *stringSliceValue) String() string {
	b, err := json.Marshal(*ss)
	if err != nil {
		return ""
	}
	return string(b)
}
