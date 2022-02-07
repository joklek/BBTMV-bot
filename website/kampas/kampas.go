package kampas

import (
	"bbtmvbot/database"
	"bbtmvbot/website"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

type Kampas struct{}

type kampasPosts struct {
	Hits []struct {
		ID          int      `json:"id"`
		Title       string   `json:"title"`
		Objectprice int      `json:"objectprice"`
		Objectarea  int      `json:"objectarea"`
		Totalfloors int      `json:"totalfloors"`
		Totalrooms  int      `json:"totalrooms"`
		Objectfloor int      `json:"objectfloor"`
		Yearbuilt   int      `json:"yearbuilt"`
		Description string   `json:"description"`
		Features    []string `json:"features"`
	} `json:"hits"`
}

const LINK = "https://www.kampas.lt/api/classifieds/search-new?query={%22municipality%22%3A%2258%22%2C%22settlement%22%3A19220%2C%22page%22%3A1%2C%22sort%22%3A%22new%22%2C%22section%22%3A%22bustas-nuomai%22%2C%22type%22%3A%22flat%22}"

func (obj *Kampas) Retrieve(db *database.Database) []*website.Post {
	posts := make([]*website.Post, 0)

	res, err := website.GetResponse(LINK)
	if err != nil {
		return posts
	}
	defer res.Body.Close()

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return posts
	}

	var results kampasPosts
	json.Unmarshal(contents, &results)

	for _, v := range results.Hits {
		p := &website.Post{}

		p.Link = fmt.Sprintf("https://www.kampas.lt/skelbimai/%d", v.ID) // https://www.kampas.lt/skelbimai/504506

		if db.InDatabase(p.Link) {
			continue
		}

		// Extract heating
		for _, feature := range v.Features {
			if strings.HasSuffix(feature, "_heating") {
				p.Heating = strings.ReplaceAll(feature, "_heating", "")
				break
			}
		}
		p.Heating = strings.ReplaceAll(p.Heating, "gas", "dujinis")
		p.Heating = strings.ReplaceAll(p.Heating, "central", "centrinis")
		p.Heating = strings.ReplaceAll(p.Heating, "thermostat", "termostatas")

		//p.Phone = "" // Impossible
		p.Description = strings.ReplaceAll(v.Description, "<br/>", "\n")
		p.District = strings.Split(v.Title, ", ")[1]
		p.Street = strings.Split(v.Title, ", ")[2]
		p.HouseNumber = ""
		p.Floor = v.Objectfloor
		p.FloorTotal = v.Totalfloors
		p.Area = v.Objectarea
		p.Price = v.Objectprice
		p.Rooms = v.Totalrooms
		p.Year = v.Yearbuilt

		p.TrimFields()
		posts = append(posts, p)
	}

	return posts
}

func init() {
	website.Add("kampas", &Kampas{})
}
