package mercury

import (
	"context"
	"sort"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const reportSummaryIndex = "documents_owner_status_archive_issue_date_currency"

type Reports interface {
	Summary(ctx context.Context, ownerID string, from, to time.Time) ([]CurrencySummary, error)
}

type CurrencySummary struct {
	Currency      model.Currency
	DocumentCount int64
	Subtotal      calculations.Money
	TotalDiscount calculations.Money
	TotalTax      calculations.Money
	GrandTotal    calculations.Money
	TaxBreakdown  []calculations.TaxBreakdown
}

type ReportRepository struct {
	collection *mongo.Collection
}

var _ Reports = (*ReportRepository)(nil)

func NewReportRepository(database *mongo.Database) *ReportRepository {
	return &ReportRepository{collection: database.Collection(model.DocumentCollection)}
}

func (r *ReportRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "ownerId", Value: 1},
			{Key: "status", Value: 1},
			{Key: "archivedAt", Value: 1},
			{Key: "issueDate", Value: 1},
			{Key: "currency", Value: 1},
		},
		Options: options.Index().SetName(reportSummaryIndex),
	})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to initialize report storage.", err)
	}
	return nil
}

func (r *ReportRepository) Summary(ctx context.Context, ownerID string, from, to time.Time) ([]CurrencySummary, error) {
	match := bson.D{{Key: "$match", Value: bson.D{
		{Key: "ownerId", Value: ownerID},
		{Key: "status", Value: model.DocumentFinalized},
		{Key: "archivedAt", Value: nil},
		{Key: "issueDate", Value: bson.D{{Key: "$gte", Value: from.UTC()}, {Key: "$lt", Value: to.UTC().AddDate(0, 0, 1)}}},
	}}}
	totals := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$currency"},
			{Key: "documentCount", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "subtotal", Value: bson.D{{Key: "$sum", Value: "$totals.subtotalMinor"}}},
			{Key: "totalDiscount", Value: bson.D{{Key: "$sum", Value: "$totals.discountMinor"}}},
			{Key: "totalTax", Value: bson.D{{Key: "$sum", Value: "$totals.taxMinor"}}},
			{Key: "grandTotal", Value: bson.D{{Key: "$sum", Value: "$totals.grandTotalMinor"}}},
		}}},
	}
	taxes := mongo.Pipeline{
		bson.D{{Key: "$unwind", Value: "$taxBreakdown"}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "currency", Value: "$currency"}, {Key: "rate", Value: "$taxBreakdown.rate"}}},
			{Key: "taxableAmount", Value: bson.D{{Key: "$sum", Value: "$taxBreakdown.taxableAmountMinor"}}},
			{Key: "taxAmount", Value: bson.D{{Key: "$sum", Value: "$taxBreakdown.taxAmountMinor"}}},
		}}},
	}
	pipeline := mongo.Pipeline{
		match,
		bson.D{{Key: "$facet", Value: bson.D{{Key: "totals", Value: totals}, {Key: "taxes", Value: taxes}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to calculate report summary.", err)
	}
	defer cursor.Close(ctx)
	var rows []summaryAggregation
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to decode report summary.", err)
	}
	if len(rows) == 0 {
		return []CurrencySummary{}, nil
	}

	summaries := make(map[model.Currency]*CurrencySummary, len(rows[0].Totals))
	for _, total := range rows[0].Totals {
		summaries[total.Currency] = &CurrencySummary{
			Currency: total.Currency, DocumentCount: total.DocumentCount,
			Subtotal: total.Subtotal, TotalDiscount: total.TotalDiscount,
			TotalTax: total.TotalTax, GrandTotal: total.GrandTotal,
			TaxBreakdown: make([]calculations.TaxBreakdown, 0),
		}
	}
	for _, tax := range rows[0].Taxes {
		if summary := summaries[tax.ID.Currency]; summary != nil {
			summary.TaxBreakdown = append(summary.TaxBreakdown, calculations.TaxBreakdown{
				Rate: tax.ID.Rate, TaxableAmountMinor: tax.TaxableAmount, TaxAmountMinor: tax.TaxAmount,
			})
		}
	}
	result := make([]CurrencySummary, 0, len(summaries))
	for _, summary := range summaries {
		sort.Slice(summary.TaxBreakdown, func(i, j int) bool { return summary.TaxBreakdown[i].Rate < summary.TaxBreakdown[j].Rate })
		result = append(result, *summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Currency < result[j].Currency })
	return result, nil
}

type summaryAggregation struct {
	Totals []struct {
		Currency      model.Currency     `bson:"_id"`
		DocumentCount int64              `bson:"documentCount"`
		Subtotal      calculations.Money `bson:"subtotal"`
		TotalDiscount calculations.Money `bson:"totalDiscount"`
		TotalTax      calculations.Money `bson:"totalTax"`
		GrandTotal    calculations.Money `bson:"grandTotal"`
	} `bson:"totals"`
	Taxes []struct {
		ID struct {
			Currency model.Currency    `bson:"currency"`
			Rate     calculations.Rate `bson:"rate"`
		} `bson:"_id"`
		TaxableAmount calculations.Money `bson:"taxableAmount"`
		TaxAmount     calculations.Money `bson:"taxAmount"`
	} `bson:"taxes"`
}
