package controllers

import (
	"testing"

	"github.com/grafana/k6-operator/pkg/cloud"
)

func TestParseInspectOutput(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"plain", "{\n  \"maxVUs\": 5\n}", false},
		{"timestamped lines", "2023-08-28T17:30:52.4951751Z {\n2023-08-28T17:30:52.4951751Z   \"maxVUs\": 5\n2023-08-28T17:30:52.4951751Z }\n", false},
		{"garbage", "not json", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out cloud.InspectOutput
			err := parseInspectOutput([]byte(tt.in), &out)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && out.MaxVUs != 5 {
				t.Fatalf("MaxVUs = %d, want 5", out.MaxVUs)
			}
		})
	}
}
