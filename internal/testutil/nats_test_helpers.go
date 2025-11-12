package testutil

import (
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	NatsTestURLEnv     = "INTEGRATION_NATS_URL"
	defaultNatsTestURL = "nats://127.0.0.1:4222"
)

func ConnectToNATS(t *testing.T) *nats.Conn {
	t.Helper()

	url := os.Getenv(NatsTestURLEnv)
	if url == "" {
		url = defaultNatsTestURL
	}

	nc, err := nats.Connect(url, nats.Timeout(2*time.Second))
	if err != nil {
		t.Skipf("не удалось подключиться к NATS (%s): %v. Запустите `docker compose up -d nats`", url, err)
	}

	if err := nc.FlushTimeout(2 * time.Second); err != nil {
		_ = nc.Drain()
		nc.Close()
		t.Skipf("нет ответа от NATS (%s): %v. Запустите `docker compose up -d nats`", url, err)
	}

	t.Cleanup(func() {
		_ = nc.Drain()
		nc.Close()
	})

	return nc
}

func RequireJetStream(t *testing.T, nc *nats.Conn) nats.JetStreamContext {
	t.Helper()

	js, err := nc.JetStream()
	if err != nil {
		t.Skipf("JetStream недоступен: %v", err)
	}

	return js
}
