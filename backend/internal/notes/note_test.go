package notes

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	title, body, err := Normalize("  Купить фильтр  ", "  масло\n")
	if err != nil {
		t.Fatal(err)
	}
	if title != "Купить фильтр" || body != "  масло\n" {
		t.Fatalf("unexpected normalized values: %q %q", title, body)
	}

	_, _, err = Normalize("   ", "text")
	if err == nil || !strings.Contains(err.Error(), "заголовок") {
		t.Fatalf("expected title error, got %v", err)
	}

	_, _, err = Normalize(strings.Repeat("я", 201), "")
	if err == nil || !strings.Contains(err.Error(), "200") {
		t.Fatalf("expected length error, got %v", err)
	}
}
