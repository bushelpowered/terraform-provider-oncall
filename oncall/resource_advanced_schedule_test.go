package oncall

import (
	"testing"
)

func Test_prettyPrintDuration(t *testing.T) {
	minuteSeconds := 60
	hourSeconds := minuteSeconds * 60
	daySeconds := hourSeconds * 24
	weekSeconds := daySeconds * 7
	tests := []struct {
		name string
		dur  int
		want string
	}{
		{
			name: "Test 1 minute",
			dur:  60,
			want: "1m",
		},
		{
			name: "Test 1 day",
			dur:  1 * daySeconds,
			want: "1d",
		},
		{
			name: "Test 1 week",
			dur:  1 * weekSeconds,
			want: "1w",
		},
		{
			name: "Test 1 day 1 hour 1 minute",
			dur:  daySeconds + hourSeconds + minuteSeconds,
			want: "1d1h1m",
		},
		{
			name: "Test 1 week 1 day 1 hour 1 minute",
			dur:  weekSeconds + daySeconds + hourSeconds + minuteSeconds,
			want: "1w1d1h1m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prettyPrintDuration(tt.dur); got != tt.want {
				t.Errorf("prettyPrintDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
