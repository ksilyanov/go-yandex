package handlers

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/storage"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type urls []struct {
	path             string
	method           string
	bodyStr          string
	expectedStatus   int
	expectedPath     string
	expectedLocation string
}

func TestRouter(t *testing.T) {

	currentConfig := config.GetConfig()

	urlsOrder := urls{
		{
			"/",
			http.MethodPost,
			"https://yandex.ru",
			http.StatusCreated,
			"1",
			"",
		},
		{
			"/",
			http.MethodPost,
			"https://www.google.com",
			http.StatusCreated,
			"2",
			"",
		},
		{
			"/api/shorten",
			http.MethodPost,
			"{\"url\":\"https://github.com\"}",
			http.StatusCreated,
			"{\"result\":\"" + currentConfig.BaseURL + "/3\"}\n",
			"",
		},
		{
			"1",
			http.MethodGet,
			"",
			http.StatusTemporaryRedirect,
			"",
			"https://yandex.ru",
		},
		{
			"2",
			http.MethodGet,
			"",
			http.StatusTemporaryRedirect,
			"",
			"https://www.google.com",
		},
	}

	var testRep = storage.New()
	for _, tc := range urlsOrder {
		request := httptest.NewRequest(tc.method, currentConfig.BaseURL+"/"+tc.path, bytes.NewBufferString(tc.bodyStr))
		writer := httptest.NewRecorder()

		if tc.method == http.MethodPost {
			if tc.path == "/api/shorten" {
				SaveURLJson(testRep, currentConfig)(writer, request)
			} else {
				SaveURL(testRep, currentConfig)(writer, request)
			}
		}

		if tc.method == http.MethodGet {
			GetURL(testRep)(writer, request)
		}

		result := writer.Result()

		respBody, err := ioutil.ReadAll(result.Body)
		require.NoError(t, err)

		result.Body.Close()

		assert.Equal(t, tc.expectedStatus, result.StatusCode)

		if tc.method == http.MethodPost {
			if tc.path == "/api/shorten" {
				assert.Equal(t, tc.expectedPath, string(respBody))
			} else {
				assert.Equal(t, currentConfig.BaseURL+"/"+tc.expectedPath, string(respBody))
			}
		}

		if tc.method == http.MethodGet {
			assert.Equal(t, tc.expectedLocation, result.Header.Get("Location"))
		}
	}
}
