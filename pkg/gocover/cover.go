package gocover

import "context"

// GoCover interface to generate coverage result.
type GoCover interface {
	Run(ctx context.Context) error
}
