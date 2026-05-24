package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveName(t *testing.T) {
	cases := map[string]string{
		"jellyfin.example.com":   "Jellyfin",
		"argo-cd.example.com":    "Argo Cd",
		"foo.bar.baz.example.io": "Foo",
		"single":                 "Single",
		"":                       "",
	}
	for in, want := range cases {
		assert.Equal(t, want, DeriveName(in), "input=%q", in)
	}
}

func TestDeriveIconSlug(t *testing.T) {
	assert.Equal(t, "jellyfin", DeriveIconSlug("Jellyfin.example.com"))
	assert.Equal(t, "argo-cd", DeriveIconSlug("argo-cd.example.com"))
	assert.Equal(t, "", DeriveIconSlug(""))
}

func TestDeriveURL(t *testing.T) {
	assert.Equal(t, "https://jellyfin.example.com",
		DeriveURL("jellyfin.example.com"))
	assert.Equal(t, "", DeriveURL(""))
}
