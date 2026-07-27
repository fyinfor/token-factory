package service

import (
	"reflect"
	"testing"
)

func TestPreferChannelFirst(t *testing.T) {
	cases := []struct {
		name        string
		order       []int
		preferredID int
		want        []int
	}{
		{name: "empty preferred", order: []int{2, 3}, preferredID: 0, want: []int{2, 3}},
		{name: "preferred already first", order: []int{5, 1, 2}, preferredID: 5, want: []int{5, 1, 2}},
		{name: "preferred in middle", order: []int{1, 5, 2}, preferredID: 5, want: []int{5, 1, 2}},
		{name: "preferred missing", order: []int{1, 2}, preferredID: 9, want: []int{9, 1, 2}},
		{name: "empty order", order: nil, preferredID: 7, want: []int{7}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PreferChannelFirst(tc.order, tc.preferredID)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("PreferChannelFirst(%v, %d) = %v, want %v", tc.order, tc.preferredID, got, tc.want)
			}
		})
	}
}
