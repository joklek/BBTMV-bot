package website

import (
	"context"
	"errors"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	_ "github.com/chromedp/chromedp"

	_ "fmt"
	_ "github.com/chromedp/cdproto/emulation"
)

var netClient = &http.Client{
	Timeout: time.Second * 10,
}

func CreateChromeContext(link string) (context.Context, error) {
	ctx, _ := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)

	// create a timeout
	ctx, _ = context.WithTimeout(ctx, 60*time.Second)

	var err = chromedp.Run(ctx,
		emulation.SetUserAgentOverride("WebScraper 1.0"),
		chromedp.Navigate(link),
	)

	return ctx, err
}

func ScrapeExistingText(ctx context.Context, selector string) (string, error) {
	// navigate to a page, wait for an element, click
	var value string
	var err = chromedp.Run(
		ctx,
		chromedp.Text(selector, &value, chromedp.ByQueryAll),
	)

	return value, err
}

func ScrapeExistingNodes(ctx context.Context, selector string) ([]*cdp.Node, error) {
	// navigate to a page, wait for an element, click
	var value []*cdp.Node
	var err = chromedp.Run(
		ctx,
		chromedp.Nodes(selector, &value, chromedp.ByQueryAll),
	)

	return value, err
}

func GetResponseChrome(link string, selector string) ([]*cdp.Node, error) {
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// navigate to a page, wait for an element, click
	var nodes []*cdp.Node
	var err = chromedp.Run(ctx,
		emulation.SetUserAgentOverride("WebScraper 1.0"),
		chromedp.Navigate(link),
		chromedp.ScrollIntoView(`footer`),
		// wait for element to be visible (ie, page is loaded)
		chromedp.WaitVisible("body > div"),
		chromedp.Nodes(selector, &nodes, chromedp.ByQueryAll),
	)

	return nodes, err
}

func GetResponse(link string) (*http.Response, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}

	myURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Host", myURL.Host)
	req.Header.Set("User-Agent", "'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36'")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	//var path = URLRegex.FindAllStringSubmatch(link, -1)
	req.Header.Set("cache-control", "max-age=0")

	resp, err := netClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		linkURL, err := url.Parse(link)
		if err != nil {
			return nil, errors.New("unable to parse link " + link)
		}
		redirectURL, err := url.Parse(resp.Header.Get("Location"))
		if err != nil {
			return nil, errors.New("unable to parse HTTP header \"Location\" of link " + link + " after redirection")
		}
		newLink := linkURL.ResolveReference(redirectURL)
		return GetResponse(newLink.String())
	}

	return nil, errors.New(link + " returned HTTP code " + strconv.Itoa(resp.StatusCode))
}

func CompileAddress(district, street string) (address string) {
	address = "Vilnius"
	if district != "" {
		address += ", " + district
	}
	if street != "" {
		address += ", " + street
	}
	return
}

func CompileAddressWithStreet(district, street, houseNumber string) (address string) {
	address = CompileAddress(district, street+" "+houseNumber)
	return
}
