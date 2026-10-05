package terminal

import (
	"encoding/base64"
	"slices"
	"testing"
)

func TestCopyOSC52(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	tests := []struct {
		name   string
		chunks []string
		want   []string
	}{
		{"bel", []string{"x\x1b]52;c;" + b64 + "\ay"}, []string{"hello"}},
		{"string terminator", []string{"\x1b]52;;" + b64 + "\x1b\\"}, []string{"hello"}},
		{"split across reads", []string{"ab\x1b]5", "2;c;aGVs", "bG8=\a"}, []string{"hello"}},
		{"two sequences", []string{"\x1b]52;c;" + b64 + "\a\x1b]52;c;" + b64 + "\a"}, []string{"hello", "hello"}},
		{"query is ignored", []string{"\x1b]52;c;?\a"}, nil},
		{"other osc", []string{"\x1b]0;title\a"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			s := &stream{clip: func(text string) { got = append(got, text) }}
			for _, c := range tt.chunks {
				s.copyOSC52([]byte(c))
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("clipboard = %q, want %q", got, tt.want)
			}
		})
	}
}
