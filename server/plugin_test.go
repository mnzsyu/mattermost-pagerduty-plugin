package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestServeHTTP(t *testing.T) {
	assert := assert.New(t)

	// Create a mock API
	api := &plugintest.API{}

	// Set up the plugin
	plugin := Plugin{}
	plugin.SetAPI(api)

	// Create a test HTTP request to /api/v1/hello
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	r.Header.Set("Mattermost-User-ID", "test-user-id") // Add this header for authorization

	// Create a test HTTP response recorder
	w := httptest.NewRecorder()

	// Call the ServeHTTP method
	plugin.ServeHTTP(nil, w, r)

	// Get the response
	result := w.Result()
	assert.NotNil(result)
	defer result.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(result.Body)
	assert.Nil(err)
	bodyString := string(bodyBytes)

	// Check that we got the expected response
	assert.Equal("Hello, world!", bodyString)
}
