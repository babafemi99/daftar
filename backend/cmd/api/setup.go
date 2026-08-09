package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/sys"
	"github.com/babafemi99/daftar/backend/internal/transport/web"
)

type application struct {
	API          *web.API
	Dependencies *sys.Dependencies
}

// setup is the process composition root. All required dependencies are ready
// before it returns, so main can safely start listening or exit immediately.
func setup(ctx context.Context, config cfg.Config) (*application, error) {
	dependencies, err := sys.NewDependencies(ctx, config)
	if err != nil {
		return nil, err
	}

	api := web.NewAPIWithLogger(dependencies.Config.HTTP, slog.Default(), dependencies.Config.Logging.SlowRequestThreshold)
	if err := api.ConfigureAuth(
		dependencies.UserService,
		dependencies.Config.JWT,
		dependencies.Config.Cookie,
	); err != nil {
		_ = dependencies.Shutdown(context.Background())
		return nil, err
	}
	if err := api.ConfigureSessions(dependencies.SessionService); err != nil {
		_ = dependencies.Shutdown(context.Background())
		return nil, err
	}
	if err := api.ConfigureDocuments(dependencies.DocumentService); err != nil {
		_ = dependencies.Shutdown(context.Background())
		return nil, err
	}
	if err := api.ConfigureReports(dependencies.ReportService); err != nil {
		_ = dependencies.Shutdown(context.Background())
		return nil, err
	}
	if err := api.ConfigureAudits(dependencies.AuditService); err != nil {
		_ = dependencies.Shutdown(context.Background())
		return nil, err
	}

	return &application{API: api, Dependencies: dependencies}, nil
}

func (a *application) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var errs []error
	if a.API != nil {
		errs = append(errs, a.API.Shutdown(ctx))
	}
	if a.Dependencies != nil {
		errs = append(errs, a.Dependencies.Shutdown(ctx))
	}
	return errors.Join(errs...)
}
