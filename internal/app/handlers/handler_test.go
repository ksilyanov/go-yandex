package handlers

import (
	"bytes"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/middlewares/compressor"
	"go-yandex/internal/app/middlewares/cookiemanager"
	"go-yandex/internal/app/storage"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRouter(t *testing.T) {
	ctx := context.Background()
	curConfig, err := config.GetConfig()
	require.NoError(t, err)

	repository := storage.New(curConfig, ctx)
	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		compressor.GzipHandler,
		cookiemanager.Handler,
	)

	r.Post("/", SaveURL(repository, curConfig))
	r.Get("/{id}", GetURL(repository))
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", SaveURLJson(repository, curConfig))
		r.Post("/shorten/batch", SaveBatch(repository, curConfig))
		r.Get("/user/urls", GetForUser(repository, curConfig))
	})
	r.Get("/ping", GetDBStatus(repository))

	ts := httptest.NewServer(r)
	defer ts.Close()

	var (
		resp     *http.Response
		respBody string
	)

	addClientCookie(t, ts)

	unixNowStr := strconv.FormatInt(time.Now().UnixMicro(), 10)
	testLinkBase := "http://www.google.ru/search?q=test_url_"
	link1 := testLinkBase + "1_" + unixNowStr
	link2 := testLinkBase + "2_" + unixNowStr
	link3 := testLinkBase + "3_" + unixNowStr
	var shortLink1, shortLink2, shortLink3 string

	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/", link1)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Regexp(t, curConfig.BaseURL+"/.+", respBody)
	shortLink1 = strings.Replace(respBody, curConfig.BaseURL+"/", "", 1)

	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/", link1)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, curConfig.BaseURL+"/"+shortLink1, respBody)

	resp, respBody = testRequest(t, ts, curConfig, http.MethodGet, "/"+shortLink1, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	//assert.Equal(t, link1, resp.Header.Get("Location"))

	resp, respBody = testRequest(t, ts, curConfig, http.MethodGet, "/not_registered_url", "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/api/shorten", "{\"url\":\""+link2+"\"}")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Regexp(t, "{\"result\":\""+curConfig.BaseURL+"/.+\"}", respBody)
	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/", link2)
	shortLink2 = strings.Replace(respBody, curConfig.BaseURL+"/", "", 1)

	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/api/shorten", "{\"url\":\""+link2+"\"}")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "{\"result\":\""+curConfig.BaseURL+"/"+shortLink2+"\"}\n", respBody)

	resp, respBody = testRequest(t, ts, curConfig, http.MethodGet, "/api/user/urls", "")
	testItem1 := "{\"short_url\":\"" + curConfig.BaseURL + "/" + shortLink1 + "\",\"original_url\":\"" + link1 + "\"}"
	testItem2 := "{\"short_url\":\"" + curConfig.BaseURL + "/" + shortLink2 + "\",\"original_url\":\"" + link2 + "\"}"
	testUserURLs := "[" + testItem1 + "," + testItem2 + "]\n"
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, testUserURLs, respBody)

	var testBatchBody, expectedBatchResponse string

	testBatchBody = "[{\"correlation_id\":\"123\",\"original_url\":\"" + link3 + "\"}]"
	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/api/shorten/batch", testBatchBody)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	_, shortLink3Full := testRequest(t, ts, curConfig, http.MethodPost, "/", link3)
	shortLink3 = strings.Replace(shortLink3Full, curConfig.BaseURL+"/", "", 1)
	expectedBatchResponse = "[{\"correlation_id\":\"123\",\"short_url\":\"" + curConfig.BaseURL + "/" + shortLink3 + "\"}]\n"
	//assert.Equal(t, expectedBatchResponse, respBody)

	testBatchBody = "[{\"correlation_id\":\"456\",\"original_url\":\"" + link3 + "\"}]"
	expectedBatchResponse = "[{\"correlation_id\":\"456\",\"short_url\":\"" + curConfig.BaseURL + "/" + shortLink3 + "\"}]\n"
	resp, respBody = testRequest(t, ts, curConfig, http.MethodPost, "/api/shorten/batch", testBatchBody)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	//assert.Equal(t, expectedBatchResponse, respBody)
	println(expectedBatchResponse)

	resp, respBody = testRequest(t, ts, curConfig, http.MethodGet, "/ping", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func addClientCookie(t *testing.T, ts *httptest.Server) {
	cookie, err := cookiemanager.GenerateCookie()
	require.NoError(t, err)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	ts.Client().Jar = jar

	ts.Client().Jar.SetCookies(&url.URL{Host: ts.URL}, []*http.Cookie{cookie})
}
func testRequest(t *testing.T, ts *httptest.Server, config config.Config, method, path string, body string) (*http.Response, string) {
	var reqContent *bytes.Buffer
	var content = []byte(body)
	reqContent = bytes.NewBuffer(content)

	req, err := http.NewRequest(method, ts.URL+path, reqContent)
	assert.NoError(t, err)

	resp, err := ts.Client().Do(req)
	assert.NoError(t, err)

	respBody, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	defer resp.Body.Close()

	return resp, string(respBody)
}
