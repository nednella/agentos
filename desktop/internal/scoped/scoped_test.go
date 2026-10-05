package scoped

import (
	"encoding/json"
	"testing"
)

func TestOf(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  string
	}{
		{"items", []int{1, 2}, `{"project":"p","items":[1,2]}`},
		{"empty", []int{}, `{"project":"p","items":[]}`},
		{"nil", nil, `{"project":"p","items":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(Of("p", tt.items))
			if err != nil || string(got) != tt.want {
				t.Errorf("got %s, %v; want %s", got, err, tt.want)
			}
		})
	}
}
