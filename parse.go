package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const userAgent = "Mozilla/5.0 (Linux; Android 9; SAMSUNG GT-I9505 Build/LRX22C) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.93 Mobile Safari/537.36"

const (
	parseLinkAlio        = "https://www.alio.lt/paieska/?category_id=1373&city_id=228626&search_block=1&search[eq][adresas_1]=228626&order=ad_id"
	parseLinkAruodas     = "https://m.aruodas.lt/?obj=1&FRegion=461&FDistrict=1&FOrder=AddDate&from_search=1&detailed_search=1&FShowOnly=FOwnerDbId0%2CFOwnerDbId1&act=search"
	parseLinkDomoplius   = "https://m.domoplius.lt/skelbimai/butai?action_type=1&address_1=461&category_id=1"
	parseLinkKampas      = "https://www.kampas.lt/api/classifieds/search-new?query=%7B%22sort%22%3A%22new%22%2C%22municipality%22%3A58%2C%22settlement%22%3A19220%2C%22type%22%3A%22flat%22%2C%22taxonomyslug%22%3A%22sale%22%7D"
	parseLinkNuomininkai = "https://nuomininkai.lt/paieska/?propery_type=butu-nuoma&propery_contract_type=&propery_location=461&imic_property_district=&new_quartals=&min_price=&max_price=&min_price_meter=&max_price_meter=&min_area=&max_area=&rooms_from=&rooms_to=&high_from=&high_to=&floor_type=&irengimas=&building_type=&house_year_from=&house_year_to=&zm_skaicius=&lot_size_from=&lot_size_to=&by_date="
	parseLinkRinka       = "https://www.rinka.lt/vilnius/nekilnojamojo-turto-skelbimai/parduodami-butai?order_type=newest"
	parseLinkSkelbiu     = "https://www.skelbiu.lt/skelbimai/?autocompleted=1&keywords=&submit_bn=&cost_min=&cost_max=&space_min=&space_max=&rooms_min=&rooms_max=&year_min=&year_max=&building=0&status=0&floor_min=&floor_max=&floor_type=0&price_per_unit_min=&price_per_unit_max=&searchAddress=&district=0&quarter=0&streets=0&ignorestreets=0&cities=465&distance=0&mainCity=1&search=1&category_id=41&type=0&user_type=0&ad_since_min=0&ad_since_max=0&visited_page=1&orderBy=1&detailsSearch=1"
)

func compileAddressWithStreet(state, street, houseNumber string) (address string) {
	address = compileAddress(state, street+" "+houseNumber)
	return
}

func compileAddress(state, street string) (address string) {
	address = "Vilnius"
	if state != "" {
		address += ", " + state
	}
	if street != "" {
		address += ", " + street
	}
	return
}

var httpClient = &http.Client{Timeout: time.Second * 30}

func fetch(link string) ([]byte, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		u, _ := url.Parse(link)
		return nil, fmt.Errorf("%s returned: %s %s", u.Host, res.Status, string(content))
	}

	return content, nil
}

func fetchDocument(link string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		u, _ := url.Parse(link)
		return nil, fmt.Errorf("%s returned: %s", u.Host, res.Status)
	}

	return goquery.NewDocumentFromReader(res.Body)
}
