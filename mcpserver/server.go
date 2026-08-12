package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ServerName         = "go-binance-futures-http"
	ServerVersion      = "1.0.0"
	DefaultHTTPPath    = "/mcp"
	DefaultHealthzPath = "/healthz"
)

type authorizationContextKey struct{}

type HTTPOptions struct {
	JSONResponse bool
	Stateless    bool
}

func NewServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	registerAPITools(server)

	return server
}

func NewHTTPHandler(server *mcp.Server, opts HTTPOptions) http.Handler {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		JSONResponse: opts.JSONResponse,
		Stateless:    opts.Stateless,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), authorizationContextKey{}, r.Header.Get("Authorization"))
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authorizationFromContext(ctx context.Context) string {
	authorization, _ := ctx.Value(authorizationContextKey{}).(string)
	return authorization
}

func decodeInput[T any](req *mcp.CallToolRequest) (T, error) {
	var input T
	if req == nil || len(req.Params.Arguments) == 0 {
		return input, nil
	}
	dec := json.NewDecoder(bytes.NewReader(req.Params.Arguments))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return input, err
	}
	return input, nil
}

func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

func mustSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	return schema
}
