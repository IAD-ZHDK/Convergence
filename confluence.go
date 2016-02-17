package main

import (
	"fmt"
	"html/template"
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
	BodyT  template.HTML
}

func NewConfluence() *Confluence {
	return &Confluence{
		cache:  cache.New(5*time.Minute, 30*time.Second),
		client: gorequest.New(),
	}
}

func (c *Confluence) url(path string) string {
	return c.BaseURL + "confluence/rest/api/" + path
}

func (c *Confluence) GetSpaces() ([]*Space, error) {
	fmt.Printf("Check cache for key 'spaces-all'.\n")

	if value, ok := c.cache.Get("spaces-all"); ok {
		fmt.Printf("Got spaces from cache with key 'spaces-all'.\n")

		spaces, _ := value.([]*Space)
		return spaces, nil
	}

	_, res, errs := c.client.Get(c.url("space")).
		Set("Accept", "application/json, */*").
		Query("expand=description.view").
		SetBasicAuth(c.Username, c.Password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("zero response")
	}

	json, err := gabs.ParseJSON([]byte(res))
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

	c.cache.Set("spaces-all", spaces, cache.DefaultExpiration)
	fmt.Printf("Put spaces in cache with key 'spaces-all'.\n")

	return spaces, nil
}

func (c *Confluence) GetSpace(key string) (*Space, error) {
	spaces, err := c.GetSpaces()
	if err != nil {
		return nil, err
	}

	for _, space := range spaces {
		if space.Key == key {
			return space, nil
		}
	}

	return nil, fmt.Errorf("space not found")
}

func (c *Confluence) GetPageByTitle(key, title string) (*Page, error) {
	cacheKey := "pages-"+key+"-"+title

	fmt.Printf("Check cache for key '%s'.\n", cacheKey)

	if value, ok := c.cache.Get(cacheKey); ok {
		fmt.Printf("Got page from cache with key '%s'.\n", cacheKey)

		page, _ := value.(*Page)
		return page, nil
	}

	_, res, errs := c.client.Get(c.url("content")).
		Set("Accept", "application/json, */*").
		Query("title="+title).
		Query("type=page").
		Query("spaceKey="+key).
		Query("expand=body.view").
		SetBasicAuth(c.Username, c.Password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("zero response")
	}

	json, err := gabs.ParseJSON([]byte(res))
	if err != nil {
		return nil, err
	}

	results, err := json.Path("results").Children()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return c.handlePageData(key, results[0])
}

func (c *Confluence) GetPageById(key, id string) (*Page, error) {
	cacheKey := "pages-"+key+"-"+id

	fmt.Printf("Check cache for key '%s'.\n", cacheKey)

	if value, ok := c.cache.Get(cacheKey); ok {
		fmt.Printf("Got page from cache with key '%s'.\n", cacheKey)

		page, _ := value.(*Page)
		return page, nil
	}

	_, res, errs := c.client.Get(c.url("content/"+id)).
		Set("Accept", "application/json, */*").
		Query("type=page").
		Query("spaceKey="+key).
		Query("expand=body.view").
		SetBasicAuth(c.Username, c.Password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("zero response")
	}

	json, err := gabs.ParseJSON([]byte(res))
	if err != nil {
		return nil, err
	}

	return c.handlePageData(key, json)
}

func (c *Confluence) handlePageData(key string, json *gabs.Container) (*Page, error) {
	page := &Page{}

	page.ID, _ = json.Path("id").Data().(string)
	page.Type, _ = json.Path("type").Data().(string)
	page.Status, _ = json.Path("status").Data().(string)
	page.Title, _ = json.Path("title").Data().(string)
	page.Link, _ = json.Path("title").Data().(string)

	body, _ := json.Path("body.view.value").Data().(string)
	body = strings.Replace(body, "wiki/display/", "", -1)
	body = strings.Replace(body, "/wiki/download/", c.BaseURL + "/wiki/download/", -1)

	page.Body = body
	page.BodyT = template.HTML(body)

	linkBase, _ := json.Path("_links.base").Data().(string)
	linkWeb, _ := json.Path("_links.webui").Data().(string)
	page.Link = linkBase + "/" + linkWeb

	cacheKey1 := "pages-"+key+"-"+page.ID
	cacheKey2 := "pages-"+key+"-"+strings.Replace(page.Title, " ", "+", -1)
	c.cache.Set(cacheKey1, page, cache.DefaultExpiration)
	c.cache.Set(cacheKey2, page, cache.DefaultExpiration)

	fmt.Printf("Put page in cache with key '%s' and '%s'.\n", cacheKey1, cacheKey2)

	return page, nil
}
