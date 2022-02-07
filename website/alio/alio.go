package alio

import (
	"bbtmvbot/database"
	"bbtmvbot/website"
	"log"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Alio struct{}

const LINK = "https://www.alio.lt/paieska/?category_id=1393&city_id=228626&search_block=1&search[eq][adresas_1]=228626&order=ad_id"

func (obj *Alio) Retrieve(db *database.Database) []*website.Post {
	posts := make([]*website.Post, 0)

	res, err := website.GetResponse(LINK)
	if err != nil {
		return posts
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return posts
	}

	doc.Find("#main_left_b > #main-content-center > div.result").Each(func(i int, s *goquery.Selection) {
		p := &website.Post{}

		upstreamID, ok := s.Attr("id")
		if !ok {
			log.Println("Post ID is not found in 'alio' website")
			return
		}
		p.Link = "https://www.alio.lt/skelbimai/ID" + strings.ReplaceAll(upstreamID, "lv_ad_id_", "") + ".html" // https://www.alio.lt/skelbimai/ID60331923.html

		if db.InDatabase(p.Link) {
			return
		}

		postRes, err := website.GetResponse(p.Link)
		if err != nil {
			return
		}
		defer postRes.Body.Close()
		postDoc, err := goquery.NewDocumentFromReader(postRes.Body)
		if err != nil {
			return
		}

		// Extract phone:
		p.Phone = postDoc.Find("#phone_val_value").Text()

		// Extract description:
		p.Description = postDoc.Find("#adv_description_b > .a_line_val").Text()

		// Extract address:
		el := postDoc.Find("div#path > .path_i > a > span")
		if el.Length() > 4 {
			el.Each(func(i int, selection *goquery.Selection) {
				if i == 4 {
					p.District = selection.Text()
				}
				if i == 5 {
					p.Street = selection.Text()
				}
			})
		}
		p.HouseNumber = ""

		// Extract heating:
		el = postDoc.Find(".data_moreinfo_b:contains(\"Šildymas\")")
		if el.Length() != 0 {
			p.Heating = el.Find(".a_line_val").Text()
		}

		// Extract floor:
		tmp := ""
		el = postDoc.Find(".data_moreinfo_b:contains(\"Buto aukštas\")")
		if el.Length() != 0 {
			tmp = el.Find(".a_line_val").Text()
			tmp = strings.TrimSpace(tmp)
			p.Floor, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Floor number from 'alio' post")
				return
			}
		}

		// Extract floor total:
		el = postDoc.Find(".data_moreinfo_b:contains(\"Aukštų skaičius pastate\")")
		if el.Length() != 0 {
			tmp = el.Find(".a_line_val").Text()
			tmp = strings.TrimSpace(tmp)
			p.FloorTotal, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract FloorTotal number from 'alio' post")
				return
			}
		}

		// Extract area:
		el = postDoc.Find(".data_moreinfo_b:contains(\"Buto plotas\")")
		if el.Length() != 0 {
			tmp = el.Find(".a_line_val").Text()
			tmp = strings.TrimSpace(tmp)
			tmp = strings.Split(tmp, " ")[0]
			var tmpArea, err = strconv.ParseFloat(tmp, 32) // Area is represented as a float and Atoi does not work on it
			if err != nil {
				log.Println("failed to extract Area number from 'alio' post")
				return
			}
			p.Area = int(tmpArea)
		}

		// Extract price:
		el = postDoc.Find(".data_moreinfo_b:contains(\"Kaina, €\")").First()
		if el.Length() != 0 {
			tmp = el.Find(".a_line_val").Text()
			tmp = strings.TrimSpace(tmp)
			tmp = strings.Split(tmp, " ")[0]
			if strings.Contains(tmp, ".") {
				tmp = strings.Split(tmp, ".")[0]
			}
			p.Price, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Price number from 'alio' post")
				return
			}
		}

		// Extract rooms:
		el = postDoc.Find(".data_moreinfo_b:contains(\"Kambarių skaičius\")")
		if el.Length() != 0 {
			tmp = el.Find(".a_line_val").Text()
			tmp = strings.TrimSpace(tmp)
			p.Rooms, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Rooms number from 'alio' post")
				return
			}
		}

		// Extract year:
		el = postDoc.Find(".data_moreinfo_b:contains(\"Statybos metai\")")
		if el.Length() != 0 {
			tmp = el.Find(".a_line_val").Text()
			tmp = strings.TrimSpace(tmp)
			tmp = strings.Split(tmp, " ")[0]
			p.Year, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Year number from 'alio' post")
				return
			}
		}

		p.TrimFields()
		posts = append(posts, p)
	})

	return posts
}

func init() {
	website.Add("alio", &Alio{})
}
