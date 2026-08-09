// Package calculations implements Daftar's deterministic financial policy.
// It has no database, HTTP, clock, or global-state dependencies.
package calculations

import (
	"math"
	"math/bits"
	"sort"
	"strings"

	"github.com/babafemi99/daftar/backend/internal/buhari"
)

const (
	PolicyVersion       = "2026-08-v1"
	RateScale      Rate = 1_000_000
	MaxQuantity         = 1_000_000
	MaxLineItems        = 200
	MaxDescription      = 500
)

type Money int64
type Rate int64
type Quantity int64
type DiscountType string

const (
	DiscountFixed      DiscountType = "fixed"
	DiscountPercentage DiscountType = "percentage"
)

type Discount struct {
	Type  DiscountType `bson:"type" json:"type"`
	Value int64        `bson:"value" json:"value"`
}

type LineInput struct {
	ID             string    `bson:"id" json:"id"`
	Description    string    `bson:"description" json:"description"`
	Quantity       Quantity  `bson:"quantity" json:"quantity"`
	UnitPriceMinor Money     `bson:"unitPriceMinor" json:"unitPriceMinor"`
	Discount       *Discount `bson:"discount,omitempty" json:"discount,omitempty"`
	TaxRate        Rate      `bson:"taxRate" json:"taxRate"`
}

type LineCalculated struct {
	SubtotalMinor         Money `bson:"subtotalMinor" json:"subtotalMinor"`
	DiscountAmountMinor   Money `bson:"discountAmountMinor" json:"discountAmountMinor"`
	DiscountedAmountMinor Money `bson:"discountedAmountMinor" json:"discountedAmountMinor"`
	TaxAmountMinor        Money `bson:"taxAmountMinor" json:"taxAmountMinor"`
	LineTotalMinor        Money `bson:"lineTotalMinor" json:"lineTotalMinor"`
}

type LineResult struct {
	LineInput  `bson:",inline"`
	Calculated LineCalculated `bson:"calculated" json:"calculated"`
}

type Totals struct {
	SubtotalMinor   Money `bson:"subtotalMinor" json:"subtotalMinor"`
	DiscountMinor   Money `bson:"discountMinor" json:"discountMinor"`
	TaxMinor        Money `bson:"taxMinor" json:"taxMinor"`
	GrandTotalMinor Money `bson:"grandTotalMinor" json:"grandTotalMinor"`
}

type TaxBreakdown struct {
	Rate               Rate  `bson:"rate" json:"rate"`
	TaxableAmountMinor Money `bson:"taxableAmountMinor" json:"taxableAmountMinor"`
	TaxAmountMinor     Money `bson:"taxAmountMinor" json:"taxAmountMinor"`
}

type DocumentResult struct {
	Lines         []LineResult   `bson:"lineItems" json:"lineItems"`
	Totals        Totals         `bson:"totals" json:"totals"`
	TaxBreakdown  []TaxBreakdown `bson:"taxBreakdown" json:"taxBreakdown"`
	PolicyVersion string         `bson:"calculationPolicyVersion" json:"calculationPolicyVersion"`
}

// CalculateDocument validates and calculates each line, then sums already
// rounded line results. Tax breakdown rows are ordered by ascending rate.
func CalculateDocument(lines []LineInput) (DocumentResult, error) {
	if len(lines) > MaxLineItems {
		return DocumentResult{}, validation("lineItems", buhari.CodeValidationFailed, "A document may contain at most 200 line items.")
	}

	result := DocumentResult{
		Lines:         make([]LineResult, 0, len(lines)),
		TaxBreakdown:  make([]TaxBreakdown, 0),
		PolicyVersion: PolicyVersion,
	}
	breakdown := make(map[Rate]TaxBreakdown)

	for index, input := range lines {
		calculated, err := calculateLine(input, index)
		if err != nil {
			return DocumentResult{}, err
		}
		result.Lines = append(result.Lines, LineResult{LineInput: input, Calculated: calculated})

		if !addMoney(&result.Totals.SubtotalMinor, calculated.SubtotalMinor) ||
			!addMoney(&result.Totals.DiscountMinor, calculated.DiscountAmountMinor) ||
			!addMoney(&result.Totals.TaxMinor, calculated.TaxAmountMinor) ||
			!addMoney(&result.Totals.GrandTotalMinor, calculated.LineTotalMinor) {
			return DocumentResult{}, monetaryOverflow()
		}

		row := breakdown[input.TaxRate]
		row.Rate = input.TaxRate
		if !addMoney(&row.TaxableAmountMinor, calculated.DiscountedAmountMinor) ||
			!addMoney(&row.TaxAmountMinor, calculated.TaxAmountMinor) {
			return DocumentResult{}, monetaryOverflow()
		}
		breakdown[input.TaxRate] = row
	}

	for _, row := range breakdown {
		result.TaxBreakdown = append(result.TaxBreakdown, row)
	}
	sort.Slice(result.TaxBreakdown, func(i, j int) bool {
		return result.TaxBreakdown[i].Rate < result.TaxBreakdown[j].Rate
	})

	return result, nil
}

