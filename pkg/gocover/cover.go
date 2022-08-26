package gocover

import "context"

type GoCover interface {
	Run(ctx context.Context) error
}
