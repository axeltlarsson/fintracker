package finance

import "testing"

func TestÖreString(t *testing.T) {
	tests := []struct {
		name string
		öre  Öre
		want string
	}{
		{name: "zero", öre: 0, want: "0,00 kr"},
		{name: "positive whole", öre: 100_00, want: "100,00 kr"},
		{name: "positive with öre", öre: 49_50, want: "49,50 kr"},
		{name: "negative", öre: -15_00, want: "-15,00 kr"},
		{name: "single öre", öre: 1, want: "0,01 kr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.öre.String()
			if got != tt.want {
				t.Errorf("Öre(%d).String() = %q, want %q", tt.öre, got, tt.want)
			}
		})
	}
}

func TestCalculateBalance(t *testing.T) {
	txns := []Transaction{
		{Amount: 1000_00},
		{Amount: -250_50},
		{Amount: -100_00},
	}
	got := CalculateBalance(txns)
	want := Öre(649_50)

	if got != want {
		t.Errorf("CalculateBalance() = %v, want %v", got, want)
	}
}
