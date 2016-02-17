package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/parnurzeal/gorequest"
	"github.com/patrickmn/go-cache"
)

type Confluence struct {
	BaseURL  string
	Username string
	Password string

	cache  *cache.Cache
	client *gorequest.SuperAgent
}

type Space struct {
	ID          string
	Key         string
	Name        string
	Type        string
	Link        string
	Description string
	HomepageID  string
}

type Page struct {
	ID     string
	Type   string
	Status string
	Title  string
	Link   string
	Body   string
}

func NewConfluence() *Confluence {
	return &Confluence{
		cache:  cache.New(5*time.Minute, 30*time.Second),
		client: gorequest.New(),
	}
}

func (c *Confluence) GetSpaces() ([]*Space, error) {
	_, body, errs := c.client.Get(c.BaseURL+"/space").
		Set("Accept", "application/json, */*").
		Query("expand=description.view").
		SetBasicAuth(c.Username, c.Password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("zero response")
	}

	json, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		return nil, err
	}

	array, err := json.Path("results").Children()
	if err != nil {
		return nil, err
	}

	spaces := make([]*Space, len(array))

	for i, obj := range array {
		space := &Space{}

		space.ID, _ = obj.Path("id").Data().(string)
		space.Key, _ = obj.Path("key").Data().(string)
		space.Name, _ = obj.Path("name").Data().(string)
		space.Type, _ = obj.Path("type").Data().(string)
		space.Link, _ = obj.Path("_links.self").Data().(string)
		space.Description, _ = obj.Path("description.view.value").Data().(string)

		linkHomepage, _ := obj.Path("_expandable.homepage").Data().(string)
		space.HomepageID = strings.Replace(linkHomepage, "/rest/api/content/", "", -1)

		spaces[i] = space
	}

	return spaces, nil
}

func (c *Confluence) GetPage(id string) (*Page, error) {
	_, body, errs := c.client.Get(c.BaseURL+"/content/"+id).
		Set("Accept", "application/json, */*").
		Query("expand=body.view").
		SetBasicAuth(c.Username, c.Password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("zero response")
	}

	json, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		return nil, err
	}

	page := &Page{}

	page.ID, _ = json.Path("id").Data().(string)
	page.Type, _ = json.Path("type").Data().(string)
	page.Status, _ = json.Path("status").Data().(string)
	page.Title, _ = json.Path("title").Data().(string)
	page.Link, _ = json.Path("title").Data().(string)
	page.Body, _ = json.Path("body.view.value").Data().(string)

	linkBase, _ := json.Path("_links.base").Data().(string)
	linkWeb, _ := json.Path("_links.webui").Data().(string)
	page.Link = linkBase + "/" + linkWeb

	return page, nil
}
