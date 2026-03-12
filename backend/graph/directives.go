package graph

import (
	"context"

	"dayos/graph/model"
	"dayos/validate"

	"github.com/99designs/gqlgen/graphql"
)

// ValidateDirective returns the @validate directive handler.
// It sanitizes and validates string fields based on the TextFieldRule.
func ValidateDirective() func(ctx context.Context, obj any, next graphql.Resolver, rule model.TextFieldRule) (any, error) {
	return func(ctx context.Context, obj any, next graphql.Resolver, rule model.TextFieldRule) (any, error) {
		val, err := next(ctx)
		if err != nil {
			return nil, err
		}
		s, ok := val.(string)
		if !ok || s == "" {
			return val, nil
		}
		var r validate.TextFieldRule
		switch rule {
		case model.TextFieldRuleSingleLine:
			r = validate.SingleLine
		case model.TextFieldRuleSingleLineShort:
			r = validate.SingleLineShort
		case model.TextFieldRulePlainText:
			r = validate.PlainText
		case model.TextFieldRuleChatMessage:
			r = validate.ChatMessage
		}
		fieldName := graphql.GetPathContext(ctx).Field
		return validate.Validate(r, *fieldName, s)
	}
}
