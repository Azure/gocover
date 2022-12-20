// kusto.go is the kusto client wrapper of the library
// "github.com/Azure/azure-kusto-go/kusto"
package dbclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/ingest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"
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

	DEFAULT_MAX_RETRY                = 5
	DEFAULT_RETRY_TIMEOUT_IN_SECONDS = 10 * time.Second

	Separator = ":"
)

func NewKustoClient(option *KustoOption) (DbClient, error) {

	authorizer := kusto.Authorization{
		Config: auth.NewClientCredentialsConfig(option.clientID, option.clientSecret, option.tenantID),
	}

	kustoClient, err := kusto.New(option.Endpoint, authorizer)
	if err != nil {
		return nil, fmt.Errorf("kusto: %w", err)
	}

	coverageIngestor, err := ingest.New(kustoClient, option.Database, option.CoverageEvent)
	if err != nil { //+gocover:ignore:block cannot test kusto connection without enough credentials
		return nil, fmt.Errorf("coverage ingestor: %w", err)
	}

	ignoreIngestor, err := ingest.New(kustoClient, option.Database, option.IgnoreEvent)
	//+gocover:ignore:block cannot test kusto connection without enough credentials

	if err != nil { //+gocover:ignore:block cannot test kusto connection without enough credentials
		return nil, fmt.Errorf("ignore ingestor: %w", err)
	}

	return &KustoClient{ //+gocover:ignore:block cannot test kusto connection without enough credentials
		coverageIngestor: coverageIngestor,
		ignoreIngestor:   ignoreIngestor,
		mappings:         option.extraMappings,
		extraData:        option.extraData,
		logger:           option.Logger.WithField("source", "KustoClient"),
	}, nil

}

// KustoClient wraps the kusto ingestor and the extra column data and corresponding mappings.
type KustoClient struct {
	coverageIngestor ingest.Ingestor
	ignoreIngestor   ingest.Ingestor
	mappings         []mapping
	extraData        map[string]interface{}
	logger           logrus.FieldLogger
}

var _ DbClient = (*KustoClient)(nil)

func (client *KustoClient) StoreCoverageDataFromFile(ctx context.Context, data []*CoverageData) error {
	file, err := os.CreateTemp("", "coveragedata")
	if err != nil {
		return err
	}

	for _, d := range data {
		d.Extra = client.extraData
		contents, err := json.Marshal(&d)
		if err != nil {
			return err
		}
		if _, err := file.Write(contents); err != nil {
			return err
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	defer func() {
		client.logger.Debugf("clean coverage data file %s", file.Name())
		_ = os.Remove(file.Name())
	}()

	mappings := append(basicCoverageMappings, client.mappings...)
	mappingsBytes, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("mappings json marshal: %w", err)
	}

	return ingestFileWithRetry(ctx, client.logger, client.coverageIngestor, file.Name(), string(mappingsBytes), DEFAULT_MAX_RETRY)
}

func (client *KustoClient) StoreIgnoreProfileDataFromFile(ctx context.Context, data []*IgnoreProfileData) error {
	file, err := os.CreateTemp("", "ignoreprofiledata")
	if err != nil {
		return err
	}

	for _, d := range data {
		d.Extra = client.extraData
		contents, err := json.Marshal(&d)
		if err != nil {
			return err
		}
		if _, err := file.Write(contents); err != nil {
			return err
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	defer func() {
		client.logger.Debugf("clean ignore profile data file %s", file.Name())
		_ = os.Remove(file.Name())
	}()

	mappings := append(basicIgnoreProfileMappings, client.mappings...)
	mappingsBytes, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("mappings json marshal: %w", err)
	}

	return ingestFileWithRetry(ctx, client.logger, client.ignoreIngestor, file.Name(), string(mappingsBytes), DEFAULT_MAX_RETRY)
}

func ingestFileWithRetry(ctx context.Context, logger logrus.FieldLogger, ingestor ingest.Ingestor, dataFilePath, mappings string, retryLimit int) error {

	for retryCount := 0; retryCount < retryLimit; retryCount++ {
		logger.Infof("ingestion %s starting", dataFilePath)
		result, err := ingestor.FromFile(
			ctx, dataFilePath,
			ingest.FileFormat(ingest.JSON),
			ingest.IngestionMapping(mappings, ingest.JSON),
			ingest.ReportResultToTable(),
		)
		if err != nil {
			logger.WithError(err).Errorf("failed to create Kusto client, will retry in %s", DEFAULT_RETRY_TIMEOUT_IN_SECONDS)
			time.Sleep(DEFAULT_RETRY_TIMEOUT_IN_SECONDS)
			retryCount++
			continue
		}

		err = <-result.Wait(ctx)
		if err != nil {
			// the operation complete with an error
			if ingest.IsRetryable(err) {
				// Handle retries
				logger.WithError(err).
					Errorf("Failed to create Kusto client, will retry in %s", DEFAULT_RETRY_TIMEOUT_IN_SECONDS)
				time.Sleep(DEFAULT_RETRY_TIMEOUT_IN_SECONDS)
				retryCount++
				continue
			} else {
				logger.WithError(err).Errorf("Failed to create Kusto client, error is not retriable")
				return err
			}
		}
		logger.Infof("Ingestion %s completed", dataFilePath)
		return nil
	}

	return fmt.Errorf("max ingestion retries count %d reached", retryLimit)
}

func (client *KustoClient) StoreCoverageData(ctx context.Context, data *CoverageData) error {
	data.Extra = client.extraData
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("data json marshal: %w", err)
	}
	err = store(ctx,
		client.coverageIngestor,
		dataBytes,
		append(basicCoverageMappings, client.mappings...),
		client.logger.WithField("ingestor", "coverage"),
	)
	if err != nil {
		return fmt.Errorf("store coverage data: %w", err)
	}
	return nil
}

func (client *KustoClient) StoreIgnoreProfileData(ctx context.Context, data *IgnoreProfileData) error {
	data.Extra = client.extraData
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("data json marshal: %w", err)
	}
	err = store(ctx,
		client.ignoreIngestor,
		dataBytes,
		append(basicIgnoreProfileMappings, client.mappings...),
		client.logger.WithField("ingestor", "ignoreProfile"),
	)
	if err != nil {
		return fmt.Errorf("store ignore profile data: %w", err)
	}
	return nil
}

