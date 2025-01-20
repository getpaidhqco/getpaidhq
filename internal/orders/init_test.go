package orders

import (
	"context"
	"testing"
)

func TestInit(t *testing.T) {
	ctx := context.Background()
	input := InitOrderInput{
		TID: "test_tid",
	}

	err := Init(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
