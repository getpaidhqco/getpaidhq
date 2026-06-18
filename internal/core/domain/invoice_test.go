package domain

import (
	"errors"
	"testing"
)

func TestInvoiceTransitions(t *testing.T) {
	cases := []struct {
		name    string
		from    InvoiceStatus
		apply   func(*Invoice) error
		want    InvoiceStatus
		wantErr bool
	}{
		{"draft->open", InvoiceStatusDraft, (*Invoice).MarkOpen, InvoiceStatusOpen, false},
		{"open->open idempotent", InvoiceStatusOpen, (*Invoice).MarkOpen, InvoiceStatusOpen, false},
		{"open->paid", InvoiceStatusOpen, (*Invoice).MarkPaid, InvoiceStatusPaid, false},
		{"open->uncollectible", InvoiceStatusOpen, (*Invoice).MarkUncollectible, InvoiceStatusUncollectible, false},
		{"draft->void", InvoiceStatusDraft, (*Invoice).Void, InvoiceStatusVoid, false},
		{"open->void", InvoiceStatusOpen, (*Invoice).Void, InvoiceStatusVoid, false},
		{"paid->open rejected", InvoiceStatusPaid, (*Invoice).MarkOpen, InvoiceStatusPaid, true},
		{"void->paid rejected", InvoiceStatusVoid, (*Invoice).MarkPaid, InvoiceStatusVoid, true},
		{"uncollectible->void rejected", InvoiceStatusUncollectible, (*Invoice).Void, InvoiceStatusUncollectible, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inv := Invoice{Status: c.from}
			err := c.apply(&inv)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidInvoiceTransition) {
					t.Fatalf("want ErrInvalidInvoiceTransition, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if inv.Status != c.want {
				t.Fatalf("status = %q, want %q", inv.Status, c.want)
			}
		})
	}
}
