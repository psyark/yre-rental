
package apihandler

import (
	// "bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/datastore"
	// "cloud.google.com/go/iterator"

	"github.com/psyark/yre-rental/constants"
	"github.com/psyark/yre-rental/models"
)

func init() {
	http.HandleFunc("/api/property/search", searchHandler)
	http.HandleFunc("/api/property/all.geojson", geoJSONHandler)
	http.HandleFunc("/api/property/residence.geojson", geoJSONHandler)
	http.HandleFunc("/api/property/parking.geojson", geoJSONHandler)
	http.HandleFunc("/api/property/business.geojson", geoJSONHandler)
	http.HandleFunc("/api/property/distinct", getDistinctHandler)
	http.HandleFunc("/api/property/", propetyHandler)
}

func getPropertyFromRequest(req *http.Request, prop *models.Property) error {
	if req.Header.Get("Content-Type") != "application/json" {
		return errors.New("Content-Type != application/json")
	}

	//To allocate slice for request body
	length, err := strconv.Atoi(req.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	body := make([]byte, length)
	length, err = req.Body.Read(body)
	if err != nil && err != io.EOF {
		return err
	}

	err = json.Unmarshal(body[:length], prop)
	if err != nil {
		return err
	}

	return nil
}

func searchHandler(rw http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer dsClient.Close()

	params := req.URL.Query()
	q := datastore.NewQuery("Property").Limit(20)
	if val, ok := params["kind"]; ok {
		q = q.Filter("kind =", val[0])
	}
	if val, ok := params["locality"]; ok {
		q = q.Filter("location.locality =", val[0])
	}
	if val, ok := params["inService"]; ok {
		switch val[0] {
		case "true":
			q = q.Filter("management.inService =", true)
		case "false":
			q = q.Filter("management.inService =", false)
		}
	}
	properties := []models.Property{}
	if _, err := dsClient.GetAll(ctx, q, &properties); err != nil {
		log.Fatalf("Err: %v", err)
	}

	jsonBytes, _ := json.MarshalIndent(properties, "", "  ")
	fmt.Fprintln(rw, string(jsonBytes))
}

type geoJSONGeopetry struct {
	Type string `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}
type geoJSONFeature struct {
	Type string `json:"type"`
	Geometry geoJSONGeopetry `json:"geometry"`
	Properties geoJSONProperties `json:"properties"`
}
type geoJSONProperties struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}
type geoJSON struct {
	Type string `json:"type"`
	Features []geoJSONFeature `json:"features"`
}

func geoJSONHandler(rw http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer dsClient.Close()

	properties := []models.Property{}
	q := datastore.NewQuery("Property")
	if _, err := dsClient.GetAll(ctx, q, &properties); err != nil {
		log.Fatalf("Err: %v", err)
	}

	url := req.URL.String()
	var filterProp func(models.Property) bool

	switch {
	case strings.HasSuffix(url, "/all.geojson"):
		filterProp = func (prop models.Property) bool {
			return true
		}
	case strings.HasSuffix(url, "/residence.geojson"):
		filterProp = func (prop models.Property) bool {
			return prop.GetCategory() == models.CategoryResidence
		}
	case strings.HasSuffix(url, "/parking.geojson"):
		filterProp = func (prop models.Property) bool {
			return prop.GetCategory() == models.CategoryParking
		}
	case strings.HasSuffix(url, "/business.geojson"):
		filterProp = func (prop models.Property) bool {
			return prop.GetCategory() == models.CategoryBusiness
		}
	}

	gj := geoJSON{Type:"FeatureCollection"}

	for _, prop := range(properties) {
		if filterProp(prop) {
			gj.Features = append(gj.Features, geoJSONFeature{
				Type:"Feature",
				Geometry:geoJSONGeopetry{
					Type:"Point",
					Coordinates: []float64{
						prop.Location.GeoCoord.Lng,
						prop.Location.GeoCoord.Lat,
					},
				},
				Properties:geoJSONProperties{
					Name:prop.Name.Ja,
					Kind:prop.Kind,
				},
			})
		}
	}

	rw.Header().Set("Content-Type", "application/geo+json; charset=UTF-8")
	jsonBytes, _ := json.MarshalIndent(gj, "", "  ")
	fmt.Fprintln(rw, string(jsonBytes))
}

type distinctResp struct {
	Kind []string `json:"kind"`
	Locality []string `json:"locality"`
}

func getDistinctHandler(rw http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer dsClient.Close()

	resp := distinctResp{}

	q := datastore.NewQuery("Property").Project("kind").DistinctOn("kind")
	props := []models.Property{}
	if _, err := dsClient.GetAll(ctx, q, &props); err != nil {
		log.Fatalf("Err: %v", err)
	}
	for _, prop := range(props) {
		resp.Kind = append(resp.Kind, prop.Kind)
	}

	q = datastore.NewQuery("Property").Project("location.locality").DistinctOn("location.locality")
	props = []models.Property{}
	if _, err := dsClient.GetAll(ctx, q, &props); err != nil {
		log.Fatalf("Err: %v", err)
	}
	for _, prop := range(props) {
		resp.Locality = append(resp.Locality, prop.Location.Locality)
	}

	jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintln(rw, string(jsonBytes))
}

func propetyHandler(rw http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	nameOrID := req.URL.String()[len("/api/property/"):]
	propKey := datastore.NameKey("Property", nameOrID, nil)

	dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer dsClient.Close()

	property := models.Property{}
	if err := dsClient.Get(ctx, propKey, &property); err != nil {
		if err == datastore.ErrNoSuchEntity {
			http.NotFound(rw, req)
			return
		}
		log.Fatalf("Err: %v", err)
	}

	if req.Method == http.MethodPut {
		if err := getPropertyFromRequest(req, &property); err != nil {
			log.Fatalf("Err: %v", err)
		}
		if _, err := dsClient.Put(ctx, propKey, &property); err != nil {
			log.Fatalf("Err: %v", err)
		}
	}

	rooms := []models.Room{}
	query := datastore.NewQuery("Room").Ancestor(propKey)
	if _, err := dsClient.GetAll(ctx, query, &rooms); err != nil {
		log.Fatalf("Err: %v", err)
	}

	propWithRooms := models.PropertyWithRooms{}
	propWithRooms.Property = property
	propWithRooms.Rooms = rooms

	jsonBytes, _ := json.MarshalIndent(propWithRooms, "", "  ")
	fmt.Fprintln(rw, string(jsonBytes))
}
