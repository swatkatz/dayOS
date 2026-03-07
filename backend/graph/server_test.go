package graph_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dayos/graph"

	"github.com/99designs/gqlgen/graphql/handler"
)

func TestGraphQLIntrospection(t *testing.T) {
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Pool: nil},
	}))

	ts := httptest.NewServer(srv)
	defer ts.Close()

	query := `{"query":"{ __schema { queryType { name } } }"}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(query))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Schema struct {
				QueryType struct {
					Name string `json:"name"`
				} `json:"queryType"`
			} `json:"__schema"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if result.Data.Schema.QueryType.Name != "Query" {
		t.Fatalf("expected query type 'Query', got %q", result.Data.Schema.QueryType.Name)
	}
}
