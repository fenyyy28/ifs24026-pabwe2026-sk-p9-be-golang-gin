// Package docs berisi spesifikasi OpenAPI/Swagger Delcom Blog API.
package docs

import (
	_ "embed"
	"strings"
)

//go:embed swagger.json
var swaggerTemplate string

// GetSwaggerJSON mengembalikan spesifikasi OpenAPI dengan host & skema yang
// menyesuaikan secara dinamis terhadap pemanggil.
func GetSwaggerJSON(host, scheme string) []byte {
	spec := strings.ReplaceAll(swaggerTemplate, "{{HOST}}", host)
	spec = strings.ReplaceAll(spec, "{{SCHEME}}", scheme)
	return []byte(spec)
}
