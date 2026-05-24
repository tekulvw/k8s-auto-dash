package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeID(t *testing.T) {
	assert.Equal(t, "media/jellyfin/jellyfin.example.com",
		ComputeID("media", "jellyfin", "jellyfin.example.com"))
}

func TestComputeID_EmptyHostnamePanicsCaller(t *testing.T) {
	// Empty hostnames should never reach this function; document via test.
	assert.Equal(t, "ns/route/", ComputeID("ns", "route", ""))
}
