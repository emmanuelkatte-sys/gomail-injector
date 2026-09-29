package email

import (
	"math/rand"
	"strings"
	"testing"
)

func TestShuffleHeadersPreservesDkimSignBlockOrder(t *testing.T) {
	headers := []string{
		"Received: from iphone.local ([100.1.2.3]) by mail.example.com with ESMTPSA id ABC123; Mon, 06 Aug 2026 00:00:00 GMT\r\n",
		"From: a@mail.example.com\r\n",
		"To: b@docomo.ne.jp\r\n",
		"Subject: test\r\n",
		"Date: Mon, 06 Aug 2026 00:00:00 +0900\r\n",
		"Message-ID: <1@mail.example.com>\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Language: ja\r\n",
		"Reply-To: reply@mail.example.com\r\n",
	}
	signHeaders := []string{"from", "to", "subject", "date", "message-id", "content-type"}
	r := rand.New(rand.NewSource(42))

	for i := 0; i < 20; i++ {
		out := shuffleHeadersWithAnchors(headers, r, signHeaders)
		if len(out) != len(headers) {
			t.Fatalf("iteration %d: length mismatch %d vs %d", i, len(out), len(headers))
		}
		block, ok := contiguousSignBlock(out, signHeaders)
		if !ok {
			t.Fatalf("iteration %d: sign block not contiguous", i)
		}
		want := strings.Join(signHeaders, ",")
		if strings.Join(block, ",") != want {
			t.Fatalf("iteration %d: sign block %v want %v", i, block, signHeaders)
		}
		if !strings.HasPrefix(out[0], "Received:") {
			t.Fatalf("iteration %d: received not first", i)
		}
	}
}

func contiguousSignBlock(headers []string, signHeaders []string) ([]string, bool) {
	order := buildDkimSignOrder(signHeaders)
	block := make([]string, 0, len(signHeaders))
	inBlock := false
	for _, h := range headers {
		name := extractFieldName(h)
		if idx, ok := order[name]; ok {
			if !inBlock {
				inBlock = true
			}
			if len(block) != idx {
				return nil, false
			}
			block = append(block, name)
			continue
		}
		if inBlock {
			break
		}
	}
	if len(block) != len(signHeaders) {
		return nil, false
	}
	return block, true
}
