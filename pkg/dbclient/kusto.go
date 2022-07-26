// kusto.go is the kusto client wrapper of the library
// "github.com/Azure/azure-kusto-go/kusto"
package dbclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/ingest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var (
	ErrEnvRequired        = errors.New("environment is required for kusto db")
	ErrFlagRequired       = errors.New("flag is required for kusto db")
	ErrFormatCustomColumn = errors.New("wrong format, kusto custom column format is {column}:{datatype}:{value}")
)

const (
	// The required credentials used to authenticate on kusto.
	tenantIDKey     string = "KUSTO_TENANT_ID"
	clientIDKey     string = "KUSTO_CLIENT_ID"
	clientSecretKey string = "KUSTO_CLIENT_SECRET"

	Separator = ":"
)

func NewKustoClient(option *KustoOption) (DbClient, error) {

	authorizer := kusto.Authorization{
		Config: auth.NewClientCredentialsConfig(option.clientID, option.clientSecret, option.tenantID),
	}

	kustoClient, err := kusto.New(option.Endpoint, authorizer)
	if err != nil {
		return nil, fmt.Errorf("kusto: %s", err)
	}

	in, err := ingest.New(kustoClient, option.Database, option.Event)
	if err != nil {
		return nil, fmt.Errorf("ingest: %s", err)
	}

	return &KustoClient{
		ingestor:  in,
		mappings:  append(basicMappings, option.extraMappings...),
		extraData: option.extraData,
		w:         option.Writer,
	}, nil

}

// KustoClient wraps the kusto ingestor and the extra column data and corresponding mappings.
type KustoClient struct {
	ingestor  *ingest.Ingestion
	mappings  []mapping
	extraData map[string]interface{}
	w         io.Writer
}

var _ DbClient = (*KustoClient)(nil)

// Store stores the data to kusto.
func (client *KustoClient) Store(ctx context.Context, data *Data) error {
	data.Extra = client.extraData
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("data json marshal: %w", err)
	}

	mappingsBytes, err := json.Marshal(client.mappings)
	if err != nil {
		return fmt.Errorf("mappings json marshal: %w", err)
	}

	_, err = client.ingestor.FromReader(
		ctx,
		bytes.NewReader(dataBytes),
		ingest.FileFormat(ingest.JSON),
		ingest.IngestionMapping(mappingsBytes, ingest.JSON),
	)
	if err != nil {
		return fmt.Errorf("ingestor from reader %w", err)
	}

	fmt.Fprintf(client.w, "send to kusto: %s\n", string(dataBytes))

	return nil
}

// properties used for kusto transform on json data.
type properties struct {
	Path      string `json:"Path"`
	Transform string `json:"Transform,omitempty"`
}

// mapping used to build mapping between kusto column and json data field.
type mapping struct {
	Column     string     `json:"Column"`
	Datatype   string     `json:"Datatype,omitempty"`
	Properties properties `json:"Properties"`
}

// basicMappings gives the fundemental mappings for Data struct and kusto table
var basicMappings = []mapping{
	{
		Column:   "preciseTimestamp",
		Datatype: "datetime",
		Properties: properties{
			Path: "$.preciseTimestamp",
		},
	},
	{
		Column:   "coverage",
		Datatype: "real",
		Properties: properties{
			Path: "$.coverage",
		},
	},
	{
		Column:   "linesCovered",
		Datatype: "long",
		Properties: properties{
			Path: "$.linesCovered",
		},
	},
	{
		Column:   "linesValid",
		Datatype: "long",
		Properties: properties{
			Path: "$.linesValid",
		},
	},
	{
		Column:   "modulePath",
		Datatype: "string",
		Properties: properties{
			Path: "$.modulePath",
		},
	},
	{
		Column:   "filePath",
		Datatype: "string",
		Properties: properties{
			Path: "$.filePath",
		},
	},
}

// KustoOption wraps the credential and kusto server information for building kusto client.
type KustoOption struct {
	UseKusto      bool
	Endpoint      string
	Database      string
	Event         string
	CustomColumns []string
	Writer        io.Writer

	tenantID     string
	clientID     string
	clientSecret string

	extraData     map[string]interface{}
	extraMappings []mapping
}

// Validate checks the validation of the input on kusto option.
func (o *KustoOption) Validate() error {
	// don't use kusto as storage, return directly
	// if !o.UseKusto {
	// 	return nil
	// }

	if o.tenantID = os.Getenv(tenantIDKey); o.tenantID == "" {
		return fmt.Errorf("%s %w", tenantIDKey, ErrEnvRequired)
	}

	if o.clientID = os.Getenv(clientIDKey); o.clientID == "" {
		return fmt.Errorf("%s %w", clientIDKey, ErrEnvRequired)
	}

	if o.clientSecret = os.Getenv(clientSecretKey); o.clientSecret == "" {
		return fmt.Errorf("%s %w", clientSecretKey, ErrEnvRequired)
	}

	if o.Endpoint == "" {
		return fmt.Errorf("%s %w", "endpoint", ErrFlagRequired)
	}
	if o.Database == "" {
		return fmt.Errorf("%s %w", "database", ErrFlagRequired)
	}
	if o.Event == "" {
		return fmt.Errorf("%s %w", "event", ErrFlagRequired)
	}
	// fallback writer to io buffer
	if o.Writer == nil {
		o.Writer = &bytes.Buffer{}
	}

	// each custom column has format: {column}:{datatype}:{value}
	// token 0: column name
	// token 1: datatype
	// token 2: column value
	for _, m := range o.CustomColumns {
		tokens := strings.Split(m, Separator)
		if len(tokens) != 3 {
			return fmt.Errorf("%s %w", m, ErrFormatCustomColumn)
		}

		if tokens[0] == "" || tokens[1] == "" || tokens[2] == "" {
			return fmt.Errorf("empty custom column string")
		}

		// build extra data kusto mapping
		o.extraMappings = append(o.extraMappings, mapping{
			Column:   tokens[0],
			Datatype: tokens[1],
			Properties: properties{
				Path: fmt.Sprintf("$.Extra.%s", tokens[0]),
			},
		})

		if o.extraData == nil {
			o.extraData = make(map[string]interface{})
		}

		// add extra data to final data struct
		o.extraData[tokens[0]] = tokens[2]
	}

	return nil
}
