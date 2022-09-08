package dbclient

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestDBOption(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		// disable data collection
		o := &DBOption{DataCollectionEnabled: false, DbType: None}
		if err := o.Validate(); err != nil {
			t.Errorf("disable data collection, should pass with nil, but get %s", err)
		}

		// enable data collection
		o.DataCollectionEnabled = true
		if err := o.Validate(); err == nil {
			t.Errorf("dbtype not support, should return error: %s, but get nil", ErrUnsupportedDBType)
		}

		// missing kusto related information
		o.DbType = Kusto
		if err := o.Validate(); err == nil {
			t.Error("does not provide appropriate information, should return error, but get nil")
		}
	})

	t.Run("GetDbClient", func(t *testing.T) {
		t.Run("NewKustoClient", func(t *testing.T) {
			o := &DBOption{
				DbType: Kusto,
				KustoOption: KustoOption{
					tenantID:      "testTenantID",
					clientID:      "testClientID",
					clientSecret:  "test123456",
					Endpoint:      "https://fake.kusto.windows.net",
					Database:      "TestDB",
					CoverageEvent: "TestCoverageTable",
					IgnoreEvent:   "TestIgnoreTable",
				},
			}
			// ignore everything deliberately, for kusto, cannot test connection if provides correct credentials
			_, _ = o.GetDbClient(logrus.New())

			o.KustoOption.Endpoint = "https://ingest-.westus.kusto.windows.net"
			_, err := o.GetDbClient(logrus.New())
			if err == nil {
				t.Errorf("incorrect endpoint %s should return error", o.KustoOption.Endpoint)
			}
		})

		t.Run("default unsupported", func(t *testing.T) {
			o := &DBOption{DbType: None}
			_, err := o.GetDbClient(logrus.New())
			if err == nil {
				t.Error("unsupported dbtype should return error, but return nil")
			}
		})
	})
}
