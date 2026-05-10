package id

import (
	"regexp"
	"testing"
)

func TestUUIDGeneratorGenerate(t *testing.T) {
	uuid, err := NewUUIDGenerator().Generate()
	if err != nil {
		t.Fatalf("generate uuid: %v", err)
	}

	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(uuid) {
		t.Fatalf("expected UUID v4, got %q", uuid)
	}
}
