package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/microcosm-cc/bluemonday"
	"github.com/parnurzeal/gorequest"
	"github.com/patrickmn/go-cache"
)

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

type Response struct {
	Status int
	Data   []byte
	Header map[string][]string
}

var errNotFound = errors.New("not found")

type Confluence struct {
	baseURL  string
	username string
	password string

	contentCache  *cache.Cache
	downloadCache *cache.Cache
	responseCache *cache.Cache
	client        *gorequest.SuperAgent
	sanitizer     *bluemonday.Policy
}

func NewConfluence(baseURL, username, password string) *Confluence {
	c := &Confluence{
		baseURL:   baseURL,
		username:  username,
		password:  password,
		client:    gorequest.New(),
		sanitizer: bluemonday.UGCPolicy(),
	}

	c.sanitizer.RequireNoFollowOnLinks(false)
	c.sanitizer.RequireNoFollowOnFullyQualifiedLinks(true)
	c.sanitizer.AllowAttrs("class").Globally()

	c.Reset()

	return c
}

func (c *Confluence) url(path string) string {
	return c.baseURL + "wiki/rest/api/" + path
}

func (c *Confluence) GetSpaces() ([]*Space, error) {
	cacheKey := "spaces"

	if value, ok := c.contentCache.Get(cacheKey); ok {
		return value.([]*Space), nil
	}

	_, res, errs := c.client.Get(c.url("space")).
		Set("Accept", "application/json, */*").
		Query("expand=description.view,homepage.body.view").
		SetBasicAuth(c.username, c.password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(res) == 0 {
		return nil, errors.New("zero response")
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
			Body:  c.processBody(obj.Path("homepage.body.view.value").Data().(string)),
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

	return nil, errNotFound
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
		SetBasicAuth(c.username, c.password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(res) == 0 {
		return nil, errors.New("zero response")
	}

	obj, err := gabs.ParseJSON([]byte(res))
	if err != nil {
		return nil, err
	}

	page := &Page{}

	page.ID = obj.Path("id").Data().(string)
	page.Title = obj.Path("title").Data().(string)
	page.Body = c.processBody(obj.Path("body.view.value").Data().(string))

	c.contentCache.Set(cacheKey, page, cache.DefaultExpiration)

	return page, nil
}

func (c *Confluence) GetPageByTitle(key, title string) (*Page, error) {
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
		SetBasicAuth(c.username, c.password).
		End()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(res) == 0 {
		return nil, errors.New("zero response")
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
		return nil, errNotFound
	}

	obj := results[0]
	page := &Page{}

	page.ID = obj.Path("id").Data().(string)
	page.Title = obj.Path("title").Data().(string)
	page.Body = c.processBody(obj.Path("body.view.value").Data().(string))

	c.contentCache.Set(cacheKey, page, cache.DefaultExpiration)

	return page, nil
}

func (c *Confluence) GetDownload(typ, id, file, version, date, api string) (*Download, error) {
	// TODO: Delegate to proxy?

	cacheKey := typ + id + file + version + date + api

	if value, ok := c.downloadCache.Get(cacheKey); ok {
		return value.(*Download), nil
	}

	res, buf, errs := c.client.Get(c.baseURL+"wiki/download/"+typ+"/"+id+"/"+file).
		Set("Accept", "*/*").
		Query("version="+version).
		Query("modificationDate="+date).
		Query("api="+api).
		SetBasicAuth(c.username, c.password).
		EndBytes()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(buf) == 0 {
		return nil, errNotFound
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
	c.responseCache = cache.New(24*time.Hour, time.Hour)
}

func (c *Confluence) getResponse(r *http.Request) (*Response, error) {
	// check cache
	if value, ok := c.responseCache.Get(r.URL.RequestURI()); ok {
		return value.(*Response), nil
	}

	// make new request
	r2, err := http.NewRequest("GET", c.baseURL+r.URL.RequestURI(), r.Body)
	if err != nil {
		return nil, err
	}

	// add authentication
	r2.SetBasicAuth(c.username, c.password)

	// make request
	res, err := http.DefaultClient.Do(r2)
	if err != nil {
		return nil, err
	}

	// read full body
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// create response
	response := &Response{
		Status: res.StatusCode,
		Data:   buf,
		Header: res.Header,
	}

	// cache it
	c.responseCache.Set(r.URL.RequestURI(), response, cache.DefaultExpiration)

	return response, nil
}

func (c *Confluence) Proxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// only proxy get requests
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// get response
		res, err := c.getResponse(r)
		if err != nil {
			println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// add headers
		for key, values := range res.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// write head and body
		w.WriteHeader(res.Status)
		w.Write(res.Data)
	})
}

// TODO: Remove empty spans?

func (c *Confluence) processBody(body string) string {
	body = strings.Replace(body, "Expand source", "", -1)
	body = c.sanitizer.Sanitize(body)
	return strings.Replace(body, c.baseURL, "/", -1)
}
