package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const resUpload = "/canned/upload"

func readGoldenFile(t *testing.T) []byte {
	bytes, err := ioutil.ReadFile(filepath.Join("testdata", t.Name()+".golden"))
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}
	return bytes
}

func commonTestResponseUpload(t *testing.T, endpoint string, expectedCode int) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	fileData := readGoldenFile(t)
	var err error
	bytes := bytes.NewBuffer(fileData)
	c.Request, err = http.NewRequest("POST", resUpload, bytes)
	if err != nil {
		t.Fatal("failed to create test request")
	}
	var p gin.Params
	p = append(p, gin.Param{Key: "endpoint", Value: endpoint})
	c.Params = p
	endpointRouter(c)

	if rec.Code != expectedCode {
		t.Fatalf("expected %v but received %v", expectedCode, rec.Code)
	}
}

func commonTestResponseUploadFile(t *testing.T, formKey string, expectedCode int) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	fileData := readGoldenFile(t)
	var err error
	b := bytes.NewBuffer(fileData)
	const fileUpload = "/canned/upload/file"
	c.Request, err = http.NewRequest("POST", fileUpload, b)
	if err != nil {
		t.Fatal("failed to create test request")
	}
	var p gin.Params
	p = append(p, gin.Param{Key: "endpoint", Value: fileUpload})
	c.Params = p

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(formKey, filepath.Join("testdata", t.Name()+".golden"))
	if err != nil {
		t.Fatal("failed to create form file")
	}
	_, err = io.Copy(part, bytes.NewReader(fileData))

	err = writer.Close()
	if err != nil {
		t.Fatal("failed to close form file")
	}

	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request.Body = ioutil.NopCloser(body)
	endpointRouter(c)

	if rec.Code != expectedCode {
		t.Fatalf("expected %v but received %v", expectedCode, rec.Code)
	}
}

func commonTestGetResponse(t *testing.T, endpoint, query string) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	fileData := readGoldenFile(t)
	var err error
	bytes := bytes.NewBuffer(fileData)
	c.Request, err = http.NewRequest("POST", resUpload, bytes)
	if err != nil {
		t.Fatal("failed to create test request")
	}
	var p gin.Params
	p = append(p, gin.Param{Key: "endpoint", Value: resUpload})
	c.Params = p
	endpointRouter(c)

	var q gin.Params
	q = append(q, gin.Param{Key: "endpoint", Value: endpoint})
	c.Params = q

	if len(query) != 0 {
		c.Request.RequestURI = query
	}

	endpointRouter(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %v but received %v", http.StatusOK, rec.Code)
	}
}

func TestStoreResponseFromFileWhenNoFilePresentWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(""))
}

func TestStoreResponseFromFileWhenInvalidJSONFileWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))
}

func TestStoreResponseFromFileWhenMissingResponseCodeWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))
}

func TestStoreResponseFromFileWhenInvalidResponseCodeWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))
}

func TestStoreResponseFromFileWhenInvalidTimeoutWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))
}

func TestStoreResponseFromFileWhenMissingEndpointWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))
}

func TestStoreResponseFromFileWhenMissingMethodWillReturnError(t *testing.T) {
	assert.NotNil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))
}

func TestStoreResponseFromFile(t *testing.T) {
	b := readGoldenFile(t)
	assert.Nil(t, storeResponsesFromFile(filepath.Join("testdata", t.Name()+".golden")))

	var expectedResponses responses
	err := json.Unmarshal(b, &expectedResponses)
	if err != nil {
		t.Fatalf("failed to unmarshal golden file %s", t.Name())
	}

	if !reflect.DeepEqual(expectedResponses.Responses, cachedResponses) {
		expectedString, err := json.Marshal(expectedResponses.Responses)
		if err != nil {
			t.Errorf("failed to create string from expected responses")
		}
		actualString, err := json.Marshal(cachedResponses)
		if err != nil {
			t.Errorf("failed to create string from actual responses")
		}
		t.Errorf("unexpected responses returned, expected %s, got %s",
			expectedString, actualString)
	}
}

func TestResponseUploadWhenInvalidJSONWillReturnBadRequest(t *testing.T) {
	commonTestResponseUpload(t, resUpload, http.StatusBadRequest)
}

func TestResponseUploadWillStoreResponse(t *testing.T) {
	commonTestResponseUpload(t, resUpload, http.StatusOK)
}

func TestFileUploadWhenIncorrectKeyWillReturnBadRequest(t *testing.T) {
	commonTestResponseUploadFile(t, "differentKey", http.StatusBadRequest)
}

func TestFileUploadWhenInvalidJSONWillReturnBadRequest(t *testing.T) {
	commonTestResponseUploadFile(t, "responses", http.StatusBadRequest)
}

func TestFileUploadWillStoreResponse(t *testing.T) {
	commonTestResponseUploadFile(t, "responses", http.StatusOK)
}

func TestGetResponseWhenKnownEndpoint(t *testing.T) {
	commonTestGetResponse(t, "/dummy/ep1", "")
}

func TestGetResponseWhenRegexMatch(t *testing.T) {
	commonTestGetResponse(t, "/dummy/ep2", "")
}

func TestGetResponseWhenGraphqlRequest(t *testing.T) {
	commonTestGetResponse(t, "/graphql", "{people{name}}")
}

func TestGetResponseWhenGraphqlDoesNotMatch(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	var err error
	c.Request, err = http.NewRequest("POST", resUpload, nil)
	if err != nil {
		t.Fatal("failed to create test request")
	}
	var p gin.Params
	p = append(p, gin.Param{Key: "endpoint", Value: "/graphql"})
	c.Params = p

	c.Request.RequestURI = "{user{name}}"

	endpointRouter(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %v but received %v", http.StatusNotFound, rec.Code)
	}
}

func TestResponseUploadWhenNoResponseMatchWillReturnNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	var err error
	c.Request, err = http.NewRequest("POST", resUpload, nil)
	if err != nil {
		t.Fatal("failed to create test request")
	}
	var p gin.Params
	p = append(p, gin.Param{Key: "endpoint", Value: "/unknownendpoint"})
	c.Params = p
	endpointRouter(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %v but received %v", http.StatusNotFound, rec.Code)
	}
}

func TestGetResponseWhenTimeoutSpecifiedWillWaitBeforeResponding(t *testing.T) {
	timedOut := false
	go func() {
		time.Sleep(time.Duration(2) * time.Second)
		timedOut = true
	}()
	commonTestGetResponse(t, "/dummy/ep1", "")
	if !timedOut {
		t.Fail()
	}
}

func TestGetResponseWhenRegexAppliedToRequestBody(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	fileData := readGoldenFile(t)
	var err error
	bytes := bytes.NewBuffer(fileData)
	c.Request, err = http.NewRequest("POST", resUpload, bytes)
	if err != nil {
		t.Fatal("failed to create test request")
	}
	var p gin.Params
	p = append(p, gin.Param{Key: "endpoint", Value: resUpload})
	c.Params = p
	endpointRouter(c)

	var q gin.Params
	q = append(q, gin.Param{Key: "endpoint", Value: "/"})
	c.Params = q

	c.Request.Body = ioutil.NopCloser(strings.NewReader("Action=ReceiveMessage&MaxNumberOfMessages=5&VisibilityTimeout=15&AttributeName=All&Expires=2020-04-18T22%3A52%3A43PST&Version=2012-11-05&AUTHPARAMS"))

	endpointRouter(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %v but received %v", http.StatusOK, rec.Code)
	}
}
