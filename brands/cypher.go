package brands

import (
	"errors"
	"fmt"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

// Driver interface
type Driver interface {
	Read(id string) (brand Brand, found bool, err error)
	CheckConnectivity() error
}

// CypherDriver struct
type CypherDriver struct {
	db  *neoism.Database
	env string
}

//NewCypherDriver instantiate driver
func NewCypherDriver(db *neoism.Database, env string) CypherDriver {
	return CypherDriver{db, env}
}

// CheckConnectivity tests neo4j by running a simple cypher query
func (driver CypherDriver) CheckConnectivity() error {
	results := []struct {
		ID int
	}{}
	query := &neoism.CypherQuery{
		Statement: "MATCH (x) RETURN ID(x) LIMIT 1",
		Result:    &results,
	}
	err := driver.db.Cypher(query)
	log.Debugf("CheckConnectivity results:%+v  err: %+v", results, err)
	return err
}

func (driver CypherDriver) Read(uuid string) (brand Brand, found bool, err error) {
	results := []struct {
		Brand
	}{}

	query := &neoism.CypherQuery{
		Statement: `
                        MATCH (upp:UPPIdentifier{value:{uuid}})-[:IDENTIFIES]->(b:Brand)
                        OPTIONAL MATCH (b)-[:HAS_PARENT]->(p:Thing)
                        OPTIONAL MATCH (b)<-[:HAS_PARENT]-(c:Thing)
                        RETURN  b.uuid as id, labels(b) as types, b.prefLabel as prefLabel,
                                b.description as description, b.descriptionXML as descriptionXML,
                                b.strapline as strapline, b.imageUrl as _imageUrl,
                                { id: p.uuid, types: labels(p), prefLabel: p.prefLabel } AS parentBrand,
                                collect (
                                { id: c.uuid, types: labels(c), prefLabel: c.prefLabel }
                                ) AS childBrands
                `,
		Parameters: neoism.Props{"uuid": uuid},
		Result:     &results,
	}
	err = driver.db.Cypher(query)
	if err != nil {
		log.Errorf("Error looking up uuid %s with query %s from neoism: %+v\n", uuid, query.Statement, err)
		return Brand{}, false, fmt.Errorf("Error accessing Brands datastore for uuid: %s", uuid)
	}
	log.Debugf("CypherResult Read Brand for uuid: %s was: %+v", uuid, results)
	if (len(results)) == 0 {
		return Brand{}, false, nil
	} else if len(results) != 1 {
		errMsg := fmt.Sprintf("Multiple brands found with the same uuid:%s !", uuid)
		log.Error(errMsg)
		return Brand{}, true, errors.New(errMsg)
	}
	publicAPITransformation(&results[0].Brand, driver.env)
	log.Debugf("Returning %v", results[0].Brand)
	return results[0].Brand, true, nil
}

func publicAPITransformation(brand *Brand, env string) {
	if brand.Parent.ID != "" {
		brand.Parent.APIURL = mapper.APIURL(brand.Parent.ID, brand.Parent.Types, env)
		brand.Parent.ID = mapper.IDURL(brand.Parent.ID)
		brand.Parent.Types = mapper.TypeURIs(brand.Parent.Types)
	} else {
		brand.Parent = nil
	}
	if brand.Children[0].ID != "" {
		for idx := range brand.Children {
			brand.Children[idx].APIURL = mapper.APIURL(brand.Children[idx].ID, brand.Children[idx].Types, env)
			brand.Children[idx].ID = mapper.IDURL(brand.Children[idx].ID)
			brand.Children[idx].Types = mapper.TypeURIs(brand.Children[idx].Types)
		}
	} else {
		brand.Children = []*Thing{}
	}
	brand.APIURL = mapper.APIURL(brand.ID, brand.Types, env)
	brand.Types = mapper.TypeURIs(brand.Types)
	brand.ID = mapper.IDURL(brand.ID)
}
