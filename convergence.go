package main

import (
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

	router.GET("/", c.root)
	router.GET("/:key", c.space)
	router.GET("/:key/:title", c.page)

	return router.Run()
}

func (c *Convergence) root(ctx *gin.Context) {
	spaces, err := c.Confluence.GetSpaces()
	if err != nil {
		ctx.AbortWithError(503, err)
		return
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"Spaces": spaces,
	})
}

func (c *Convergence) space(ctx *gin.Context) {
	key, _ := ctx.Params.Get("key")

	space, err := c.Confluence.GetSpace(key)
	if err != nil {
		ctx.AbortWithError(503, err)
		return
	}

	page, err := c.Confluence.GetPageById(key, space.HomepageID)
	if err != nil {
		ctx.AbortWithError(503, err)
		return
	}

	ctx.HTML(http.StatusOK, "page.html", page)
}

func (c *Convergence) page(ctx *gin.Context) {
	key, _ := ctx.Params.Get("key")
	title, _ := ctx.Params.Get("title")

	page, err := c.Confluence.GetPageByTitle(key, title)
	if err != nil {
		ctx.AbortWithError(503, err)
		return
	}

	ctx.HTML(http.StatusOK, "page.html", page)
}
