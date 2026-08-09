package observability

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/cfg"
)

func TestNewLoggerWritesJSONWithPermanentFields(t *testing.T) {
	var output bytes.Buffer
	config := cfg.Default()
	config.ServiceName = "daftar-api"
	config.Environment = "test"
	config.Logging.Format = "json"
	logger := NewLogger(&output, config)

	logger.Info("ready", "address", ":8080")

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if record["service"] != "daftar-api" || record["environment"] != "test" || record["msg"] != "ready" {
		t.Fatalf("log record = %#v", record)
	}
}
