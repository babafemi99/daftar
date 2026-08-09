package mercury

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var (
	ErrDatabaseRequired    = errors.New("mercury: database is required")
	ErrMongoURIRequired    = errors.New("mercury: MongoDB URI is required")
	ErrTransactionRequired = errors.New("mercury: transaction callback is required")
)

type Transaction func(ctx context.Context) error

type CoreRepository interface {
	Shutdown(ctx context.Context) error
	RunInTx(ctx context.Context, fn Transaction) error
	EnsureSchemaValidators(ctx context.Context) error
	EnsureIndexes(ctx context.Context) error
	User() Users
	Document() Documents
	DocumentReference() DocumentReferences
	Report() Reports
	Audit() Audits
	RefreshSession() RefreshSessions
}

// Core is the central entry point to Daftar's repositories. Application
// wiring creates one Core and passes the required repository interfaces to
// services and handlers.
type Core struct {
	client             *mongo.Client
	database           *mongo.Database
	Users              *UserRepository
	Documents          *DocumentRepository
	DocumentReferences *DocumentReferenceRepository
	Reports            *ReportRepository
	Audits             *AuditRepository
	RefreshSessions    *RefreshSessionRepository
}

var _ CoreRepository = (*Core)(nil)

// Connect creates and verifies MongoDB, then wires all Mercury repositories.
// It is intended to be called once by application configuration/bootstrap.
func Connect(ctx context.Context, uri, databaseName string) (*Core, error) {
	if strings.TrimSpace(uri) == "" {
		return nil, ErrMongoURIRequired
	}
	if strings.TrimSpace(databaseName) == "" {
		return nil, ErrDatabaseRequired
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mercury: connect to MongoDB: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mercury: ping MongoDB: %w", err)
	}

	return NewCore(client.Database(databaseName))
}

func NewCore(database *mongo.Database) (*Core, error) {
	if database == nil {
		return nil, ErrDatabaseRequired
	}

	return &Core{
		client:             database.Client(),
		database:           database,
		Users:              NewUserRepository(database),
		Documents:          NewDocumentRepository(database),
		DocumentReferences: NewDocumentReferenceRepository(database),
		Reports:            NewReportRepository(database),
		Audits:             NewAuditRepository(database),
		RefreshSessions:    NewRefreshSessionRepository(database),
	}, nil
}

func (c *Core) User() Users {
	if c == nil {
		return nil
	}
	return c.Users
}

func (c *Core) Document() Documents {
	if c == nil {
		return nil
	}
	return c.Documents
}

func (c *Core) DocumentReference() DocumentReferences {
	if c == nil {
		return nil
	}
	return c.DocumentReferences
}

func (c *Core) Report() Reports {
	if c == nil {
		return nil
	}
	return c.Reports
}

func (c *Core) Audit() Audits {
	if c == nil {
		return nil
	}
	return c.Audits
}

func (c *Core) RefreshSession() RefreshSessions {
	if c == nil {
		return nil
	}
	return c.RefreshSessions
}

func (c *Core) Shutdown(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mercury: disconnect MongoDB: %w", err)
	}
	return nil
}

// RunInTx executes fn in a MongoDB transaction. The driver may retry fn for
// transient failures, so transaction callbacks must be idempotent and must
// return every database operation error.
func (c *Core) RunInTx(ctx context.Context, fn Transaction) error {
	if c == nil || c.client == nil {
		return ErrDatabaseRequired
	}
	if fn == nil {
		return ErrTransactionRequired
	}

	session, err := c.client.StartSession()
	if err != nil {
		return fmt.Errorf("mercury: start transaction session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		return nil, fn(txCtx)
	})
	if err != nil {
		return fmt.Errorf("mercury: run transaction: %w", err)
	}

	return nil
}

// EnsureIndexes initializes the indexes owned by all Mercury repositories.
// Add each new repository's index initialization here.
func (c *Core) EnsureIndexes(ctx context.Context) error {
	if c == nil || c.Users == nil || c.Documents == nil || c.DocumentReferences == nil || c.Reports == nil || c.Audits == nil || c.RefreshSessions == nil {
		return ErrDatabaseRequired
	}
	if err := c.Users.EnsureIndexes(ctx); err != nil {
		return err
	}
	if err := c.Documents.EnsureIndexes(ctx); err != nil {
		return err
	}
	if err := c.DocumentReferences.EnsureIndexes(ctx); err != nil {
		return err
	}
	if err := c.Reports.EnsureIndexes(ctx); err != nil {
		return err
	}
	if err := c.Audits.EnsureIndexes(ctx); err != nil {
		return err
	}
	return c.RefreshSessions.EnsureIndexes(ctx)
}
