// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package metrics

import (
	"testing"
	"time"
)

func TestFormatHashrate(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0.00 H/s"},
		{12.5, "12.50 H/s"},
		{999.0, "999.00 H/s"},
		{1500.0, "1.50 KH/s"},
		{2_500_000, "2.50 MH/s"},
		{3_700_000_000, "3.70 GH/s"},
		{1_200_000_000_000, "1.20 TH/s"},
	}

	for _, tt := range tests {
		got := FormatHashrate(tt.input)
		if got != tt.want {
			t.Errorf("FormatHashrate(%g) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3661 * time.Second, "1h01m01s"},
		{7200 * time.Second, "2h00m00s"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.input)
		if got != tt.want {
			t.Errorf("FormatDuration(%s) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
