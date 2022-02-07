package domoplius

import (
	"bbtmvbot/database"
	"bbtmvbot/website"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Domoplius struct{}

const LINK = "https://m.domoplius.lt/skelbimai/butai?action_type=3&address_1=461&sell_price_from=&sell_price_to=&qt="

var reExtractFloors = regexp.MustCompile(`(\d+), (\d+) `)

func (obj *Domoplius) Retrieve(db *database.Database) []*website.Post {
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

	doc.Find("ul.list > li[id^='ann_']").Each(func(i int, s *goquery.Selection) {
		p := &website.Post{}

		upstreamID, ok := s.Attr("id")
		if !ok {
			log.Println("Post ID is not found in 'domoplius' website")
			return
		}
		p.Link = "https://domoplius.lt/skelbimai/-" + strings.ReplaceAll(upstreamID, "ann_", "") + ".html" // https://domoplius.lt/skelbimai/-5806213.html

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
		tmp, exists := postDoc.Find("#phone_button_4 > span").Attr("data-value")
		if exists {
			p.Phone = domopliusDecodeNumber(tmp)
		}

		// Extract description:
		p.Description = postDoc.Find("div.container > div.group-comments").Text()

		// Extract address:
		addressBreadcrumbs := postDoc.Find(".breadcrumb-item > a > span[itemprop=name]")

		addressBreadcrumbs.Each(func(i int, selection *goquery.Selection) {
			if i == 1 {
				p.District = selection.Text()
			}
			if i == 2 {
				p.Street = selection.Text()
			}
		})
		p.HouseNumber = ""

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
			arr := reExtractFloors.FindStringSubmatch(tmp)
			p.Floor, _ = strconv.Atoi(tmp) // will be 0 on failure, will be number if success
			if len(arr) == 3 {
				p.Floor, err = strconv.Atoi(arr[1])
				if err != nil {
					log.Println("failed to extract Floor number from 'domoplius' post")
					return
				}
				p.FloorTotal, err = strconv.Atoi(arr[2])
				if err != nil {
					log.Println("failed to extract FloorTotal number from 'domoplius' post")
					return
				}
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
			p.Area, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Area number from 'domoplius' post")
				return
			}
		}

		// Extract price:
		tmp = postDoc.Find(".field-price > .price-column > .h1").Text()
		if tmp != "" {
			tmp = strings.TrimSpace(tmp)
			tmp = strings.ReplaceAll(tmp, " ", "")
			tmp = strings.ReplaceAll(tmp, "€", "")
			p.Price, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Price number from 'domoplius' post")
				return
			}
		}

		// Extract rooms:
		el = postDoc.Find(".view-field-title:contains(\"Kambarių skaičius:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			p.Rooms, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Rooms number from 'domoplius' post")
				return
			}
		}

		// Extract year:
		el = postDoc.Find(".view-field-title:contains(\"Statybos metai:\")")
		if el.Length() != 0 {
			el = el.Parent()
			el.Find("span").Remove()
			tmp = el.Text()
			tmp = strings.TrimSpace(tmp)
			p.Year, err = strconv.Atoi(tmp)
			if err != nil {
				log.Println("failed to extract Year number from 'domoplius' post")
				return
			}
		}

		p.TrimFields()
		posts = append(posts, p)
	})

	return posts
}

func domopliusDecodeNumber(str string) string {
	msg, err := base64.StdEncoding.DecodeString(str[2:])
	if err != nil {
		fmt.Printf("Error decoding string: %s ", err.Error())
		return ""
	}

	return string(msg)
}

func init() {
	website.Add("domoplius", &Domoplius{})
}
