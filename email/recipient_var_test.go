package email

import (
	"math/rand"
	"strings"
	"testing"

	"__MODULE_PLACEHOLDER__/config"
	"__MODULE_PLACEHOLDER__/types"
)

func Test4PyVariablesAndRecipientModes(t *testing.T) {
	cfg := &config.Config{
		Headers: config.HeadersConfig{
			RecipientDisplayMode: "3",
			RecipientHonorific:   "様",
			RecipientCustomPhrases: []string{"様", "さん", "お客様"},
		},
	}
	vp := NewVariableProcessor(cfg, 12345)

	// 1. Test {third_day_date_kanji}
	text := "Hello {third_day_date_kanji}, {THIRD_DAY_DATE_KANJI}, {{third_day_date_kanji}}, %tomorrow_date, [RECEIVER_ADDRESS]"
	recipient := &types.Recipient{
		Email: "testuser@example.com",
		Name:  "TestUser",
	}
	processed := vp.Process(text, recipient)

	if strings.Contains(processed, "{third_day_date_kanji}") {
		t.Fatalf("{third_day_date_kanji} was not replaced: %s", processed)
	}
	if strings.Contains(processed, "{THIRD_DAY_DATE_KANJI}") {
		t.Fatalf("{THIRD_DAY_DATE_KANJI} was not replaced: %s", processed)
	}
	if strings.Contains(processed, "{{third_day_date_kanji}}") {
		t.Fatalf("{{third_day_date_kanji}} was not replaced: %s", processed)
	}
	if !strings.Contains(processed, "testuser@example.com") {
		t.Fatalf("recipient email was not replaced in [RECEIVER_ADDRESS]: %s", processed)
	}
	if !strings.Contains(processed, "年") || !strings.Contains(processed, "月") || !strings.Contains(processed, "日") {
		t.Fatalf("Kanji date not present in processed: %s", processed)
	}

	// 2. Test resolveRecipientDisplayName across all modes
	rng := rand.New(rand.NewSource(12345))
	
	// Mode 0: only email -> empty display name
	name0, _ := resolveRecipientDisplayName("testuser@example.com", "", "0", "様", nil, rng)
	if name0 != "" {
		t.Fatalf("Mode 0 expected empty name, got: %s", name0)
	}

	// Mode 1: email as name
	name1, _ := resolveRecipientDisplayName("testuser@example.com", "", "1", "様", nil, rng)
	if name1 != "testuser@example.com" {
		t.Fatalf("Mode 1 expected email, got: %s", name1)
	}

	// Mode 2: prefix as name
	name2, _ := resolveRecipientDisplayName("testuser@example.com", "", "2", "様", nil, rng)
	if name2 != "Testuser" {
		t.Fatalf("Mode 2 expected Testuser, got: %s", name2)
	}

	// Mode 3: prefix + honorific
	name3, _ := resolveRecipientDisplayName("testuser@example.com", "", "3", "様", nil, rng)
	if name3 != "Testuser様" {
		t.Fatalf("Mode 3 expected Testuser様, got: %s", name3)
	}

	// Mode 4: email + honorific
	name4, _ := resolveRecipientDisplayName("testuser@example.com", "", "4", "様", nil, rng)
	if name4 != "testuser@example.com様" {
		t.Fatalf("Mode 4 expected testuser@example.com様, got: %s", name4)
	}

	// Mode 5: independent phrase
	name5, _ := resolveRecipientDisplayName("testuser@example.com", "", "5", "様", []string{"お客様"}, rng)
	if name5 != "お客様" {
		t.Fatalf("Mode 5 expected お客様, got: %s", name5)
	}

	t.Logf("All Go injector checks passed! Processed sample: %s", processed)
}
