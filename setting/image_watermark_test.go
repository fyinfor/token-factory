package setting

import "testing"

func TestImageWatermarkPolicy(t *testing.T) {
	oldPolicy := ImageWatermarkPolicy
	oldIDs := ImageWatermarkUserIDsToString()
	t.Cleanup(func() {
		ImageWatermarkPolicy = oldPolicy
		_ = UpdateImageWatermarkUserIDs(oldIDs)
	})

	SetImageWatermarkPolicy(ImageWatermarkPolicyAll)
	if !IsImageWatermarkForcedForUser(42) {
		t.Fatal("all policy should apply to every user")
	}
	if err := UpdateImageWatermarkUserIDs("2, 7\n9"); err != nil {
		t.Fatal(err)
	}
	SetImageWatermarkPolicy(ImageWatermarkPolicyUsers)
	if !IsImageWatermarkForcedForUser(7) || IsImageWatermarkForcedForUser(8) {
		t.Fatal("users policy did not match configured IDs")
	}
}

func TestImageWatermarkPositionAliases(t *testing.T) {
	for _, value := range []string{"bottom_right", "bottom-right", "center"} {
		if err := CheckImageWatermarkPosition(value); err != nil {
			t.Fatalf("position %q rejected: %v", value, err)
		}
	}
}

func TestCheckImageWatermarkTextRejectsUnsupportedCharacters(t *testing.T) {
	if err := CheckImageWatermarkText("TokenFactory"); err != nil {
		t.Fatalf("ASCII text rejected: %v", err)
	}
	if err := CheckImageWatermarkText("中文水印"); err == nil {
		t.Fatal("unsupported CJK text should be rejected")
	}
}

func TestCheckImageWatermarkEnablement(t *testing.T) {
	oldPolicy := ImageWatermarkPolicy
	oldType := ImageWatermarkType
	oldText := ImageWatermarkText
	oldLogo := ImageWatermarkLogoURL
	oldIDs := ImageWatermarkUserIDsToString()
	t.Cleanup(func() {
		ImageWatermarkPolicy = oldPolicy
		ImageWatermarkType = oldType
		ImageWatermarkText = oldText
		ImageWatermarkLogoURL = oldLogo
		_ = UpdateImageWatermarkUserIDs(oldIDs)
	})

	SetImageWatermarkType(ImageWatermarkTypeLogo)
	ImageWatermarkLogoURL = ""
	if err := CheckImageWatermarkEnablement(ImageWatermarkPolicyAll); err == nil {
		t.Fatal("logo mode without a logo should not be enabled")
	}
	ImageWatermarkLogoURL = "/api/uploads/permanent/watermarks/logo.png"
	if err := CheckImageWatermarkEnablement(ImageWatermarkPolicyAll); err != nil {
		t.Fatalf("valid logo mode rejected: %v", err)
	}
	if err := UpdateImageWatermarkUserIDs(""); err != nil {
		t.Fatal(err)
	}
	if err := CheckImageWatermarkEnablement(ImageWatermarkPolicyUsers); err == nil {
		t.Fatal("users policy without users should not be enabled")
	}
}

func TestCheckImageWatermarkContentUpdateWhileEnabled(t *testing.T) {
	oldPolicy := ImageWatermarkPolicy
	oldType := ImageWatermarkType
	oldText := ImageWatermarkText
	oldLogo := ImageWatermarkLogoURL
	t.Cleanup(func() {
		ImageWatermarkPolicy = oldPolicy
		ImageWatermarkType = oldType
		ImageWatermarkText = oldText
		ImageWatermarkLogoURL = oldLogo
	})

	SetImageWatermarkPolicy(ImageWatermarkPolicyAll)
	SetImageWatermarkType(ImageWatermarkTypeText)
	ImageWatermarkText = "TF"
	ImageWatermarkLogoURL = ""
	if err := CheckImageWatermarkContentUpdate("ImageWatermarkText", ""); err == nil {
		t.Fatal("enabled text watermark should reject empty text")
	}
	if err := CheckImageWatermarkContentUpdate("ImageWatermarkType", ImageWatermarkTypeLogo); err == nil {
		t.Fatal("enabled watermark should reject switching to logo without a logo")
	}
	if err := CheckImageWatermarkContentUpdate("ImageWatermarkLogoURL", "/api/uploads/logo.png"); err != nil {
		t.Fatalf("valid dormant logo rejected: %v", err)
	}
}
