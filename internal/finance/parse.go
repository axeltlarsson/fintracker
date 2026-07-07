package finance

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseAmount(s string) (Öre, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")

	parts := strings.SplitN(s, ",", 2)

	kronor, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var öre int64
	if len(parts) == 2 {
		// pad or truncate to exactly 2 digits
		örePart := parts[1]
		if len(örePart) > 2 {
			return 0, fmt.Errorf("too many decimal digits: %q", s)
		}
		if len(örePart) == 1 {
			örePart += "0"
		}
		öre, err = strconv.ParseInt(örePart, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	total := kronor*100 + öre
	if kronor < 0 {
		total = kronor*100 - öre
	}
	return Öre(total), nil

}
