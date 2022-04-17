package flagextra

import "fmt"

type StringList []string

func (ss *StringList) String() string {
	return fmt.Sprintf("%s", *ss)
}

func (ss *StringList) Set(v string) error {
	*ss = append(*ss, v)
	return nil
}
