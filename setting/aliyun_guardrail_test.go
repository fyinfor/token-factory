package setting

import "testing"

func TestAliyunGuardrailUserScope(t *testing.T) {
	original := AliyunGuardrailUserIDsToString()
	t.Cleanup(func() {
		if err := UpdateAliyunGuardrailUserIDs(original); err != nil {
			t.Fatalf("restore Aliyun guardrail user scope: %v", err)
		}
	})

	if err := UpdateAliyunGuardrailUserIDs("9,2\n9"); err != nil {
		t.Fatalf("update user scope: %v", err)
	}
	if got := AliyunGuardrailUserIDsToString(); got != "2,9" {
		t.Fatalf("unexpected normalized user IDs: %q", got)
	}
	if !HasAliyunGuardrailUserScope() {
		t.Fatal("expected configured user scope")
	}
	if !IsAliyunGuardrailEnabledForUser(2) || IsAliyunGuardrailEnabledForUser(3) {
		t.Fatal("user scope did not apply as expected")
	}

	if err := UpdateAliyunGuardrailUserIDs(""); err != nil {
		t.Fatalf("clear user scope: %v", err)
	}
	if HasAliyunGuardrailUserScope() {
		t.Fatal("empty scope must apply to all users")
	}
	if !IsAliyunGuardrailEnabledForUser(3) {
		t.Fatal("empty user scope should allow every user")
	}
}

func TestCheckAliyunGuardrailUserIDs(t *testing.T) {
	if err := CheckAliyunGuardrailUserIDs("1,invalid"); err == nil {
		t.Fatal("expected invalid user ID to be rejected")
	}
	if err := CheckAliyunGuardrailUserIDs("0"); err == nil {
		t.Fatal("expected non-positive user ID to be rejected")
	}
}
