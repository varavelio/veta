package sourcefile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAllowed verifies the source file name whitelist.
func TestAllowed(t *testing.T) {
	tests := []struct {
		name    string
		allowed bool
	}{
		{name: "site.js", allowed: true},
		{name: "site_data.yaml", allowed: true},
		{name: "123.json", allowed: true},
		{name: "site.test.js", allowed: false},
		{name: "site-name.js", allowed: false},
		{name: "site name.js", allowed: false},
		{name: ".site.js", allowed: false},
		{name: "site.js~", allowed: false},
		{name: "site", allowed: false},
		{name: "site.", allowed: false},
		{name: "site.j-s", allowed: false},
		{name: "sitio_á.js", allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.allowed, Allowed(test.name))
		})
	}
}
