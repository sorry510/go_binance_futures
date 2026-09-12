package binance

import "testing"

func TestFormatOwnedOrderDecimalRemovesFloatNoise(t *testing.T) {
	tests := map[float64]string{
		0.0024000000000000002: "0.0024",
		74639.70000000001:     "74639.7",
		0.00000001:            "0.00000001",
		123.45000000000002:    "123.45",
		0:                     "0",
	}
	for input, want := range tests {
		if got := formatOwnedOrderDecimal(input); got != want {
			t.Fatalf("formatOwnedOrderDecimal(%0.18f)=%q want %q", input, got, want)
		}
	}
}
