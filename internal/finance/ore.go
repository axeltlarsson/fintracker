package finance

import (
	"fmt"
)

type Öre int64

func (ö Öre) String() string {
	sign := ""
	v := int64(ö)

	if v < 0 {
		sign = "-"
		v = -v
	}

	return fmt.Sprintf("%s%d,%02d kr", sign, v/100, v%100)
}
