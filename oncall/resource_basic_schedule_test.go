package oncall

import (
	"testing"

	"maze.io/x/duration"
)

func Test_secondsToDayHourMinute(t *testing.T) {
	tests := []struct {
		name        string
		inSeconds   int
		wantDays    int
		wantHours   int
		wantMinutes int
	}{
		{
			name:        "Start of week",
			inSeconds:   0 * int(duration.Hour.Seconds()),
			wantDays:    0,
			wantHours:   0,
			wantMinutes: 0,
		},
		{
			name:        "Noon on sunday",
			inSeconds:   12 * int(duration.Hour.Seconds()),
			wantDays:    0,
			wantHours:   12,
			wantMinutes: 0,
		},
		{
			name:        "12:31 on sunday",
			inSeconds:   12*int(duration.Hour.Seconds()) + 31*int(duration.Minute.Seconds()),
			wantDays:    0,
			wantHours:   12,
			wantMinutes: 31,
		},
		{
			name:        "12:31 on Monday",
			inSeconds:   1*int(duration.Day.Seconds()) + 12*int(duration.Hour.Seconds()) + 31*int(duration.Minute.Seconds()),
			wantDays:    1,
			wantHours:   12,
			wantMinutes: 31,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDays, gotHours, gotMinutes := secondsToDayHourMinute(tt.inSeconds)
			if gotDays != tt.wantDays {
				t.Errorf("secondsToDayHourMinute() gotDays = %v, want %v", gotDays, tt.wantDays)
			}
			if gotHours != tt.wantHours {
				t.Errorf("secondsToDayHourMinute() gotHours = %v, want %v", gotHours, tt.wantHours)
			}
			if gotMinutes != tt.wantMinutes {
				t.Errorf("secondsToDayHourMinute() gotMinutes = %v, want %v", gotMinutes, tt.wantMinutes)
			}
		})
	}
}

func Test_weekdayStartTimeToSeconds(t *testing.T) {
	type args struct {
		weekday   string
		startTime string
	}
	tests := []struct {
		name        string
		args        args
		wantSeconds int
		wantErr     bool
	}{
		{
			name: "Start of week",
			args: args{
				weekday:   "Sunday",
				startTime: "00:00",
			},
			wantSeconds: 0,
			wantErr:     false,
		},
		{
			name: "One minute into the week",
			args: args{
				weekday:   "Sunday",
				startTime: "00:01",
			},
			wantSeconds: 60,
			wantErr:     false,
		},
		{
			name: "Monday at 11:58 PM",
			args: args{
				weekday:   "Monday",
				startTime: "23:58",
			},
			wantSeconds: 1*int(duration.Day.Seconds()) + 23*int(duration.Hour.Seconds()) + 58*int(duration.Minute.Seconds()),
			wantErr:     false,
		},
		{
			name: "Test bad time",
			args: args{
				weekday:   "Monday",
				startTime: "23:60",
			},
			wantSeconds: -1,
			wantErr:     true,
		},
		{
			name: "Test bad day",
			args: args{
				weekday:   "Oliverday",
				startTime: "23:58",
			},
			wantSeconds: -1,
			wantErr:     true,
		},
		{
			name: "Test 12 hour tiem",
			args: args{
				weekday:   "Friday",
				startTime: "11:30 PM",
			},
			wantSeconds: -1,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSeconds, err := weekdayStartTimeToSeconds(tt.args.weekday, tt.args.startTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("startTimeWeekdayToSeconds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSeconds != tt.wantSeconds {
				t.Errorf("startTimeWeekdayToSeconds() = %v, want %v", gotSeconds, tt.wantSeconds)
			}
		})
	}
}
