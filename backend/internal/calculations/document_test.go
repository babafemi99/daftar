package calculations

import (
	"errors"
	"math"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/buhari"
)

func TestCalculateDocumentAssignmentSample(t *testing.T) {
	result, err := CalculateDocument([]LineInput{
		{Description: "Widget A", Quantity: 2, UnitPriceMinor: 10_000, Discount: &Discount{Type: DiscountPercentage, Value: 100_000}, TaxRate: 50_000},
		{Description: "Widget B", Quantity: 1, UnitPriceMinor: 5_000, TaxRate: 50_000},
		{Description: "Service fee", Quantity: 1, UnitPriceMinor: 20_000, Discount: &Discount{Type: DiscountFixed, Value: 2_000}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Totals{SubtotalMinor: 45_000, DiscountMinor: 4_000, TaxMinor: 1_150, GrandTotalMinor: 42_150}
	if result.Totals != want {
		t.Fatalf("totals = %+v, want %+v", result.Totals, want)
	}
	if result.PolicyVersion != PolicyVersion {
		t.Fatalf("policy = %q, want %q", result.PolicyVersion, PolicyVersion)
	}
	if len(result.TaxBreakdown) != 2 || result.TaxBreakdown[0] != (TaxBreakdown{Rate: 0, TaxableAmountMinor: 18_000}) ||
		result.TaxBreakdown[1] != (TaxBreakdown{Rate: 50_000, TaxableAmountMinor: 23_000, TaxAmountMinor: 1_150}) {
		t.Fatalf("unexpected tax breakdown: %+v", result.TaxBreakdown)
	}
}

func TestCalculateDocumentDiscountAndTaxCases(t *testing.T) {
	tests := []struct {
		name     string
		line     LineInput
		want     LineCalculated
		wantCode buhari.Code
	}{
		{
			name: "no discount and no tax",
			line: LineInput{Description: "Line", Quantity: 2, UnitPriceMinor: 500},
			want: LineCalculated{SubtotalMinor: 1_000, DiscountedAmountMinor: 1_000, LineTotalMinor: 1_000},
		},
		{
			name: "fixed discount then tax",
			line: LineInput{Description: "Line", Quantity: 1, UnitPriceMinor: 2_000, Discount: &Discount{Type: DiscountFixed, Value: 500}, TaxRate: 100_000},
			want: LineCalculated{SubtotalMinor: 2_000, DiscountAmountMinor: 500, DiscountedAmountMinor: 1_500, TaxAmountMinor: 150, LineTotalMinor: 1_650},
		},
		{
			name: "percentage discount then tax",
			line: LineInput{Description: "Line", Quantity: 1, UnitPriceMinor: 2_000, Discount: &Discount{Type: DiscountPercentage, Value: 250_000}, TaxRate: 100_000},
			want: LineCalculated{SubtotalMinor: 2_000, DiscountAmountMinor: 500, DiscountedAmountMinor: 1_500, TaxAmountMinor: 150, LineTotalMinor: 1_650},
		},
		{
			name: "fixed discount equal to subtotal",
			line: LineInput{Description: "Line", Quantity: 1, UnitPriceMinor: 2_000, Discount: &Discount{Type: DiscountFixed, Value: 2_000}, TaxRate: RateScale},
			want: LineCalculated{SubtotalMinor: 2_000, DiscountAmountMinor: 2_000},
		},
		{
			name:     "fixed discount exceeds subtotal",
			line:     LineInput{Description: "Line", Quantity: 1, UnitPriceMinor: 2_000, Discount: &Discount{Type: DiscountFixed, Value: 2_001}},
			wantCode: buhari.CodeDiscountTooLarge,
		},
		{
			name: "one hundred percent discount",
			line: LineInput{Description: "Line", Quantity: 1, UnitPriceMinor: 2_000, Discount: &Discount{Type: DiscountPercentage, Value: int64(RateScale)}, TaxRate: RateScale},
			want: LineCalculated{SubtotalMinor: 2_000, DiscountAmountMinor: 2_000},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := CalculateDocument([]LineInput{test.line})
			if test.wantCode != "" {
				assertCode(t, err, test.wantCode)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Lines[0].Calculated != test.want {
				t.Fatalf("calculated = %+v, want %+v", result.Lines[0].Calculated, test.want)
			}
		})
	}
}

func TestCalculateDocumentRoundsPerLineHalfUp(t *testing.T) {
	result, err := CalculateDocument([]LineInput{
		{Description: "Boundary", Quantity: 1, UnitPriceMinor: 1_005, Discount: &Discount{Type: DiscountPercentage, Value: 100_000}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Lines[0].Calculated.DiscountAmountMinor; got != 101 {
		t.Fatalf("discount = %d, want 101", got)
	}
}

func TestCalculateDocumentUsesRoundedLinesForTotals(t *testing.T) {
	result, err := CalculateDocument([]LineInput{
		{Description: "First", Quantity: 1, UnitPriceMinor: 5, TaxRate: 100_000},
		{Description: "Second", Quantity: 1, UnitPriceMinor: 5, TaxRate: 100_000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Totals.TaxMinor != 2 {
		t.Fatalf("tax = %d, want 2 from per-line rounding", result.Totals.TaxMinor)
	}
	if result.Totals.GrandTotalMinor != result.Totals.SubtotalMinor-result.Totals.DiscountMinor+result.Totals.TaxMinor {
		t.Fatal("document total invariant does not hold")
	}
	if result.TaxBreakdown[0].TaxAmountMinor != result.Totals.TaxMinor {
		t.Fatal("tax breakdown invariant does not hold")
	}
}

func TestCalculateDocumentZeroTaxIncludesBreakdown(t *testing.T) {
	result, err := CalculateDocument([]LineInput{{Description: "Zero rated", Quantity: 1, UnitPriceMinor: 500}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TaxBreakdown) != 1 || result.TaxBreakdown[0] != (TaxBreakdown{Rate: 0, TaxableAmountMinor: 500}) {
		t.Fatalf("unexpected breakdown: %+v", result.TaxBreakdown)
	}
}

func TestCalculateDocumentOverflow(t *testing.T) {
	_, err := CalculateDocument([]LineInput{{Description: "Overflow", Quantity: 2, UnitPriceMinor: Money(math.MaxInt64)}})
	assertCode(t, err, buhari.CodeMonetaryOverflow)

	_, err = CalculateDocument([]LineInput{{Description: "Addition overflow", Quantity: 1, UnitPriceMinor: Money(math.MaxInt64), TaxRate: 1}})
	assertCode(t, err, buhari.CodeMonetaryOverflow)
}

func TestCalculateDocumentMaximumSafeValue(t *testing.T) {
	result, err := CalculateDocument([]LineInput{{Description: "Maximum", Quantity: 1, UnitPriceMinor: Money(math.MaxInt64)}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Totals.GrandTotalMinor != Money(math.MaxInt64) {
		t.Fatalf("grand total = %d", result.Totals.GrandTotalMinor)
	}
}

func assertCode(t *testing.T, err error, code buhari.Code) {
	t.Helper()
	var appErr *buhari.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want code %s", err, code)
	}
	if appErr.Code == code {
		return
	}
	for _, field := range appErr.Fields {
		if field.Code == code {
			return
		}
	}
	t.Fatalf("error = %v, want code %s", err, code)
}
