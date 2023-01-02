# Public API for Brands (public-brands-api)
[![Circle CI](https://circleci.com/gh/Financial-Times/public-brands-api.svg?style=shield)](https://circleci.com/gh/Financial-Times/public-brands-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/public-brands-api)](https://goreportcard.com/report/github.com/Financial-Times/public-brands-api)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/public-brands-api/badge.svg)](https://coveralls.io/github/Financial-Times/public-brands-api)

## Migration

Brands are being migrated to be served from the new [Public Concepts API](https://github.com/Financial-Times/public-concepts-api) and as such this API will eventually be deprecated. From July 2018 requests to this service will be redirected via the concepts api then transformed to match the existing contract and returned.

Provides a public API for Brands data

## Build & deployment etc:

_NB You will need to tag a commit in order to build
* [Build & Deploy](https://upp-k8s-jenkins.in.ft.com/job/k8s-deployment/job/apps-deployment/job/public-brands-api-auto-deploy/)

## Runbook
Information about the service can be found in the (public-brands-api runbook)[https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/public-brands-api]

## Installation & running locally
* You will need to load some data, see [Brands RW API](https://github.com/Financial-Times/brands-rw-neo4j) for help with that
* `go get -u github.com/Financial-Times/public-brands-api`
* `cd $GOPATH/src/github.com/Financial-Times/public-brands-api`
* `go test ./...`
* `go install`
* `$GOPATH/bin/public-brands-api --neo-url={neo4jUrl} --port={port} --log-level={DEBUG|INFO|WARN|ERROR}`
_Both arguments are optional.
--neo-url defaults to http://localhost:7474/db/data, which is the out of box url for a local neo4j instance.
--port defaults to 8080._
* `curl http://localhost:8080/brands/2d3e16e0-61cb-4322-8aff-3b01c59f4daa | json_pp`


## API definition
* The API only supports HTTP GET requests and only takes one parameter, uuid:
  `http://api.ft.com/brands/{uuid}`
* The an example result structure is shown below, _note that when there is no parent or child brand then we omit the those attribute_:

```
{
  "id": "http://api.ft.com/things/{uuid}",
  "apiUrl": "http://api.ft.com/brands/{uuid}",
  "types": [
    "http://www.ft.com/ontology/core/Thing",
    "http://www.ft.com/ontology/concept/Concept",
    "http://www.ft.com/ontology/classification/Classification",
    "http://www.ft.com/ontology/product/Brand"
  ],
  "directType": "http://www.ft.com/ontology/product/Brand",
  "prefLabel": "Brand Name",
  "description": "A description of the brand, in plain text",
  "descriptionXML": "<body><p>A description of the brand, in <i>bodyXML</i></p></body>",
  "strapline": "A subsidiary heading, caption or advertising slogan",
  "_imageUrl": "http://images.ft.com/tempImageWhilstThisIsResolved.jpg",
  "childBrands": [],
  "parentBrands"" []
}
```

* Brands can have parents, as illustrated below when considering Lex Live:

```
{
  "id": "http://api.ft.com/things/e363dfb8-f6d9-4f2c-beba-5162b334272b",
  "apiUrl": "http://api.ft.com/brands/e363dfb8-f6d9-4f2c-beba-5162b334272b",
  "types": [
    "http://www.ft.com/ontology/core/Thing",
    "http://www.ft.com/ontology/concept/Concept",
    "http://www.ft.com/ontology/classification/Classification",
    "http://www.ft.com/ontology/product/Brand"
  ],
  "directType": "http://www.ft.com/ontology/product/Brand",
  "prefLabel": "Lex Live",
  "description": "",
  "descriptionXML": "",
  "strapline": "",
  "_imageUrl": "",
  "parentBrands": [ 
    {
        "id": "http://api.ft.com/things/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
        "apiUrl": "http://api.ft.com/brands/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
        "types": [
          "http://www.ft.com/ontology/product/Brand"
        ],
        "prefLabel": "Lex"
    }
  ],  
  "childBrands": []
}
```

* As well a parents, brands can have children as shown below:

```
{
  "id": "http://api.ft.com/things/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
  "apiUrl": "http://api.ft.com/brands/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
  "types": [
    "http://www.ft.com/ontology/core/Thing",
    "http://www.ft.com/ontology/concept/Concept",
    "http://www.ft.com/ontology/classification/Classification",
    "http://www.ft.com/ontology/product/Brand"
  ],
  "directType": "http://www.ft.com/ontology/product/Brand",
  "prefLabel": "Lex",
  "description": "",
  "descriptionXML": "",
  "strapline": "",
  "_imageUrl": "",
  "parentBrands": [
     {
        "id": "http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
        "apiUrl": "http://api.ft.com/brands/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54",
        "types": [
            "http://www.ft.com/ontology/core/Thing",
            "http://www.ft.com/ontology/concept/Concept",
            "http://www.ft.com/ontology/classification/Classification",
            "http://www.ft.com/ontology/product/Brand"
        ],
        "directType": "http://www.ft.com/ontology/product/Brand",
        "prefLabel": "Financial Times"
     }
  ],   
  "childBrands": [
    {
      "id": "http://api.ft.com/things/e363dfb8-f6d9-4f2c-beba-5162b334272b",
      "apiUrl": "http://api.ft.com/brands/e363dfb8-f6d9-4f2c-beba-5162b334272b",
      "types": [
         "http://www.ft.com/ontology/core/Thing",
         "http://www.ft.com/ontology/concept/Concept",
         "http://www.ft.com/ontology/classification/Classification",
         "http://www.ft.com/ontology/product/Brand"
      ],
      "directType": "http://www.ft.com/ontology/product/Brand",
      "prefLabel": "Lex Live"
    }
  ]
}
```

## Endpoints

* Based on the following [google doc](https://docs.google.com/document/d/1SC4Uskl-VD78y0lg5H2Gq56VCmM4OFHofZM-OvpsOFo/edit#heading=h.qjo76xuvpj83).
* See the [api](_ft/api.yml) Swagger file for endpoints definitions.  
test
