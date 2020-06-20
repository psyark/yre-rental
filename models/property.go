
package models

import (
	"time"
	"cloud.google.com/go/datastore"
)

// ExposeKey はフロント側にキー情報を公開するモデルです
type ExposeKey struct {
	NameOrID string `datastore:"-" json:"nameOrId"`
}

// LoadKey https://stackoverflow.com/questions/57916528/how-to-fetch-name-key-in-from-datastore
func (ek *ExposeKey) LoadKey(k *datastore.Key) error {
	if k.Name != "" {
		ek.NameOrID = k.Name
	} else {
		ek.NameOrID = string(k.ID)
	}
    return nil
}

// Name は物件の名前です
type Name struct {
	Ja string `datastore:"ja" json:"ja"`
	JaKata string `datastore:"ja_kata" json:"ja_kata"`
}

// GeoCoord (Geographic Coordinate)は緯度経度です
type GeoCoord struct {
    Lat float64 `datastore:"lat" json:"lat"`
	Lng float64 `datastore:"lng" json:"lng"`
}

// Location は物件の所在地です
type Location struct {
	Address string `datastore:"address" json:"address"`

	// 以降は住所からジオコーディングなどの手法で導出した派生情報
	PostalCode string `datastore:"postalCode" json:"postalCode"`
	GeoCoord GeoCoord `datastore:"geoCoord" json:"geoCoord"`
	Locality string `datastore:"locality" json:"locality"`
}

// Management は物件の賃貸管理に関する情報です
type Management struct {
	StartDate *time.Time `datastore:"startDate" json:"startDate"`
	EndDate *time.Time `datastore:"endDate" json:"endDate"`
	InService bool `datastore:"inService" json:"inService"`
}

// Property ば単一の賃貸物件を表すDatastoreモデルです
type Property struct {
	ExposeKey
	Name Name `datastore:"name,flatten" json:"name"`
	Location Location `datastore:"location,flatten" json:"location"`
	Kind string `datastore:"kind" json:"kind"`
	Management Management `datastore:"management,flatten" json:"management"`
}

// Load https://stackoverflow.com/questions/57916528/how-to-fetch-name-key-in-from-datastore
func (prop *Property) Load(ps []datastore.Property) error {
    return datastore.LoadStruct(prop, ps)
}

// Save https://stackoverflow.com/questions/57916528/how-to-fetch-name-key-in-from-datastore
func (prop *Property) Save() ([]datastore.Property, error) {
    return datastore.SaveStruct(prop)
}

// KindCategory は物件種別の分類です
type KindCategory struct { value string }

// CategoryResidence は居住用です
var CategoryResidence = KindCategory{"residence"}
// CategoryParking は駐車場です
var CategoryParking = KindCategory{"parking"}
// CategoryBusiness は事業用です
var CategoryBusiness = KindCategory{"business"}
// CategoryOther はその他です
var CategoryOther = KindCategory{"other"}

// GetCategory https://stackoverflow.com/questions/57916528/how-to-fetch-name-key-in-from-datastore
func (prop *Property) GetCategory() KindCategory {
	switch prop.Kind {
	case "一戸建て": fallthrough
	case "アパート": fallthrough
	case "マンション": fallthrough
	case "共同住宅": fallthrough
	case "テラスハウス":
		return CategoryResidence
	case "駐車場": fallthrough
	case "駐輪場":
		return CategoryParking
	case "店舗": fallthrough
	case "住宅付店舗": fallthrough
	case "事務所": fallthrough
	case "倉庫": fallthrough
	case "ビル": fallthrough
	case "貸地":
		return CategoryBusiness
	default:
		return CategoryOther
	}
}

// PropertyWithRooms は 部屋のリストを持つAPIレスポンス専用の構造体です
type PropertyWithRooms struct {
	Property
	Rooms []Room `datastore:"-" json:"rooms"`
}

// TimePeriod は期間を表します
type TimePeriod struct {
	From string `datastore:"from" json:"from"`
	To string `datastore:"to" json:"to"`
}

// Tenant は Contractにおける賃借人です
type Tenant struct {
	ID string `datastore:"id" json:"id"`
	Name string `datastore:"name" json:"name"`
}

// Contract は 単一のRoomに対する賃貸契約を表すDatastoreモデルです
type Contract struct {
	Period TimePeriod `datastore:"period,flatten" json:"period"`
	Tenant Tenant `datastore:"tenant,flatten" json:"tenant"`
}

// Rentable はRoomが賃貸可能かどうかを表すDatastoreモデルです
type Rentable struct {
	Rentable bool `datastore:"rentable" json:"rentable"`
	Reason string `datastore:"reason" json:"reason"`
}

// Room は Propertyの部屋または駐車区画を表すDatastoreモデルです
type Room struct {
	ExposeKey
	Layout string `datastore:"layout,noindex" json:"layout"`
	Contract *Contract `datastore:"contract,flatten" json:"contract"`
	Rentable Rentable `datastore:"rentable,flatten" json:"rentable"`
}

// Load https://stackoverflow.com/questions/57916528/how-to-fetch-name-key-in-from-datastore
func (ek *Room) Load(ps []datastore.Property) error {
    return datastore.LoadStruct(ek, ps)
}

// Save https://stackoverflow.com/questions/57916528/how-to-fetch-name-key-in-from-datastore
func (ek *Room) Save() ([]datastore.Property, error) {
    return datastore.SaveStruct(ek)
}
