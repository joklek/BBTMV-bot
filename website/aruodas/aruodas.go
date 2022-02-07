package aruodas

import (
	"bbtmvbot/database"
	"bbtmvbot/website"
	"log"
	"strconv"
	"strings"

	"github.com/chromedp/cdproto/cdp"
)

type Aruodas struct{}

const LINK = "https://m.aruodas.lt/?obj=4&FRegion=461&FDistrict=1&FOrder=AddDate&from_search=1&detailed_search=1&FShowOnly=FOwnerDbId0%2CFOwnerDbId1&act=search"

func processItem(node *cdp.Node, posts []*website.Post, db *database.Database) {
	p := &website.Post{}

	upstreamID, ok := node.Attribute("data-id")
	if !ok {
		log.Println("Post ID is not found in 'aruodas' website")
		return
	}
	p.Link = "https://aruodas.lt/" + strings.ReplaceAll(upstreamID, "loadObject", "") // https://aruodas.lt/4-919937

	if db.InDatabase(p.Link) {
		return
	}

	chromeContext, err := website.CreateChromeContext(p.Link)
	if err != nil {
		return
	}

	var tmp string

	// Extract phone:
	tempPhone, err := website.ScrapeExistingText(chromeContext, "span.phone_item_0")
	if err != nil {
		return
	}

	if tempPhone == "" {
		tempPhone, err = website.ScrapeExistingText(chromeContext, "div.phone")

		if err != nil {
			return
		}
	}
	p.Phone = tempPhone

	// Extract description:
	p.Description, err = website.ScrapeExistingText(chromeContext, "#collapsedTextBlock > #collapsedText")
	if err != nil {
		return
	}

	// Extract address:
	temp, err := website.ScrapeExistingText(chromeContext, ".main-content > .obj-cont > h1")
	if err != nil {
		return
	}

	splitAddress := strings.Split(temp, ",")

	dlList, err := website.ScrapeExistingNodes(chromeContext, "dl dt, dl dd")
	var houseNumberIndex = -1
	var heatingIndex = -1
	var floorIndex = -1
	var floorTotalIndex = -1
	var areaIndex = -1
	var priceIndex = -1
	var roomsIndex = -1
	var yearIndex = -1
	for index, keyNode := range dlList {
		if keyNode.LocalName != "dt" {
			continue
		}

		var value = strings.TrimSpace(keyNode.Children[0].NodeValue)

		if strings.Contains(value, "Namo numeris") {
			houseNumberIndex = index + 1
		} else if strings.Contains(value, "Šildymas") {
			heatingIndex = index + 1
		} else if strings.Contains(value, "Aukštas") {
			floorIndex = index + 1
		} else if strings.Contains(value, "Aukštų sk.") {
			floorTotalIndex = index + 1
		} else if strings.Contains(value, "Plotas") {
			areaIndex = index + 1
		} else if strings.Contains(value, "Kaina mėn.") {
			priceIndex = index + 1
		} else if strings.Contains(value, "Kambarių sk.") {
			roomsIndex = index + 1
		} else if strings.Contains(value, "Metai") {
			yearIndex = index + 1
		}
	}

	// Extract house number
	p.District = splitAddress[1]
	p.Street = splitAddress[2]
	if houseNumberIndex != -1 {
		var houseNumber = dlList[houseNumberIndex].Value
		p.HouseNumber = houseNumber
	} else {
		p.HouseNumber = ""
	}

	// Extract heating:
	if heatingIndex != -1 {
		p.Heating = strings.TrimSpace(dlList[heatingIndex].Children[0].NodeValue)
	}

	// Extract floor:
	if floorIndex != -1 {
		tmp = strings.TrimSpace(dlList[floorIndex].Children[0].NodeValue)
		p.Floor, err = strconv.Atoi(tmp)
		if err != nil {
			log.Println("failed to extract Floor number from 'aruodas' post")
			return
		}
	}

	// Extract floor total:
	if floorTotalIndex != -1 {
		tmp = strings.TrimSpace(dlList[floorTotalIndex].Children[0].NodeValue)
		p.FloorTotal, err = strconv.Atoi(tmp)
		if err != nil {
			log.Println("failed to extract FloorTotal number from 'aruodas' post")
			return
		}
	}

	// Extract area:
	if areaIndex != -1 {
		tmp = strings.TrimSpace(dlList[areaIndex].Children[0].NodeValue)
		if strings.Contains(tmp, ",") {
			tmp = strings.Split(tmp, ",")[0]
		} else {
			tmp = strings.Split(tmp, " ")[0]
		}
		p.Area, err = strconv.Atoi(tmp)
		if err != nil {
			log.Println("failed to extract Area number from 'aruodas' post")
			return
		}
	}

	// Extract price:
	if priceIndex != -1 {
		tmp = strings.TrimSpace(dlList[priceIndex].Children[0].NodeValue)
		tmp = strings.ReplaceAll(tmp, " ", "")
		tmp = strings.ReplaceAll(tmp, "€", "")
		p.Price, err = strconv.Atoi(tmp)
		if err != nil {
			log.Println("failed to extract Price number from 'aruodas' post")
			return
		}
	}

	// Extract rooms:
	if roomsIndex != -1 {
		tmp = strings.TrimSpace(dlList[roomsIndex].Children[0].NodeValue)
		p.Rooms, err = strconv.Atoi(tmp)
		if err != nil {
			log.Println("failed to extract Rooms number from 'aruodas' post")
			return
		}
	}

	// Extract year:
	if yearIndex != -1 {
		tmp = strings.TrimSpace(dlList[yearIndex].Children[0].NodeValue)
		if strings.Contains(tmp, " ") {
			tmp = strings.Split(tmp, " ")[0]
		}
		p.Year, err = strconv.Atoi(tmp)
		if err != nil {
			log.Println("failed to extract Year number from 'aruodas' post")
			return
		}
	}

	p.TrimFields()
	posts = append(posts, p)
}

func (obj *Aruodas) Retrieve(db *database.Database) []*website.Post {
	posts := make([]*website.Post, 0)

	//res, err := website.GetResponse(LINK)
	var chromeRes, err = website.GetResponseChrome(LINK, "ul.search-result-list-v2 > li.result-item-v3:not([style='display: none'])")
	if err != nil {
		return posts
	}

	for _, node := range chromeRes {
		processItem(node, posts, db)
	}

	return posts
}

func init() {
	website.Add("aruodas", &Aruodas{})
}
