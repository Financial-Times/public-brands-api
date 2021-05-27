package brands

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"fmt"
	"io/ioutil"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	logger "github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/service-status-go/gtg"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// CacheControlHeader is the value to set on http header
var CacheControlHeader string

type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

const (
	validUUID     = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"
	thingsApiUrl  = "http://api.ft.com/things/"
	ftThing       = "http://www.ft.com/thing/"
	brandOntology = "http://www.ft.com/ontology/product/Brand"
	queryParams   = "?showRelationship=broader&showRelationship=narrower"
)

type BrandsHandler struct {
	client      httpClient
	conceptsURL string
}

func NewHandler(client httpClient, conceptsURL string) BrandsHandler {
	return BrandsHandler{
		client:      client,
		conceptsURL: conceptsURL,
	}
}

func (h *BrandsHandler) Checker() (string, error) {
	req, err := http.NewRequest("GET", h.conceptsURL+"/__gtg", nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("User-Agent", "UPP public-brands-api")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("health check returned a non-200 HTTP status: %v", resp.StatusCode)
	}
	return "Public Concepts API is healthy", nil

}

func (h *BrandsHandler) HealthCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "public-concepts-api-check",
		BusinessImpact:   "Unable to respond to Public Brands api requests",
		Name:             "Check connectivity to public-concepts-api",
		PanicGuide:       "https://runbooks.in.ft.com/public-brands-api",
		Severity:         2,
		TechnicalSummary: "Not being able to communicate with public-concepts-api means that requests for organisations cannot be performed.",
		Checker:          h.Checker,
	}
}

// MethodNotAllowedHandler does stuff
func (h *BrandsHandler) MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}

func (h *BrandsHandler) RegisterHandlers(router *mux.Router) {
	logger.Info("Registering handlers")
	mh := handlers.MethodHandler{
		"GET": http.HandlerFunc(h.GetBrand),
	}

	// These paths need to actually be the concept type
	router.Handle("/brands/{uuid}", mh)
}

// GetBrand is the public API
func (h *BrandsHandler) GetBrand(w http.ResponseWriter, r *http.Request) {
	uuidMatcher := regexp.MustCompile(validUUID)
	vars := mux.Vars(r)
	UUID := vars["uuid"]
	transID := transactionidutils.GetTransactionIDFromRequest(r)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", CacheControlHeader)

	if UUID == "" || !uuidMatcher.MatchString(UUID) {
		msg := fmt.Sprintf(`uuid '%s' is either missing or invalid`, UUID)
		logger.WithTransactionID(transID).WithUUID(UUID).Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "` + msg + `"}`))
		return
	}

	brand, canonicalUUID, found, err := h.getBrandViaConceptsAPI(UUID, transID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "failed to return brand"}`))
		return
	}

	if found && canonicalUUID != "" && canonicalUUID != UUID {
		redirectURL := strings.Replace(r.RequestURI, UUID, canonicalUUID, 1)
		logger.WithTransactionID(transID).WithUUID(UUID).Debug("serving redirect")
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}
	if !found {
		msg := fmt.Sprint("brand not found")
		w.WriteHeader(http.StatusNotFound)
		logger.WithTransactionID(transID).WithUUID(UUID).Info(msg)
		w.Write([]byte(`{"message": "` + msg + `"}`))
		return
	}

	logger.Debugf("Brand (uuid): %s\n", brand.ID)

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(brand)
	if err != nil {
		msg := fmt.Sprintf("brand: %v could not be marshaled", brand)
		logger.WithError(err).WithTransactionID(transID).WithUUID(UUID).Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + msg + `"}`))
	}
}

//GoodToGo returns a 503 if the healthcheck fails - suitable for use from varnish to check availability of a node
func (h *BrandsHandler) GTG() gtg.Status {
	statusCheck := func() gtg.Status {
		return gtgCheck(h.Checker)
	}
	return gtg.FailFastParallelCheck([]gtg.StatusChecker{statusCheck})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *BrandsHandler) getBrandViaConceptsAPI(UUID string, transID string) (brand Brand, canonicalUuid string, found bool, err error) {
	logger.WithTransactionID(transID).WithUUID(UUID).Debug("retrieving brand via concepts api")
	mappedBrand := Brand{}
	reqURL := h.conceptsURL + "/concepts/" + UUID + queryParams
	request, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		msg := fmt.Sprintf("failed to create request to %s", reqURL)
		logger.WithError(err).WithUUID(UUID).WithTransactionID(transID).Error(msg)
		return mappedBrand, "", false, err
	}

	request.Header.Set("X-Request-Id", transID)
	resp, err := h.client.Do(request)
	if err != nil {
		msg := fmt.Sprintf("request to %s returned status: %d", reqURL, resp.StatusCode)
		logger.WithError(err).WithUUID(UUID).WithTransactionID(transID).Error(msg)
		return mappedBrand, "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return mappedBrand, "", false, nil
	}

	conceptsApiResponse := ConceptApiResponse{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read response body: %v", resp.Body)
		logger.WithError(err).WithUUID(UUID).WithTransactionID(transID).Error(msg)
		return mappedBrand, "", false, err
	}
	if err = json.Unmarshal(body, &conceptsApiResponse); err != nil {
		msg := fmt.Sprintf("failed to unmarshal response body: %v", body)
		logger.WithError(err).WithUUID(UUID).WithTransactionID(transID).Error(msg)
		return mappedBrand, "", false, err
	}

	if conceptsApiResponse.Type != brandOntology {
		logger.WithTransactionID(transID).WithUUID(UUID).Debug("requested concept is not a brand")
		return mappedBrand, "", false, nil
	}

	mappedBrand.ID = convertID(conceptsApiResponse.ID)
	mappedBrand.APIURL = convertApiUrl(conceptsApiResponse.ApiURL)
	mappedBrand.PrefLabel = conceptsApiResponse.PrefLabel
	mappedBrand.IsDeprecated = conceptsApiResponse.IsDeprecated
	mappedBrand.Types = mapper.FullTypeHierarchy(conceptsApiResponse.Type)
	mappedBrand.DirectType = conceptsApiResponse.Type
	mappedBrand.ImageURL = conceptsApiResponse.ImageURL
	mappedBrand.DescriptionXML = conceptsApiResponse.DescriptionXML
	mappedBrand.Strapline = conceptsApiResponse.Strapline

	for _, broader := range conceptsApiResponse.Broader {
		if broader.Concept.Type == brandOntology {
			mappedBrand.Parent = convertRelationship(broader)
			break
		}
	}
	var children []Thing
	for _, narrower := range conceptsApiResponse.Narrower {
		children = append(children, *convertRelationship(narrower))
	}
	mappedBrand.Children = children
	return mappedBrand, strings.TrimPrefix(mappedBrand.ID, thingsApiUrl), true, nil
}

func convertRelationship(rc RelatedConcept) *Thing {
	return &Thing{
		ID:         convertID(rc.Concept.ID),
		APIURL:     convertApiUrl(rc.Concept.ApiURL),
		Types:      mapper.FullTypeHierarchy(rc.Concept.Type),
		DirectType: rc.Concept.Type,
		PrefLabel:  rc.Concept.PrefLabel,
	}
}

func convertApiUrl(conceptsApiUrl string) string {
	return strings.Replace(conceptsApiUrl, "concepts", "brands", 1)
}

func convertID(conceptsApiID string) string {
	return strings.Replace(conceptsApiID, ftThing, thingsApiUrl, 1)
}
