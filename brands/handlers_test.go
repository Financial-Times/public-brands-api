package brands

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/go-logger"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestHandlers(t *testing.T) {
	logger.InitLogger("test-service", "debug")
	var mockClient mockHTTPClient
	router := mux.NewRouter()

	type testCase struct {
		name         string
		url          string
		clientCode   int
		clientBody   string
		clientError  error
		expectedCode int
		expectedBody string
	}
	invalidUUID := testCase{
		"Get Brand - Invalid UUID results in error",
		"/brands/1234",
		200,
		getBasicBrandAsConcept,
		nil,
		400,
		`{"message": "uuid '1234' is either missing or invalid"}`,
	}
	conceptApiError := testCase{
		"Get Brand - Concepts API Error results in error",
		"/brands/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
		503,
		"",
		errors.New("Downstream error"),
		500,
		`{"message": "failed to return brand"}`,
	}
	redirectedUUID := testCase{
		"Get Brand - Given UUID was not canonical",
		"/brands/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
		200,
		getRedirectedBrand,
		nil,
		301,
		``,
	}
	errorOnInvalidJson := testCase{
		"Get Brand - Error on invalid json",
		"/brands/52aa645b-79d6-4f6f-910b-e1cff3f25a15",
		200,
		`{`,
		nil,
		500,
		`{"message": "failed to return brand"}`,
	}
	brandNotFound := testCase{
		"Get Brand - Brand not found",
		"/brands/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
		404,
		"",
		nil,
		404,
		`{"message": "brand not found"}`,
	}
	nonBrandReturnsNotFound := testCase{
		"Get Brand - Other type returns not found",
		"/brands/f92a4ca4-84f9-11e8-8f42-da24cd01f044",
		200,
		getPersonAsConcept,
		nil,
		404,
		`{"message": "brand not found"}`,
	}
	successfulRequest := testCase{
		"Get Brand - Retrieves and transforms correctly",
		"/brands/9636919c-838d-11e8-8f42-da24cd01f044",
		200,
		getCompleteBrandAsConcept,
		nil,
		200,
		transformedCompleteBrand,
	}

	testCases := []testCase{
		invalidUUID,
		conceptApiError,
		redirectedUUID,
		errorOnInvalidJson,
		brandNotFound,
		nonBrandReturnsNotFound,
		successfulRequest,
	}

	for _, test := range testCases {
		mockClient.resp = test.clientBody
		mockClient.statusCode = test.clientCode
		mockClient.err = test.clientError
		bh := NewHandler(&mockClient, "localhost:8080/concepts", "public-brands-api", "public-brands-api")
		bh.RegisterHandlers(router)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", test.url, nil)

		router.ServeHTTP(rr, req)
		assert.Equal(t, test.expectedCode, rr.Code, test.name+" failed: status codes do not match!")
		if rr.Code == 200 {
			assert.Equal(t, transformBody(test.expectedBody), rr.Body.String(), test.name+" failed: status body does not match!")
			continue
		}
		assert.Equal(t, test.expectedBody, rr.Body.String(), test.name+" failed: status body does not match!")

	}
}

func transformBody(testBody string) string {
	stripNewLines := strings.Replace(testBody, "\n", "", -1)
	stripTabs := strings.Replace(stripNewLines, "\t", "", -1)
	return stripTabs + "\n"
}

type mockHTTPClient struct {
	resp       string
	statusCode int
	err        error
}

func (mhc *mockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(mhc.resp)))
	return &http.Response{Body: cb, StatusCode: mhc.statusCode}, mhc.err
}

var getBasicBrandAsConcept = `{
	"id": "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
	"apiUrl": "http://api.ft.com/concepts/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
	"type": "http://www.ft.com/ontology/product/Brand",
	"prefLabel": "Lex"
}`

var getPersonAsConcept = `{
	"id": "http://www.ft.com/thing/f92a4ca4-84f9-11e8-8f42-da24cd01f044",
	"apiUrl": "http://api.ft.com/concepts/f92a4ca4-84f9-11e8-8f42-da24cd01f044",
	"type": "http://www.ft.com/ontology/person/Person",
	"prefLabel": "Not a brand"
}`

