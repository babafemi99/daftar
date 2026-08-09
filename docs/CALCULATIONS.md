# DAFTAR (دَفْتَر) - FINANCIAL CALCULATION SPECIFICATION

**Version:** 1.0  
**Date:** 8 August 2026  
**Product:** Daftar  
**Policy ID:** `2026-08-v1`

## Purpose

This document is the authoritative pricing and rounding contract for Daftar. The Go calculation module and its tests must implement these rules exactly. The frontend may format results but must not become the calculation source of truth.

## Representation

### Money

Money is stored as signed 64-bit integer minor units.

```text
USD 100.25 -> 10025 cents
AED 100.25 -> 10025 fils
```

The supported initial currencies all use two minor-unit decimal places. The API accepts monetary inputs as decimal strings such as `"100.25"`, validates at most two decimal places, and converts them to minor units. JSON numbers are not accepted for monetary inputs.

### Quantities

Version 1 supports whole-number quantities only, as required by the assignment:

```text
1 <= quantity <= 1,000,000
```

This avoids ambiguous fractional-unit rounding. Fractional quantities are a documented future extension.

### Rates

Percentage rates use a scale of 1,000,000 units for 100%:

```text
100%    = 1,000,000
10%     =   100,000
5.25%   =    52,500
0.0001% =         1
```

The API accepts a decimal percentage string with at most four fractional percentage digits and converts it to the scaled integer.

## Rounding policy

Each derived monetary amount is rounded to the nearest minor unit using **round half away from zero**. Because valid pricing inputs are nonnegative, an exact half minor unit rounds upward.

Rounding is performed per line:

1. Calculate line subtotal exactly in minor units.
2. Calculate and round percentage discount amount.
3. Calculate discounted amount exactly by subtraction.
4. Calculate and round tax amount.
5. Calculate line total exactly by addition.
6. Sum already-rounded line results for document totals.

This policy matches the assignment's example and makes every displayed line reconcile to the document total.

## Accepted inputs

For each line:

- Non-empty description
- Whole quantity >= 1
- Unit price >= 0
- No discount, fixed discount, or percentage discount
- Tax percentage from 0% to 100%

Limits exist to protect integer overflow and abuse. The service checks multiplication and addition overflow before accepting a result.

## Discount rules

### No discount

```text
discountAmount = 0
discountedAmount = subtotal
```

### Fixed discount

The fixed discount is supplied in minor units.

```text
discountAmount = fixed discount
```

If fixed discount exceeds subtotal, reject the line with `FIXED_DISCOUNT_EXCEEDS_SUBTOTAL`. The value is never silently clamped.

### Percentage discount

```text
discountAmount = round(subtotal x discountRate / 1,000,000)
```

Percentage discount must be between 0% and 100% inclusive.

A line has at most one discount type. Sending both is structurally impossible in the typed API model; malformed payloads are rejected.

## Tax rules

Tax is always applied after discount:

```text
discountedAmount = subtotal - discountAmount
taxAmount = round(discountedAmount x taxRate / 1,000,000)
lineTotal = discountedAmount + taxAmount
```

Tax rates must be between 0% and 100% inclusive. No jurisdiction-specific compounding or inclusive-tax mode exists in version 1.

## Per-line algorithm

Given:

```text
quantity = Q
unit price minor = P
discount configuration = D
tax rate = T
rate scale = 1,000,000
```

Calculate:

```text
subtotal = Q x P

if no discount:
    discount = 0
if fixed discount:
    discount = fixed amount
if percentage discount:
    discount = round(subtotal x D / rate scale)

discounted = subtotal - discount
tax = round(discounted x T / rate scale)
line total = discounted + tax
```

## Document aggregation

For all calculated lines:

```text
document subtotal       = sum(line subtotal)
document discount       = sum(line discount amount)
document tax            = sum(line tax amount)
document grand total    = sum(line total)
```

The following invariant must always hold:

```text
grand total = subtotal - total discount + total tax
```

## Tax breakdown

Lines are grouped by their exact scaled tax rate:

```text
taxable amount at rate = sum(line discounted amount)
tax amount at rate     = sum(line tax amount)
```

Zero-rated lines are included with rate `0`, taxable amount, and zero tax so the PDF explains all amounts.

## Assignment sample

### Widget A

```text
2 x 100.00              = 200.00 subtotal
10% discount            =  20.00 discount
discounted amount       = 180.00
5% tax on 180.00        =   9.00 tax
line total              = 189.00
```

### Widget B

```text
1 x 50.00               =  50.00 subtotal
no discount             =   0.00 discount
discounted amount       =  50.00
5% tax                  =   2.50 tax
line total              =  52.50
```

### Service fee

```text
1 x 200.00              = 200.00 subtotal
fixed 20.00 discount    =  20.00 discount
discounted amount       = 180.00
no tax                  =   0.00 tax
line total              = 180.00
```

### Totals

```text
subtotal                = 450.00
total discount          =  40.00
total tax               =  11.50
grand total             = 421.50
```

## Rounding examples

### Percentage discount boundary

```text
subtotal = 10.05
discount = 10%
raw discount = 1.005
rounded discount = 1.01
```

Tax is then calculated on `9.04`, not on the unrounded intermediate.

### Small tax

```text
discounted amount = 0.05
tax = 5%
raw tax = 0.0025
rounded tax = 0.00
```

Per-line rounding may differ from computing tax once on the document total. The application intentionally uses per-line rounding so displayed lines reconcile exactly.

## Zero-value behaviour

- Unit price may be zero.
- A 100% discount is allowed and produces zero discounted amount, zero tax, and zero line total.
- A 0% discount is valid but normalized to no effective discount for calculation.
- A 0% tax rate produces a zero-rated tax-breakdown row.

## Currency rules

- Every document has exactly one currency.
- Every line inherits the document currency.
- Currency cannot change after finalization.
- Reports filter or group by currency.
- Values from different currencies are never summed.
- No exchange-rate conversion is performed.

## Recalculation rules

The server recalculates the entire document when:

- A document is created.
- Any draft metadata affecting output changes.
- A line is added, edited, reordered, or removed.
- A document is finalized.
- A finalized document is duplicated into a new draft.

PDFs and reports consume stored server-calculated results. Finalization performs one final recalculation and stores the policy version.

## Validation error codes

| Code | Condition |
|---|---|
| `INVALID_MONEY_FORMAT` | Money is not a valid two-decimal string |
| `INVALID_QUANTITY` | Quantity is below 1, non-integer, or above limit |
| `INVALID_DISCOUNT_TYPE` | Discount type is unsupported |
| `INVALID_DISCOUNT_RATE` | Percentage discount is outside 0-100% |
| `FIXED_DISCOUNT_EXCEEDS_SUBTOTAL` | Fixed discount is larger than line subtotal |
| `INVALID_TAX_RATE` | Tax is outside 0-100% |
| `MONETARY_OVERFLOW` | Calculation exceeds the supported integer range |
| `DOCUMENT_REQUIRES_LINE` | Finalization attempted without a line |

## Required calculation tests

1. Assignment sample equals `450.00 / 40.00 / 11.50 / 421.50`.
2. No discount and no tax.
3. Fixed discount followed by tax.
4. Percentage discount followed by tax.
5. Fixed discount equal to subtotal.
6. Fixed discount greater than subtotal rejected.
7. 100% percentage discount.
8. 0% tax and tax-breakdown inclusion.
9. Half-minor-unit rounding boundary.
10. Multiple lines whose individually rounded tax differs from aggregate rounding.
11. Document invariant and tax-breakdown invariant.
12. Maximum safe values and overflow rejection.
