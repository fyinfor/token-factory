package setting

import "testing"

func TestVideoWatermarkPolicy(t *testing.T) {
	oldPolicy := VideoWatermarkPolicy
	t.Cleanup(func() { VideoWatermarkPolicy = oldPolicy })
	oldIDs := VideoWatermarkUserIDsToString()
	t.Cleanup(func() { _ = UpdateVideoWatermarkUserIDs(oldIDs) })

	SetVideoWatermarkPolicy(VideoWatermarkPolicyAll)
	if !IsVideoWatermarkForcedForUser(123) {
		t.Fatal("all policy should match every user")
	}
	if err := UpdateVideoWatermarkUserIDs("2, 7\n9"); err != nil {
		t.Fatal(err)
	}
	SetVideoWatermarkPolicy(VideoWatermarkPolicyUsers)
	if !IsVideoWatermarkForcedForUser(7) || IsVideoWatermarkForcedForUser(8) {
		t.Fatal("users policy matched unexpected user")
	}
	if err := CheckVideoWatermarkUserIDs("1, bad"); err == nil {
		t.Fatal("invalid user IDs should be rejected")
	}
}
