package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var regexDomopliusExtractFloors = regexp.MustCompile(`(\d+), (\d+) `)

func parseDomoplius() {
	// Download page
	doc, err := fetchDocument(parseLinkDomoplius)
	if err != nil {
		log.Println(err)
		return
	}

	// Iterate posts in webpage
	doc.Find("ul.list > li[id^=\"ann_\"]").Each(func(i int, s *goquery.Selection) {

		p := &Post{}

		upstreamID, ok := s.Attr("id")
		if !ok {
			return
		}
		p.Link = "https://m.domoplius.lt/skelbimai/-" + strings.ReplaceAll(upstreamID, "ann_", "") + ".html" // https://m.domoplius.lt/skelbimai/-5806213.html

		// Skip if already in database:
		if p.InDatabase() {
			return
		}

		// Get post's content as Goquery Document:
		postDoc, err := fetchDocument(p.Link)
		if err != nil {
			log.Println(err)
			return
		}

		// ------------------------------------------------------------

		// Extract phone:
		tmp, exists := postDoc.Find("#phone_button_4 > span").Attr("data-value")
		if exists {
			p.Phone = domopliusDecodeNumber(tmp)
		}

		// Extract description:
		p.Description = postDoc.Find("div.container > div.group-comments").Text()

		// Extract address:
		tmp = ""
		postDoc.Find(".breadcrumb-item > a > span[itemprop=name]").Each(func(i int, selection *goquery.Selection) {
			if i != 0 {
				tmp += ", "
			}
			tmp += selection.Text()
		})
		if tmp != "" {
			p.Address = tmp
		}

		// Extract heating:
		el := postDoc.Find(".view-field-title:contains(\"Šildymas:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			p.Heating = el.Text()
		}

		// Extract floor and floor total:
		el = postDoc.Find(".view-field-title:contains(\"Aukštas:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = strings.TrimSpace(el.Text())
			arr := regexDomopliusExtractFloors.FindStringSubmatch(tmp)
			p.Floor, _ = strconv.Atoi(tmp) // will be 0 on failure, will be number if success
			if len(arr) == 3 {
				p.Floor, _ = strconv.Atoi(arr[1])
				p.FloorTotal, _ = strconv.Atoi(arr[2])
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
			p.Area, _ = strconv.Atoi(tmp)
		}

		// Extract price:
		tmp = postDoc.Find(".field-price > .price-column > .h1").Text()
		if tmp != "" {
			tmp = strings.TrimSpace(tmp)
			tmp = strings.ReplaceAll(tmp, " ", "")
			tmp = strings.ReplaceAll(tmp, "€", "")
			p.Price, _ = strconv.Atoi(tmp)
		}

		// Extract rooms:
		el = postDoc.Find(".view-field-title:contains(\"Kambarių skaičius:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			p.Rooms, _ = strconv.Atoi(tmp)
		}

		// Extract year:
		el = postDoc.Find(".view-field-title:contains(\"Statybos metai:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			p.Year, _ = strconv.Atoi(tmp)
		}

		go p.Handle()
	})
}

func domopliusDecodeNumber(str string) string {

	msgRaw, err := base64.StdEncoding.DecodeString(str[2:])
	if err != nil {
		fmt.Printf("Error decoding string: %s ", err.Error())
		return ""
	}
	msg := strings.ReplaceAll(string(msgRaw), " ", "")

	// Replace 86 in the beginning to +3706
	if strings.HasPrefix(msg, "86") {
		return strings.Replace(msg, "86", "+3706", 1)
	}

	return msg
}
