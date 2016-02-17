package main

import (
	"os"

	"github.com/kr/pretty"
)

func main() {
	c := NewConfluence()
	c.BaseURL = os.Getenv("BASE_URL")
	c.Username = os.Getenv("USERNAME")
	c.Password = os.Getenv("PASSWORD")

	spaces, err := c.GetSpaces()
	if err != nil {
		panic(err)
	}

	pretty.Println(spaces)

	for _, space := range spaces {
		page, err := c.GetPage(space.HomepageID)
		if err != nil {
			panic(err)
		}

		pretty.Println(page)
	}
}
