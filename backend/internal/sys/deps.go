// Package sys wires and owns Daftar's long-lived application dependencies.
package sys

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/service"
)

type Dependencies struct {
	Config          cfg.Config
	Repositories    *mercury.Core
	UserService     *service.UserService
	DocumentService *service.DocumentService
	ReportService   *service.ReportService
	AuditService    *service.AuditService
	SessionService  *service.SessionService
}

// NewDependencies validates configuration, connects infrastructure, ensures
// database indexes, constructs services, and optionally creates the bootstrap
// user. It returns only after every required dependency is ready.
func NewDependencies(ctx context.Context, config cfg.Config) (*Dependencies, error) {
	config.HTTP = config.HTTP.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	connectTimeout := config.MongoDB.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second
	}
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	repositories, err := mercury.Connect(connectCtx, config.MongoDB.URI, config.MongoDB.Database)
	if err != nil {
		return nil, err
	}
	slog.Info("mongodb connected", "database", config.MongoDB.Database)
	cleanupOnError := func() {
		_ = repositories.Shutdown(context.Background())
	}

	if err := repositories.EnsureSchemaValidators(connectCtx); err != nil {
		cleanupOnError()
		return nil, err
	}
	slog.Info("mongodb schema validators ensured")
	if err := repositories.EnsureIndexes(connectCtx); err != nil {
		cleanupOnError()
		return nil, err
	}
	slog.Info("mongodb indexes ensured")
	users, err := service.NewUserService(repositories.User())
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	if err := ensureBootstrapUser(ctx, users, config.BootstrapUser); err != nil {
		cleanupOnError()
		return nil, err
	}
	documents, err := service.NewAuditedDocumentService(repositories.Document(), repositories.DocumentReference(), repositories.Audit(), repositories)
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	reports, err := service.NewReportService(repositories.Report())
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	audits, err := service.NewAuditService(repositories.Audit(), repositories.Document())
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	sessions, err := service.NewSessionService(repositories.RefreshSession(), repositories.User(), repositories, config.JWT.RefreshTTL)
	if err != nil {
		cleanupOnError()
		return nil, err
	}

	return &Dependencies{
		Config:          config,
		Repositories:    repositories,
		UserService:     users,
		DocumentService: documents,
		ReportService:   reports,
		AuditService:    audits,
		SessionService:  sessions,
	}, nil
}

func (d *Dependencies) Shutdown(ctx context.Context) error {
	if d == nil || d.Repositories == nil {
		return nil
	}
	return d.Repositories.Shutdown(ctx)
}

type userRegistrar interface {
	Register(ctx context.Context, request *model.CreateUserRequest) (*model.User, error)
}

func ensureBootstrapUser(ctx context.Context, users userRegistrar, bootstrap cfg.BootstrapUser) error {
	if !bootstrap.Enabled {
		return nil
	}
	_, err := users.Register(ctx, &model.CreateUserRequest{
		Email: bootstrap.Email, Password: bootstrap.Password,
		FirstName: bootstrap.FirstName, LastName: bootstrap.LastName,
	})
	if err == nil {
		return nil
	}
	var appErr *buhari.Error
	if errors.As(err, &appErr) && appErr.Code == buhari.CodeEmailAlreadyRegistered {
		return nil
	}
	return err
}
