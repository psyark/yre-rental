
package apihandler

import (
	// "bytes"
	"context"
	"encoding/csv"
	// "encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unicode"

	"cloud.google.com/go/datastore"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/psyark/yre-rental/constants"
	"github.com/psyark/yre-rental/models"
)

func init() {
	http.HandleFunc("/api/import/ck-properties", ckPropertiesHandler)
	http.HandleFunc("/api/import/ck-property-managements", ckPropertyManagementsHandler)
	http.HandleFunc("/api/import/ck-rooms", ckRoomsHandler)
}

func parseCSV(ctx context.Context, out chan<- map[string]string, file io.ReadCloser) {
	defer file.Close()
	defer close(out)

	reader := csv.NewReader(transform.NewReader(file, japanese.ShiftJIS.NewDecoder()))
	var record []string
	count := 0

	for header, err := reader.Read(); err != io.EOF; record, err = reader.Read() {
		if err != nil {
			panic(err)
		}
		if record != nil {
			out <- zip(header, record)
			count++
		}
	}

	log.Println(count)
}

func ckPropertiesHandler(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	file, _, err := req.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	ctx := context.Background()
	chRecord := make(chan map[string]string)
	go parseCSV(ctx, chRecord, file)

	dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer dsClient.Close()

	keys := []*datastore.Key{}
	entities := []models.Property{}

	wg := sync.WaitGroup{}
	putBatch := func(keys []*datastore.Key, entities []models.Property) {
		if _, err := dsClient.PutMulti(ctx, keys, entities); err != nil {
			log.Printf("Error: %v\n", err)
		} else {
			log.Printf("%v OK\n", len(keys))
		}
		wg.Done()
	}

	concatAddress := func (a string, b string) string {
		ra := []rune(a)
		rb := []rune(b)
		if a != "" && b != "" && unicode.IsNumber(ra[len(ra)-1]) && unicode.IsNumber(rb[0]) {
			return a + "-" + b
		}
		return a + b
	}

	for {
		recMap, more := <-chRecord
		if more {
			propKey := datastore.NameKey("Property", fmt.Sprintf("ck-%v", recMap["物件No"]), nil)
			property := models.Property{
				Name: models.Name{Ja: recMap["物件名"], JaKata: recMap["物件名カナ"]},
				Location: models.Location{
					PostalCode: recMap["郵便番号"],
					Address: concatAddress(recMap["都道府県名"] + recMap["市区町村名"] + recMap["町地域"] + recMap["丁目など"], recMap["番地"]),
				},
				Kind: recMap["物件分類"],
			}

			keys = append(keys, propKey)
			entities = append(entities, property)

			if len(keys) == 200 {
				wg.Add(1)
				go putBatch(keys, entities)

				keys = []*datastore.Key{}
				entities = []models.Property{}
			}
		} else {
			if len(keys) > 0 {
				wg.Add(1)
				go putBatch(keys, entities)
			}
			break
		}
	}

	wg.Wait()
	log.Println("All OK")
}

func ckPropertyManagementsHandler(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	file, _, err := req.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	ctx := context.Background()
	chRecord := make(chan map[string]string)
	go parseCSV(ctx, chRecord, file)

	dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer dsClient.Close()

	wg := sync.WaitGroup{}
	addManagementData := func (propKey *datastore.Key, management models.Management) {
		_, err := dsClient.RunInTransaction(ctx, func (tx *datastore.Transaction) error {
			prop := models.Property{}
			if err := tx.Get(propKey, &prop); err != nil {
				return err
			}
			prop.Management = management
			if _, err := tx.Put(propKey, &prop); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("Errpr: %v\n", err)
		}
		wg.Done()
	}

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	now := time.Now()

	for {
		recMap, more := <-chRecord
		if more {
			propKey := datastore.NameKey("Property", fmt.Sprintf("ck-%v", recMap["物件No"]), nil)
			man := models.Management{}
			if len(recMap["業務対象開始"]) == 7 {
				y, _ := strconv.Atoi(recMap["業務対象開始"][0:4])
				m, _ := strconv.Atoi(recMap["業務対象開始"][5:7])
				t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, loc)
				man.StartDate = &t
			}
			if len(recMap["業務対象終了"]) == 7 {
				y, _ := strconv.Atoi(recMap["業務対象終了"][0:4])
				m, _ := strconv.Atoi(recMap["業務対象終了"][5:7])
				t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, loc).AddDate(0, 1, 0).Add(-time.Second)
				man.EndDate = &t
			}
			started := man.StartDate != nil && man.StartDate.Before(now)
			ended := man.EndDate != nil && man.EndDate.Before(now)
			man.InService = started && !ended

			wg.Add(1)
			go addManagementData(propKey, man)
		} else {
			break
		}
	}

	wg.Wait()
	log.Println("All OK")
}

func ckRoomsHandler(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	file, _, err := req.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	ctx := context.Background()
	chRecord := make(chan map[string]string)
	go parseCSV(ctx, chRecord, file)

    dsClient, err := datastore.NewClient(ctx, constants.ProjectID)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
	defer dsClient.Close()

	keys := []*datastore.Key{}
	entities := []models.Room{}

	for len(keys) < 200 {
		recMap, more := <-chRecord
		if more {
			if recMap["部屋No"] != "" {  // 合計行は無視
				propertyKey := datastore.NameKey("Property", fmt.Sprintf("ck-%v", recMap["物件No"]), nil)
				roomKey := datastore.NameKey("Room", recMap["部屋No"], propertyKey)

				room := models.Room{Layout: recMap["間取り"]}

				switch recMap["契約状況"] {
				case "契約中":
					fallthrough
				case "解約予定":
					fallthrough
				case "契約終了":
					room.Contract = &models.Contract{}
					room.Contract.Period.From = recMap["契約始期"]
					room.Contract.Period.To = recMap["契約始期"]
					room.Contract.Tenant.Name = recMap["契約者名(SJIS)"]
					room.Contract.Tenant.ID = fmt.Sprintf("ck-tenant-%v", recMap["契約者No"])
				case "空　室":
					room.Rentable.Rentable = true
				case "契約中(他社)":
					room.Rentable.Reason = "契約中(他社)"
				default:
					fmt.Println(recMap["契約状況"])
				}

				keys = append(keys, roomKey)
				entities = append(entities, room)
			}
		} else {
			break
		}
	}

	log.Println("Start")
	if _, err := dsClient.PutMulti(ctx, keys, entities); err != nil {
		fmt.Fprintf(rw, "Error: %v\n", err)
	} else {
		log.Println("Done")
	}
}

func zip(keys []string, values []string) map[string]string {
    zipped := map[string]string{}
    for i, k := range keys {
        zipped[k] = values[i]
    }
    return zipped
}
