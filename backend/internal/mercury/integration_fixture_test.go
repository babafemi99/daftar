//go:build integration

package mercury

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/testcontainers/testcontainers-go"
	mongotestcontainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var integrationMongoClient *mongo.Client

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	container, err := mongotestcontainer.Run(ctx, "mongo:8.0", mongotestcontainer.WithReplicaSet("rs0"))
	if err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "start MongoDB test container: %v\n", err)
		os.Exit(1)
	}

	uri, err := container.ConnectionString(ctx)
	if err == nil {
		// A single-node test replica set advertises its container hostname.
		// Direct mode keeps the host-side client on Testcontainers' mapped port.
		integrationMongoClient, err = mongo.Connect(options.Client().ApplyURI(uri).SetDirect(true))
	}
	if err == nil {
		err = integrationMongoClient.Ping(ctx, readpref.Primary())
	}
	cancel()
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		fmt.Fprintf(os.Stderr, "connect to MongoDB test container: %v\n", err)
		os.Exit(1)
	}

	status := m.Run()
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if integrationMongoClient != nil {
		_ = integrationMongoClient.Disconnect(cleanupCtx)
	}
	cleanupCancel()
	if err := testcontainers.TerminateContainer(container); err != nil {
		fmt.Fprintf(os.Stderr, "terminate MongoDB test container: %v\n", err)
		if status == 0 {
			status = 1
		}
	}
	os.Exit(status)
}

func newMongoTestDatabase(t *testing.T) *mongo.Database {
	t.Helper()
	if integrationMongoClient == nil {
		t.Fatal("MongoDB integration fixture is not initialized")
	}
	name := strings.ReplaceAll("daftar_test_"+lid.NewRequest(), "-", "_")
	database := integrationMongoClient.Database(name)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := database.Drop(ctx); err != nil {
			t.Errorf("drop integration database %s: %v", name, err)
		}
	})
	return database
}
