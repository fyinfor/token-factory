package service

import (
	"reflect"
	"testing"

	"github.com/fyinfor/router-engine/pkg/router"
)

func TestOrderEndpointCandidateIDsByPrice(t *testing.T) {
	got := orderEndpointCandidateIDsByPrice([]*router.EndpointCandidate{
		{ChannelID: 3, UnitPrice: 3},
		{ChannelID: 1, UnitPrice: 1},
		nil,
		{ChannelID: 2, UnitPrice: 2},
	})
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderEndpointCandidateIDsByPrice = %v, want %v", got, want)
	}
	if orderEndpointCandidateIDsByPrice(nil) != nil {
		t.Fatal("nil candidates should return nil")
	}
}

func TestPreferChannelFirstAfterVideoFilterSemantics(t *testing.T) {
	// 模拟：视频过滤后剩余 [hw=1106, u15=1070]，偏好 u15 → 有序应为 [1070, 1106]
	ordered := PreferChannelFirst([]int{1106, 1070}, 1070)
	want := []int{1070, 1106}
	if !reflect.DeepEqual(ordered, want) {
		t.Fatalf("PreferChannelFirst = %v, want %v", ordered, want)
	}
}
