package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/middlewares/cookieManager"
	"go-yandex/internal/app/storage"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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

	currentConfig, err := config.GetConfig()
	require.NoError(t, err)

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

	var testRep = storage.New(currentConfig)
	curCookie, err := cookieManager.GenerateCookie()
	require.NoError(t, err)

	for _, tc := range urlsOrder {
		request := httptest.NewRequest(tc.method, currentConfig.BaseURL+"/"+tc.path, bytes.NewBufferString(tc.bodyStr))
		writer := httptest.NewRecorder()

		ctx := context.WithValue(request.Context(), cookieManager.CookieName, curCookie.Value)
		request = request.WithContext(ctx)

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
		writer.Flush()

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

	request := httptest.NewRequest(http.MethodGet, currentConfig.BaseURL+"/user/urls", nil)
	writer := httptest.NewRecorder()
	ctx := context.WithValue(request.Context(), cookieManager.CookieName, curCookie.Value)
	request = request.WithContext(ctx)

	GetForUser(testRep, currentConfig)(writer, request)
	result := writer.Result()
	writer.Flush()

	respBody, err := ioutil.ReadAll(result.Body)
	require.NoError(t, err)
	result.Body.Close()

	var some []storage.ItemUrls
	err = json.Unmarshal(respBody, &some)
	require.NoError(t, err)
	assert.Equal(t, urlsOrder[1].bodyStr, some[1].FullURL)
	assert.Equal(t, currentConfig.BaseURL+"/"+urlsOrder[1].expectedPath, some[1].ShortURL)

	newCookie, err := cookieManager.GenerateCookie()
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodGet, currentConfig.BaseURL+"/user/urls", nil)
	writer = httptest.NewRecorder()
	ctx = context.WithValue(request.Context(), cookieManager.CookieName, newCookie.Value)
	request = request.WithContext(ctx)

	GetForUser(testRep, currentConfig)(writer, request)
	result = writer.Result()
	writer.Flush()

	var urlsList []byte
	_, err = result.Body.Read(urlsList)
	require.NoError(t, err)
	result.Body.Close()

	assert.Equal(t, "", string(urlsList))
	assert.Equal(t, http.StatusNoContent, result.StatusCode)

	os.Remove(currentConfig.FileStoragePath)
}