func calculateLine(input LineInput, index int) (LineCalculated, error) {
	prefix := "lineItems[" + integerString(index) + "]"
	description := strings.TrimSpace(input.Description)
	if description == "" || len([]rune(description)) > MaxDescription {
		return LineCalculated{}, validation(prefix+".description", buhari.CodeValidationFailed, "Description must be between 1 and 500 characters.")
	}
	if input.Quantity < 1 || input.Quantity > MaxQuantity {
		return LineCalculated{}, validation(prefix+".quantity", buhari.CodeInvalidQuantity, "Quantity must be a whole number between 1 and 1,000,000.")
	}
	if input.UnitPriceMinor < 0 {
		return LineCalculated{}, validation(prefix+".unitPrice", buhari.CodeInvalidMoneyFormat, "Unit price cannot be negative.")
	}
	if input.TaxRate < 0 || input.TaxRate > RateScale {
		return LineCalculated{}, validation(prefix+".taxRate", buhari.CodeInvalidTaxRate, "Tax rate must be between 0% and 100%.")
	}

	quantity := Money(input.Quantity)
	if input.UnitPriceMinor != 0 && quantity > Money(math.MaxInt64)/input.UnitPriceMinor {
		return LineCalculated{}, monetaryOverflow()
	}
	calculated := LineCalculated{SubtotalMinor: quantity * input.UnitPriceMinor}

	if input.Discount != nil {
		switch input.Discount.Type {
		case DiscountFixed:
			if input.Discount.Value < 0 {
				return LineCalculated{}, validation(prefix+".discount.value", buhari.CodeInvalidMoneyFormat, "Fixed discount cannot be negative.")
			}
			calculated.DiscountAmountMinor = Money(input.Discount.Value)
			if calculated.DiscountAmountMinor > calculated.SubtotalMinor {
				return LineCalculated{}, validation(prefix+".discount.value", buhari.CodeDiscountTooLarge, "Fixed discount cannot exceed the line subtotal.")
			}
		case DiscountPercentage:
			rate := Rate(input.Discount.Value)
			if rate < 0 || rate > RateScale {
				return LineCalculated{}, validation(prefix+".discount.value", buhari.CodeInvalidDiscount, "Percentage discount must be between 0% and 100%.")
			}
			var ok bool
			calculated.DiscountAmountMinor, ok = multiplyRate(calculated.SubtotalMinor, rate)
			if !ok {
				return LineCalculated{}, monetaryOverflow()
			}
		default:
			return LineCalculated{}, validation(prefix+".discount.type", buhari.CodeInvalidDiscountType, "Discount type must be fixed or percentage.")
		}
	}

	calculated.DiscountedAmountMinor = calculated.SubtotalMinor - calculated.DiscountAmountMinor
	var ok bool
	calculated.TaxAmountMinor, ok = multiplyRate(calculated.DiscountedAmountMinor, input.TaxRate)
	if !ok || calculated.DiscountedAmountMinor > Money(math.MaxInt64)-calculated.TaxAmountMinor {
		return LineCalculated{}, monetaryOverflow()
	}
	calculated.LineTotalMinor = calculated.DiscountedAmountMinor + calculated.TaxAmountMinor
	return calculated, nil
}

// multiplyRate multiplies nonnegative money by a scaled rate and rounds half
// away from zero. 128-bit intermediate arithmetic prevents false overflow.
func multiplyRate(amount Money, rate Rate) (Money, bool) {
	hi, lo := bits.Mul64(uint64(amount), uint64(rate))
	quotient, remainder := bits.Div64(hi, lo, uint64(RateScale))
	if remainder*2 >= uint64(RateScale) {
		if quotient == math.MaxUint64 {
			return 0, false
		}
		quotient++
	}
	if quotient > math.MaxInt64 {
		return 0, false
	}
	return Money(quotient), true
}

func addMoney(target *Money, value Money) bool {
	if value < 0 || *target > Money(math.MaxInt64)-value {
		return false
	}
	*target += value
	return true
}

func validation(path string, code buhari.Code, message string) error {
	return buhari.Validation(buhari.FieldError{Path: path, Code: code, Message: message})
}

func monetaryOverflow() error {
	return buhari.New(buhari.CodeMonetaryOverflow, "The calculated monetary value is too large.")
}

func integerString(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}
