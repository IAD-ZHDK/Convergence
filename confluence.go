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

	contentCache  *cache.Cache
	downloadCache *cache.Cache
	client        *gorequest.SuperAgent
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

type Download struct {
	Data        []byte
	ContentType string
}

var ErrNotFound = errors.New("not found")

func NewConfluence() *Confluence {
	c := &Confluence{
		client: gorequest.New(),
	}

	c.Reset()

	return c
}

func (c *Confluence) url(path string) string {
	return c.BaseURL + "wiki/rest/api/" + path
}

func (c *Confluence) GetSpaces() ([]*Space, error) {
	cacheKey := "spaces"

	if value, ok := c.contentCache.Get(cacheKey); ok {
		return value.([]*Space), nil
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
			ID:    obj.Path("homepage.id").Data().(string),
			Title: obj.Path("homepage.title").Data().(string),
			Body:  obj.Path("homepage.body.view.value").Data().(string),
		}

		spaces[i] = space
	}

	c.contentCache.Set(cacheKey, spaces, cache.DefaultExpiration)

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

func (c *Confluence) GetPageByID(key, id string) (*Page, error) {
	cacheKey := "page-" + key + "-" + id

	if value, ok := c.contentCache.Get(cacheKey); ok {
		return value.(*Page), nil
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

	obj, err := gabs.ParseJSON([]byte(res))
	if err != nil {
		return nil, err
	}

	page := &Page{}

	page.ID = obj.Path("id").Data().(string)
	page.Title = obj.Path("title").Data().(string)
	page.Body = obj.Path("body.view.value").Data().(string)

	c.contentCache.Set(cacheKey, page, cache.DefaultExpiration)

	return page, nil
}

func (c *Confluence) GetPage(key, title string) (*Page, error) {
	cacheKey := "page-" + key + "-" + title

	if value, ok := c.contentCache.Get(cacheKey); ok {
		return value.(*Page), nil
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

	c.contentCache.Set(cacheKey, page, cache.DefaultExpiration)

	return page, nil
}

func (c *Confluence) GetDownload(typ, id, file, version, date, api string) (*Download, error) {
	cacheKey := typ + id + file + version + date + api

	if value, ok := c.downloadCache.Get(cacheKey); ok {
		return value.(*Download), nil
	}

	res, buf, errs := c.client.Get(c.BaseURL+"wiki/download/"+typ+"/"+id+"/"+file).
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

	download := &Download{
		ContentType: res.Header.Get("Content-Type"),
		Data:        buf,
	}

	c.downloadCache.Set(cacheKey, download, cache.DefaultExpiration)

	return download, nil
}

func (c *Confluence) Reset() {
	c.contentCache = cache.New(30*time.Minute, time.Minute)
	c.downloadCache = cache.New(24*time.Hour, time.Hour)
}