func store(ctx context.Context,
	ingestor ingest.Ingestor,
	dataBytes []byte,
	mappings []mapping,
	logger logrus.FieldLogger,
) error {
	mappingsBytes, err := json.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("mappings json marshal: %w", err)
	}

	_, err = ingestor.FromReader(
		ctx,
		bytes.NewReader(dataBytes),
		ingest.FileFormat(ingest.JSON),
		ingest.IngestionMapping(mappingsBytes, ingest.JSON),
	)
	if err != nil {
		return fmt.Errorf("ingestor from reader %w", err)
	}

	logger.Debugf("send to kusto: %s\n", string(dataBytes))
	return nil
}

// KustoOption wraps the credential and kusto server information for building kusto client.
type KustoOption struct {
	UseKusto      bool
	Endpoint      string
	Database      string
	CoverageEvent string
	IgnoreEvent   string
	CustomColumns []string
	Logger        logrus.FieldLogger

	tenantID     string
	clientID     string
	clientSecret string

	extraData     map[string]interface{}
	extraMappings []mapping
}

// Validate checks the validation of the input on kusto option.
func (o *KustoOption) Validate() error {
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
	if o.CoverageEvent == "" {
		return fmt.Errorf("%s %w", "coverage-event", ErrFlagRequired)
	}
	if o.IgnoreEvent == "" {
		return fmt.Errorf("%s %w", "ignore-event", ErrFlagRequired)
	}

	// each custom column has format: {column}:{datatype}:{value}
	// token 0: column name
	// token 1: datatype
	// token 2: column value
	for _, col := range o.CustomColumns {
		decColumn, err := base64.StdEncoding.DecodeString(col)
		if err != nil {
			return fmt.Errorf("invalid base64: %w", err)
		}

		m := string(decColumn)

		tokens := strings.Split(string(decColumn), Separator)
		if len(tokens) < 2 {
			return fmt.Errorf("%s %w", m, ErrFormatCustomColumn)
		}

		var messages []string
		if tokens[0] == "" {
			messages = append(messages, "column name is empty")
		}
		if tokens[1] == "" {
			messages = append(messages, "datatype is empty")
		}
		if len(messages) != 0 {
			return fmt.Errorf("%s: %w, %s", m, ErrFormatCustomColumn, strings.Join(messages, ","))
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
		if len(tokens) < 3 {
			o.extraData[tokens[0]] = ""
		} else {
			o.extraData[tokens[0]] = tokens[2]
		}
	}

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

// basicCoverageMappings gives the fundemental mappings for Data struct and kusto table
var basicCoverageMappings = []mapping{
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
		Column:   "totalLines",
		Datatype: "long",
		Properties: properties{
			Path: "$.totalLines",
		},
	},
	{
		Column:   "effectiveLines",
		Datatype: "long",
		Properties: properties{
			Path: "$.effectiveLines",
		},
	},
	{
		Column:   "ignoredLines",
		Datatype: "long",
		Properties: properties{
			Path: "$.ignoredLines",
		},
	},
	{
		Column:   "coveredLines",
		Datatype: "long",
		Properties: properties{
			Path: "$.coveredLines",
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
	{
		Column:   "coverageMode",
		Datatype: "string",
		Properties: properties{
			Path: "$.coverageMode",
		},
	},
}

var basicIgnoreProfileMappings = []mapping{
	{
		Column:   "preciseTimestamp",
		Datatype: "datetime",
		Properties: properties{
			Path: "$.preciseTimestamp",
		},
	},
	{
		Column:   "filePath",
		Datatype: "string",
		Properties: properties{
			Path: "$.filePath",
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
		Column:   "annotation",
		Datatype: "string",
		Properties: properties{
			Path: "$.annotation",
		},
	},
	{
		Column:   "lineNumber",
		Datatype: "int",
		Properties: properties{
			Path: "$.lineNumber",
		},
	},
	{
		Column:   "startLine",
		Datatype: "int",
		Properties: properties{
			Path: "$.startLine",
		},
	},
	{
		Column:   "endLine",
		Datatype: "int",
		Properties: properties{
			Path: "$.endLine",
		},
	},
	{
		Column:   "comments",
		Datatype: "string",
		Properties: properties{
			Path: "$.comments",
		},
	},
	{
		Column:   "contents",
		Datatype: "string",
		Properties: properties{
			Path: "$.contents",
		},
	},
	{
		Column:   "ignoreType",
		Datatype: "string",
		Properties: properties{
			Path: "$.ignoreType",
		},
	},
}
