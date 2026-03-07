package model

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type Date struct {
	time.Time
}

func MarshalDate(t Date) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format("2006-01-02")))
	})
}

func UnmarshalDate(v any) (Date, error) {
	switch v := v.(type) {
	case string:
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return Date{}, fmt.Errorf("parsing date: %w", err)
		}
		return Date{Time: t}, nil
	default:
		return Date{}, fmt.Errorf("date must be a string, got %T", v)
	}
}

type DateTime struct {
	time.Time
}

func MarshalDateTime(t DateTime) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format(time.RFC3339)))
	})
}

func UnmarshalDateTime(v any) (DateTime, error) {
	switch v := v.(type) {
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return DateTime{}, fmt.Errorf("parsing datetime: %w", err)
		}
		return DateTime{Time: t}, nil
	default:
		return DateTime{}, fmt.Errorf("datetime must be a string, got %T", v)
	}
}
