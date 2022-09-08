package dbclient

import (
	"os"
	"testing"
)

func TestKustoOptionValidate(t *testing.T) {

	t.Run("require credentials", func(t *testing.T) {
		o := &KustoOption{UseKusto: true}
		err := o.Validate()
		if err == nil {
			t.Errorf("%s", tenantIDKey)
		}

		os.Setenv(tenantIDKey, "tenant-id")
		err = o.Validate()
		if err == nil {
			t.Errorf("%s", clientIDKey)
		}
		if o.tenantID != "tenant-id" {
			t.Errorf("expect tenant id of option %s, but %s", "tenant-id", o.tenantID)
		}

		os.Setenv(clientIDKey, "client-id")
		err = o.Validate()
		if err == nil {
			t.Errorf("%s", clientSecretKey)
		}
		if o.clientID != "client-id" {
			t.Errorf("expect tenant id of option %s, but %s", "client-id", o.clientID)
		}

		os.Setenv(clientSecretKey, "client-secret")
		err = o.Validate()
		if o.clientSecret != "client-secret" {
			t.Errorf("expect tenant id of option %s, but %s", "client-secret", o.clientSecret)
		}
		if err == nil {
			t.Error("endpoint")
		}

		o.Endpoint = "fake.kusto.windows.net"
		err = o.Validate()
		if err == nil {
			t.Error("database")
		}

		o.Database = "database"
		err = o.Validate()
		if err == nil {
			t.Error("event")
		}

		o.CoverageEvent = "cover-event"
		err = o.Validate()
		if err == nil {
			t.Errorf("should success, but get %s", err)
		}

		o.IgnoreEvent = "ignore-event"
		err = o.Validate()
		if err != nil {
			t.Errorf("should success, but get %s", err)
		}

		o.CustomColumns = []string{": :"}
		err = o.Validate()
		if err == nil {
			t.Errorf("wrong custom column")
		}

		o.CustomColumns = []string{"newColumn:string:foo"}
		err = o.Validate()
		if err != nil {
			t.Errorf("should success, but get %s", err)
		}
	})
}
