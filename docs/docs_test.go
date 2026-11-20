package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSwaggerJSON(t *testing.T) {
	raw := GetSwaggerJSON("localhost:8081", "http")
	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &spec))
	assert.Equal(t, "localhost:8081", spec["host"])
	assert.Equal(t, []interface{}{"http"}, spec["schemes"])
	assert.Equal(t, "2.0", spec["swagger"])
	assert.NotEmpty(t, spec["paths"])
}

func TestGetSwaggerJSONDynamicHost(t *testing.T) {
	raw := GetSwaggerJSON("api.example.com", "https")
	assert.Contains(t, string(raw), "api.example.com")
	assert.Contains(t, string(raw), `"https"`)
	assert.NotContains(t, string(raw), "{{HOST}}")
}
