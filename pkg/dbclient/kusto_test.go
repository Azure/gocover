package dbclient

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/Azure/azure-kusto-go/kusto/ingest"
	"github.com/sirupsen/logrus"
)

func TestKustoOptionValidate(t *testing.T) {
	t.Run("require credentials", func(t *testing.T) {
		oldTenantID := os.Getenv(tenantIDKey)
		oldClientID := os.Getenv(clientIDKey)
		oldClientSecret := os.Getenv(clientSecretKey)
		os.Unsetenv(tenantIDKey)
		os.Unsetenv(clientIDKey)
		os.Unsetenv(clientSecretKey)
		defer func() {
			os.Setenv(tenantIDKey, oldTenantID)
			os.Setenv(clientIDKey, oldClientID)
			os.Setenv(clientSecretKey, oldClientSecret)
		}()

		o := &KustoOption{UseKusto: true}
		if err := o.Validate(); err == nil {
			t.Errorf("%s", tenantIDKey)
		}

		os.Setenv(tenantIDKey, "tenant-id")
		if err := o.Validate(); err == nil {
			t.Errorf("%s", clientIDKey)
		}
		if o.tenantID != "tenant-id" {
			t.Errorf("expect tenant id of option %s, but %s", "tenant-id", o.tenantID)
		}

		os.Setenv(clientIDKey, "client-id")
		if err := o.Validate(); err == nil {
			t.Errorf("%s", clientSecretKey)
		}
		if o.clientID != "client-id" {
			t.Errorf("expect tenant id of option %s, but %s", "client-id", o.clientID)
		}

		os.Setenv(clientSecretKey, "client-secret")
		if err := o.Validate(); err == nil {
			t.Error("endpoint")
		}
		if o.clientSecret != "client-secret" {
			t.Errorf("expect tenant id of option %s, but %s", "client-secret", o.clientSecret)
		}

		o.Endpoint = "fake.kusto.windows.net"
		if err := o.Validate(); err == nil {
			t.Error("database")
		}

		o.Database = "database"
		if err := o.Validate(); err == nil {
			t.Error("event")
		}

		o.CoverageEvent = "cover-event"
		if err := o.Validate(); err == nil {
			t.Errorf("should success, but get %s", err)
		}

		o.IgnoreEvent = "ignore-event"
		if err := o.Validate(); err != nil {
			t.Errorf("should success, but get %s", err)
		}

		failedColumns := []string{"buildID", "buildID::123456", ":int:123456", "::"}
		for _, col := range failedColumns {
			o.CustomColumns = []string{col}
			if err := o.Validate(); err == nil {
				t.Errorf("for custom column [%s], should return error, but return nil", col)
			}
		}

		succeededColumns := []string{"buildID:string:123456", "buildID:string:", "buildID:string"}
		for _, col := range succeededColumns {
			o.CustomColumns = []string{col}
			if err := o.Validate(); err != nil {
				t.Errorf("for custom column [%s], should return nil, but return error: %s", col, err)
			}
		}

	})
}

func TestKustoClient(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	goodIngestor := &mockIngestor{
		fromFileFn: func(ctx context.Context, fPath string, options ...ingest.FileOption) (*ingest.Result, error) {
			return &ingest.Result{}, nil
		},
		fromReaderFn: func(ctx context.Context, reader io.Reader, options ...ingest.FileOption) (*ingest.Result, error) {
			return &ingest.Result{}, nil
		},
	}
	badIngestor := &mockIngestor{
		fromFileFn: func(ctx context.Context, fPath string, options ...ingest.FileOption) (*ingest.Result, error) {
			return nil, errors.New("from file failed")
		},
		fromReaderFn: func(ctx context.Context, reader io.Reader, options ...ingest.FileOption) (*ingest.Result, error) {
			return nil, errors.New("from reader failed")
		},
	}

	t.Run("StoreCoverageData", func(t *testing.T) {
		t.Run("store succeeded", func(t *testing.T) {
			client := KustoClient{
				coverageIngestor: goodIngestor,
				ignoreIngestor:   goodIngestor,
				mappings:         []mapping{},
				extraData:        map[string]interface{}{},
				logger:           logger,
			}
			if err := client.StoreCoverageData(ctx, &CoverageData{}); err != nil {
				t.Errorf("should return nil, but return %s", err)
			}
		})

		t.Run("store failed", func(t *testing.T) {
			client := KustoClient{
				coverageIngestor: badIngestor,
				ignoreIngestor:   badIngestor,
				mappings:         []mapping{},
				extraData:        map[string]interface{}{},
				logger:           logger,
			}
			if err := client.StoreCoverageData(ctx, &CoverageData{}); err == nil {
				t.Error("should return error, but return nil")
			}
		})
	})

	t.Run("StoreIgnoreProfileData", func(t *testing.T) {
		t.Run("store succeeded", func(t *testing.T) {
			client := KustoClient{
				coverageIngestor: goodIngestor,
				ignoreIngestor:   goodIngestor,
				mappings:         []mapping{},
				extraData:        map[string]interface{}{},
				logger:           logger,
			}
			if err := client.StoreIgnoreProfileData(ctx, &IgnoreProfileData{}); err != nil {
				t.Errorf("should return nil, but return %s", err)
			}
		})

		t.Run("store failed", func(t *testing.T) {
			client := KustoClient{
				coverageIngestor: badIngestor,
				ignoreIngestor:   badIngestor,
				mappings:         []mapping{},
				extraData:        map[string]interface{}{},
				logger:           logger,
			}
			if err := client.StoreIgnoreProfileData(ctx, &IgnoreProfileData{}); err == nil {
				t.Error("should return error, but return nil")
			}
		})
	})

}

type mockIngestor struct {
	fromFileFn   func(ctx context.Context, fPath string, options ...ingest.FileOption) (*ingest.Result, error)
	fromReaderFn func(ctx context.Context, reader io.Reader, options ...ingest.FileOption) (*ingest.Result, error)
}

func (i *mockIngestor) Close() error {
	return nil
}

func (i *mockIngestor) FromFile(ctx context.Context, fPath string, options ...ingest.FileOption) (*ingest.Result, error) {
	return i.fromFileFn(ctx, fPath, options...)
}

func (i *mockIngestor) FromReader(ctx context.Context, reader io.Reader, options ...ingest.FileOption) (*ingest.Result, error) {
	return i.fromReaderFn(ctx, reader, options...)
}
