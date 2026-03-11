package model

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
)

type UUID = uuid.UUID

func MarshalUUID(u UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", u.String())
	})
}

func UnmarshalUUID(v any) (UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.Parse(v)
	default:
		return uuid.UUID{}, fmt.Errorf("uuid must be a string, got %T", v)
	}
}
