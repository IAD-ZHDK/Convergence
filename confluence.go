package main

import (
	"time"
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/parnurzeal/gorequest"
	"github.com/Jeffail/gabs"
)

type Confluence struct {
	BaseURL string
	Username string
	Password string

	cache  *cache.Cache
	client *gorequest.SuperAgent
}

type Space struct {
	ID string
	Key string
	Name string
	Type string
	Link string
	Description string
	HomepageRef string
}

func NewConfluence() *Confluence {
	return &Confluence{
		cache: cache.New(5 * time.Minute, 30 * time.Second),
		client: gorequest.New(),
	}
}

func (c *Confluence) GetSpaces() ([]Space, error) {
	_, body, errs := c.client.Get(c.BaseURL + "/space").
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

	spaces := make([]Space, len(array))

	for i, obj := range array {
		spaces[i].ID, _ = obj.Path("id").Data().(string)
		spaces[i].Key, _ = obj.Path("key").Data().(string)
		spaces[i].Name, _ = obj.Path("name").Data().(string)
		spaces[i].Type, _ = obj.Path("type").Data().(string)
		spaces[i].Link, _ = obj.Path("_links.self").Data().(string)
		spaces[i].Description, _ = obj.Path("description.view.value").Data().(string)
		spaces[i].HomepageRef, _ = obj.Path("_expandable.homepage").Data().(string)

	}

	return spaces, nil
}
