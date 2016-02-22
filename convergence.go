package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Convergence struct {
	Confluence *Confluence
}

func NewConvergence() *Convergence {
	return &Convergence{}
}

func (c *Convergence) Run() error {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.Static("/assets", "./assets")

	router.GET("/", c.root)
	router.GET("/page/:key", c.space)
	router.GET("/page/:key/:title", c.page)

	router.NoRoute(c.notFound)

	return router.Run()
}

func (c *Convergence) root(ctx *gin.Context) {
	spaces, err := c.Confluence.GetSpaces()
	if err != nil {
		c.error(ctx, err)
		return
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"Title":  "IAD Wiki",
		"Spaces": spaces,
	})
}

func (c *Convergence) space(ctx *gin.Context) {
	key, _ := ctx.Params.Get("key")

	space, err := c.Confluence.GetSpace(key)
	if err != nil {
		c.error(ctx, err)
		return
	}

	page, err := c.Confluence.GetPageById(key, space.HomepageID)
	if err != nil {
		c.error(ctx, err)
		return
	}

	ctx.HTML(http.StatusOK, "page.html", gin.H{
		"Title": page.Title,
		"Page":  page,
		"Index": key,
	})
}

func (c *Convergence) page(ctx *gin.Context) {
	key, _ := ctx.Params.Get("key")
	title, _ := ctx.Params.Get("title")

	page, err := c.Confluence.GetPageByTitle(key, title)
	if err != nil {
		c.error(ctx, err)
		return
	}

	ctx.HTML(http.StatusOK, "page.html", gin.H{
		"Title": page.Title,
		"Page":  page,
		"Index": key,
	})
}

func (c *Convergence) error(ctx *gin.Context, err error) {
	fmt.Printf("Error: %s\n", err.Error())

	if err == ErrNotFound {
		ctx.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Not Found",
		})
		return
	}

	ctx.HTML(http.StatusInternalServerError, "503.html", gin.H{
		"Title": "Internal Server Error",
	})
}

func (c *Convergence) notFound(ctx *gin.Context) {
	ctx.HTML(http.StatusNotFound, "404.html", gin.H{
		"Title": "Not Found",
	})
}
