package main

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var regexDomopliusExtractNumberMap = regexp.MustCompile(`(\w+)='([^']+)'`)
var regexDomopliusExtractNumberSeq = regexp.MustCompile(`document\.write\(([\w+]+)\);`)
var regexDomopliusExtractFloors = regexp.MustCompile(`(\d+), (\d+) `)

func parseDomoplius() {

	url := "https://m.domoplius.lt/skelbimai/butai?action_type=3&address_1=461&sell_price_from=&sell_price_to=&qt="

	// Get content as Goquery Document:
	doc, err := getGoqueryDocument(url)
	if err != nil {
		log.Println(err)
		return
	}

	// For each post in page:
	doc.Find("ul.list > li[id^=\"ann_\"]").Each(func(i int, s *goquery.Selection) {

		// Get postURL:
		postUpstreamID, exists := s.Attr("id")
		if !exists {
			return
		}
		link := "https://m.domoplius.lt/skelbimai/-" + strings.ReplaceAll(postUpstreamID, "ann_", "") + ".html" // https://m.domoplius.lt/skelbimai/-5806213.html

		// Skip if post already in DB:
		exists, err := postURLInDB(link)
		if err != nil {
			log.Println(err)
			return
		}
		if exists {
			return
		}

		// Get post's content as Goquery Document:
		postDoc, err := getGoqueryDocument(link)
		if err != nil {
			log.Println(err)
			return
		}

		// ------------------------------------------------------------
		p := post{url: link}

		// Extract phone:
		tmp, err := postDoc.Find("#phone_button_4").Html()
		if err == nil {
			tmp = domopliusDecodeNumber(tmp)
			p.phone = strings.ReplaceAll(tmp, " ", "")
		}

		// Extract description:
		p.description = postDoc.Find("div.container > div.group-comments").Text()

		// Extract address:
		tmp = postDoc.Find(".panel > .container > .container > h1").Text()
		if tmp != "" {
			p.address = strings.Split(tmp, "nuoma ")[1]
		}

		// Extract heating:
		el := postDoc.Find(".view-field-title:contains(\"Šildymas:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			p.heating = el.Text()
		}

		// Extract floor and floor total:
		el = postDoc.Find(".view-field-title:contains(\"Aukštas:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = strings.TrimSpace(el.Text())
			arr := regexDomopliusExtractFloors.FindStringSubmatch(tmp)
			p.floor, _ = strconv.Atoi(tmp) // will be 0 on failure, will be number if success
			if len(arr) == 3 {
				p.floor, _ = strconv.Atoi(arr[1])
				p.floorTotal, _ = strconv.Atoi(arr[2])
			}
		}

		// Extract area:
		el = postDoc.Find(".view-field-title:contains(\"Buto plotas (kv. m):\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			tmp = strings.Split(tmp, ".")[0]
			p.area, _ = strconv.Atoi(tmp)
		}

		// Extract price:
		tmp = postDoc.Find(".field-price > .price-column > .h1").Text()
		if tmp != "" {
			tmp = strings.TrimSpace(tmp)
			tmp = strings.ReplaceAll(tmp, " ", "")
			tmp = strings.ReplaceAll(tmp, "€", "")
			p.price, _ = strconv.Atoi(tmp)
		}

		// Extract rooms:
		el = postDoc.Find(".view-field-title:contains(\"Kambarių skaičius:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			p.rooms, _ = strconv.Atoi(tmp)
		}

		// Extract year:
		el = postDoc.Find(".view-field-title:contains(\"Statybos metai:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			p.year, _ = strconv.Atoi(tmp)
		}

		go p.processPost()
	})

}

func domopliusDecodeNumber(str string) string {
	// Create map:
	arr := regexDomopliusExtractNumberMap.FindAllSubmatch([]byte(str), -1)
	mymap := make(map[string]string, len(arr))
	for _, v := range arr {
		mymap[string(v[1])] = string(v[2])
	}

	// Create sequence:
	arr = regexDomopliusExtractNumberSeq.FindAllSubmatch([]byte(str), -1)
	var seq string
	for _, v := range arr {
		seq += "+" + string(v[1])
	}
	seq = strings.TrimLeft(seq, "+")

	// Split sequence into array:
	splittedSeq := strings.Split(seq, "+")

	// Build final string:
	var msg string
	for _, v := range splittedSeq {
		msg += mymap[v]
	}

	return strings.ReplaceAll(msg, " ", "")
}
