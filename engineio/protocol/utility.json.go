package protocol

import (
	"strconv"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	i, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	*d = Duration(time.Duration(i) * time.Millisecond)
	return err
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	c := strconv.Itoa(int(time.Duration(d) / time.Millisecond))
	return []byte(c), nil
}
