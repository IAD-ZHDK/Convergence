package main

import (
	"errors"
	"fmt"
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
	Key         string
	Name        string
	Description string
	Homepage    Page
}

type Page struct {
	ID    string
	Title string
	Body  string
}

type Attachment struct {
	Data        []byte
	ContentType string
}

var ErrNotFound = errors.New("not found")

func NewConfluence() *Confluence {
	return &Confluence{
		cache:  cache.New(30*time.Minute, 5*time.Minute),
		client: gorequest.New(),
	}
}

func (c *Confluence) url(path string) string {
	return c.BaseURL + "wiki/rest/api/" + path
}

func (c *Confluence) GetSpaces() ([]*Space, error) {
	fmt.Printf("Check cache for key 'spaces-all'.\n")

	if value, ok := c.cache.Get("spaces-all"); ok {
		fmt.Printf("Got spaces from cache with key 'spaces-all'.\n")

		spaces := value.([]*Space)
		return spaces, nil
	}

	_, res, errs := c.client.Get(c.url("space")).
		Set("Accept", "application/json, */*").
		Query("expand=description.view,homepage.body.view").
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

		space.Key = obj.Path("key").Data().(string)
		space.Name = obj.Path("name").Data().(string)
		space.Description = obj.Path("description.view.value").Data().(string)
		space.Homepage = Page{
			ID: obj.Path("homepage.id").Data().(string),
			Title: obj.Path("homepage.title").Data().(string),
			Body: obj.Path("homepage.body.view.value").Data().(string),
		}

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

	return nil, ErrNotFound
}

func (c *Confluence) GetPageByTitle(key, title string) (*Page, error) {
	cacheKey := "pages-" + key + "-" + title

	fmt.Printf("Check cache for key '%s'.\n", cacheKey)

	if value, ok := c.cache.Get(cacheKey); ok {
		fmt.Printf("Got page from cache with key '%s'.\n", cacheKey)

		page := value.(*Page)
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
		return nil, ErrNotFound
	}

	obj := results[0]
	page := &Page{}

	page.ID = obj.Path("id").Data().(string)
	page.Title = obj.Path("title").Data().(string)
	page.Body = obj.Path("body.view.value").Data().(string)

	c.cache.Set(cacheKey, page, cache.DefaultExpiration)
	fmt.Printf("Put page in cache with key '%s'.\n", cacheKey)

	return page, nil
}

func (c *Confluence) GetAttachment(id, file, version, date, api string) (*Attachment, error) {
	cacheKey := "attachment-" + id + "-" + file + "-" + date

	fmt.Printf("Check cache for key '%s'.\n", cacheKey)

	if value, ok := c.cache.Get(cacheKey); ok {
		fmt.Printf("Got attachment from cache with key '%s'.\n", cacheKey)

		attachment := value.(*Attachment)
		return attachment, nil
	}

	res, buf, errs := c.client.Get(c.BaseURL+"wiki/download/attachments/"+id+"/"+file).
		Set("Accept", "*/*").
		Query("version="+version).
		Query("modificationDate="+date).
		Query("api="+api).
		SetBasicAuth(c.Username, c.Password).
		EndBytes()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(buf) == 0 {
		return nil, ErrNotFound
	}

	attachment := &Attachment{
		ContentType: res.Header.Get("Content-Type"),
		Data:        buf,
	}

	c.cache.Set(cacheKey, attachment, cache.DefaultExpiration)

	return attachment, nil
}

func (c *Confluence) Reset() {
	c.cache = cache.New(5*time.Minute, 30*time.Second)
}
