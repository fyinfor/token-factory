package operation_setting

import "testing"

func TestParseAutoTestModelTagsJSON(t *testing.T) {
	t.Parallel()

	t.Run("json array of text tag", func(t *testing.T) {
		got, err := ParseAutoTestModelTagsJSON(`["文本"]`)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got) != 1 || got[0] != "文本" {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("empty array is valid", func(t *testing.T) {
		got, err := ParseAutoTestModelTagsJSON(`[]`)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("plain string rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`文本`); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("json string rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`"文本"`); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("null rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`null`); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty raw rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(``); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("object rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`{"tag":"文本"}`); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("non string element rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`[1]`); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty element rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`["文本",""]`); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("duplicate rejected", func(t *testing.T) {
		if _, err := ParseAutoTestModelTagsJSON(`["文本","文本"]`); err == nil {
			t.Fatal("expected error")
		}
	})
}
