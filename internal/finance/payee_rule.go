package finance

type PayeeRule struct {
	ID               int64
	Pattern          string
	NormalizedPayee  string
	DefaultAccountID *int64 // nil = no account yet assigned
	Priority         int
}
