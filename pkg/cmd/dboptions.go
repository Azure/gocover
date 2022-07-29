package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/Azure/gocover/pkg/dbclient"
)

var ErrUnsupportedDBType = errors.New("unsupported DB client type")

type DBOption struct {
	DataCollectionEnabled bool
	DbType                dbclient.ClientType
	KustoOption           dbclient.KustoOption
	Writer                io.Writer
}

func (o *DBOption) Validate() error {
	if !o.DataCollectionEnabled {
		return nil
	}

	if o.DbType == dbclient.Kusto {
		return o.KustoOption.Validate()
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedDBType, o.DbType)
}

func (o *DBOption) GetDbClient() (dbclient.DbClient, error) {
	if o.DbType == dbclient.Kusto {
		o.KustoOption.Writer = o.Writer
		return dbclient.NewKustoClient(&o.KustoOption)
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedDBType, o.DbType)
}
