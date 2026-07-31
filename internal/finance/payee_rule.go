package finance

import "strings"

type PayeeRule struct {
	ID               int64
	Pattern          string
	NormalizedPayee  string
	DefaultAccountID *int64 // nil = no account yet assigned
	Priority         int
}

func (r PayeeRule) Matches(rawPayee string) bool {
	return strings.Contains(strings.ToUpper(rawPayee), strings.ToUpper(r.Pattern))
}
