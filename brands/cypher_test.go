package brands

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/brands-rw-neo4j/brands"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

var validSimpleBrand = brands.Brand{
	UUID:           "0c63a9bf-6fc4-49d0-809b-7bc3dc8b8ec9",
	PrefLabel:      "validSimpleBrand",
	Strapline:      "Keeping it simple",
	Description:    "This brand has no parent but otherwise has valid values for all fields",
	DescriptionXML: "<body>This <i>brand</i> has no parent but otherwise has valid values for all fields</body>",
	ImageURL:       "http://media.ft.com/validSimpleBrand.png",
}

var parentBrand = brands.Brand{
	UUID:           "d851e146-e889-43f3-8f4c-269da9bb0298",
	PrefLabel:      "parentBrand",
	Strapline:      "Keeping it in the family",
	Description:    "This brand has is a parent",
	DescriptionXML: "<body>This brand has is a parent</body>",
	ImageURL:       "http://media.ft.com/parentBrand.png",
}

var childBrand = brands.Brand{
	UUID:           "a806e270-edbc-423f-b8db-d21ae90e06c8",
	ParentUUID:     "d851e146-e889-43f3-8f4c-269da9bb0298",
	PrefLabel:      "childBrand1",
	Strapline:      "I live in one family",
	Description:    "This brand has a parent and valid values for all fields",
	DescriptionXML: "<body>This <i>brand</i> has a parent and valid values for all fields</body>",
	ImageURL:       "http://media.ft.com/childBrand1.png",
}

func TestSimpleBrand(t *testing.T) {
	assert := assert.New(t)
	brandsWriter := getBrandRWDriver(t)
	writeJSONToService(brandsWriter, "./fixtures/ValidSimpleBrand-0c63a9bf-6fc4-49d0-809b-7bc3dc8b8ec9.json", assert)
	readAndCompare(&validSimpleBrand, nil, nil, t)
	cleanUp(validSimpleBrand.UUID, t)
}

func TestSimpleBrandAsParent(t *testing.T) {
	assert := assert.New(t)

	brandsWriter := getBrandRWDriver(t)
	writeJSONToService(brandsWriter, "./fixtures/ParentBrand-d851e146-e889-43f3-8f4c-269da9bb0298.json", assert)
	writeJSONToService(brandsWriter, "./fixtures/ChildBrand-a806e270-edbc-423f-b8db-d21ae90e06c8.json", assert)

	readAndCompare(&childBrand, &parentBrand, nil, t)
	cleanUp(childBrand.UUID, t)
	cleanUp(parentBrand.UUID, t)
}

func TestConnectivityCheck(t *testing.T) {
	driver := getBrandRWDriver(t)
	err := driver.Check()
	assert.NoError(t, err)
}

func readAndCompare(source *brands.Brand, parent *brands.Brand, children []*brands.Brand, t *testing.T) {
	brand, found, err := getBrandDriver(t).Read(source.UUID)
	expected := makeBrand(source, parent, children, t)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.NotEmpty(t, brand)
	fmt.Printf("**Made %v+\n\n**Found %v+\n", expected, brand)
	if brand.Parent != nil {
		fmt.Printf("\n**Made.Parent %v+\n\n**Found.Parent %v+\n", *expected.Parent, *brand.Parent)
	}
	for _, child := range brand.Children {
		fmt.Printf("brand.child %v+\n", *child)
	}
	assert.EqualValues(t, expected, brand)
}

func makeBrand(source *brands.Brand, parent *brands.Brand, children []*brands.Brand, t *testing.T) (brand Brand) {
	brand.Thing = makeThing(source, t)
	brand.PrefLabel = source.PrefLabel
	brand.Strapline = source.Strapline
	brand.Description = source.Description
	brand.DescriptionXML = source.DescriptionXML
	brand.ImageURL = source.ImageURL
	if parent == nil {
		brand.Parent = nil
	} else {
		parentBrand := makeThing(parent, t)
		brand.Parent = &parentBrand
	}
	childrenBrands := make([]*Thing, len(children))
	for idx := range children {
		child := makeThing(children[idx], t)
		childrenBrands[idx] = &child
	}
	brand.Children = childrenBrands
	return brand
}

func makeThing(source *brands.Brand, t *testing.T) Thing {
	thing := Thing{}
	thing.ID = "http://api.ft.com/things/" + source.UUID
	thing.APIURL = "http://test.api.ft.com/brands/" + source.UUID
	thing.Types = []string{"http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/classification/Classification", "http://www.ft.com/ontology/product/Brand"}
	thing.PrefLabel = source.PrefLabel
	return thing
}

func getBrandRWDriver(t *testing.T) (service baseftrwapp.Service) {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	db, err := neoism.Connect(url)
	assert.NoError(t, err, "Error setting up connection to %s", url)
	return brands.NewCypherBrandsService(neoutils.StringerDb{db}, db)
}

func getBrandDriver(t *testing.T) CypherDriver {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	db, err := neoism.Connect(url)
	assert.NoError(t, err, "Error setting up connection to %s", url)
	return NewCypherDriver(db, "test")
}

func cleanUp(uuid string, t *testing.T) {
	found, err := getBrandRWDriver(t).Delete(uuid)
	assert.True(t, found, "Unable to delete brand with uuid %s", uuid)
	assert.NoError(t, err, "Error deleting brand with uuid %s", uuid)
}


func writeJSONToService(service baseftrwapp.Service, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst)
	assert.NoError(errrr)
}