var getRedirectedBrand = `{
	"id": "http://www.ft.com/thing/d44db9cd-276d-4035-873f-39a9d8226641",
	"apiUrl": "http://api.ft.com/concepts/d44db9cd-276d-4035-873f-39a9d8226641",
	"type": "http://www.ft.com/ontology/product/Brand",
	"prefLabel": "Redirex"
}`

var getCompleteBrandAsConcept = `{
	"id": "http://www.ft.com/thing/9636919c-838d-11e8-8f42-da24cd01f044",
	"apiUrl": "http://api.ft.com/concepts/9636919c-838d-11e8-8f42-da24cd01f044",
	"prefLabel": "Lex",
	"type": "http://www.ft.com/ontology/product/Brand",
	"imageUrl": "www.imgur.com",
	"descriptionXML": "One brand to rule them all, one brand to find them, one brand to bring them all and in the darkness bind them",
	"strapline": "Something",
	"broaderConcepts": [
		{
			"concept": {
				"id": "http://www.ft.com/thing/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
				"apiUrl": "http://api.ft.com/concepts/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
				"prefLabel": "Old father Lex",
				"type": "http://www.ft.com/ontology/product/Brand"
			}
		}
	],
	"narrowerConcepts": [
		{
			"concept": {
				"id": "http://www.ft.com/thing/0be232ac-841f-11e8-8f42-da24cd01f044",
				"apiUrl": "http://api.ft.com/concepts/0be232ac-841f-11e8-8f42-da24cd01f044",
				"prefLabel": "Little Lex",
				"type": "http://www.ft.com/ontology/product/Brand"
			}
		},
		{
			"concept": {
				"id": "http://www.ft.com/thing/c0eab380-07fe-4672-a277-14ca51ef537e",
				"apiUrl": "http://api.ft.com/concepts/c0eab380-07fe-4672-a277-14ca51ef537e",
				"prefLabel": "Baby Lex",
				"type": "http://www.ft.com/ontology/product/Brand"
			}
		}
	]
}`

var transformedCompleteBrand = `{
	"id":"http://api.ft.com/things/9636919c-838d-11e8-8f42-da24cd01f044",
	"apiUrl":"http://api.ft.com/brands/9636919c-838d-11e8-8f42-da24cd01f044",
	"types":[
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/concept/Concept",
		"http://www.ft.com/ontology/classification/Classification",
		"http://www.ft.com/ontology/product/Brand"
	],
	"directType":"http://www.ft.com/ontology/product/Brand",
	"prefLabel":"Lex",
	"descriptionXML":"One brand to rule them all, one brand to find them, one brand to bring them all and in the darkness bind them",
	"strapline":"Something",
	"_imageUrl":"www.imgur.com",
	"parentBrand":{
		"id":"http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
		"apiUrl":"http://api.ft.com/brands/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
		"types":[
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/classification/Classification",
			"http://www.ft.com/ontology/product/Brand"
		],
		"directType":"http://www.ft.com/ontology/product/Brand",
		"prefLabel":"Old father Lex"
	},
	"childBrands":[
		{
			"id":"http://api.ft.com/things/0be232ac-841f-11e8-8f42-da24cd01f044",
			"apiUrl":"http://api.ft.com/brands/0be232ac-841f-11e8-8f42-da24cd01f044",
			"types":[
				"http://www.ft.com/ontology/core/Thing",
				"http://www.ft.com/ontology/concept/Concept",
				"http://www.ft.com/ontology/classification/Classification",
				"http://www.ft.com/ontology/product/Brand"
			],
			"directType":"http://www.ft.com/ontology/product/Brand",
			"prefLabel":"Little Lex"
		},{
			"id":"http://api.ft.com/things/c0eab380-07fe-4672-a277-14ca51ef537e",
			"apiUrl":"http://api.ft.com/brands/c0eab380-07fe-4672-a277-14ca51ef537e",
			"types":[
				"http://www.ft.com/ontology/core/Thing",
				"http://www.ft.com/ontology/concept/Concept",
				"http://www.ft.com/ontology/classification/Classification",
				"http://www.ft.com/ontology/product/Brand"
			],
			"directType":"http://www.ft.com/ontology/product/Brand",
			"prefLabel":"Baby Lex"
		}
	]
}`
