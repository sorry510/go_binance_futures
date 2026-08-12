package middlewares

import "testing"

func TestMCPRouteRequiresJWT(t *testing.T) {
	for _, route := range excludeRoutes {
		matched, err := pathMatch("/mcp", route)
		if err != nil {
			t.Fatalf("pathMatch() failed: %v", err)
		}
		if matched {
			t.Fatalf("MCP route must not be excluded from JWT authentication: %q", route)
		}
	}
}
