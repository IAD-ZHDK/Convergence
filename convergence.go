package main

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var linkRegex = regexp.MustCompile(`"/wiki/pages/viewpage\.action\?pageId=(\d+)"`)

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
	router.GET("/download/:type/:id/:file", c.download)
	router.GET("/reset", c.reset)

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
	key := ctx.Param("key")

	space, err := c.Confluence.GetSpace(key)
	if err != nil {
		c.error(ctx, err)
		return
	}

	ctx.HTML(http.StatusOK, "page.html", gin.H{
		"Title": space.Name,
		"Body":  c.processBody(space.Homepage.Body, key),
		"Index": key,
	})
}

func (c *Convergence) page(ctx *gin.Context) {
	key := ctx.Param("key")
	title := ctx.Param("title")

	var err error
	var page *Page
	if _, err := strconv.Atoi(title); err == nil {
		page, err = c.Confluence.GetPageByID(key, title)
		if err != nil {
			c.error(ctx, err)
			return
		}
	}

	if page == nil {
		page, err = c.Confluence.GetPage(key, title)
		if err != nil {
			c.error(ctx, err)
			return
		}
	}

	ctx.HTML(http.StatusOK, "page.html", gin.H{
		"Title": page.Title,
		"Body":  c.processBody(page.Body, key),
		"Index": key,
	})
}

func (c *Convergence) download(ctx *gin.Context) {
	typ := ctx.Param("type")
	id := ctx.Param("id")
	file := ctx.Param("file")
	version := ctx.Query("version")
	date := ctx.Query("modificationDate")
	api := ctx.Query("api")

	download, err := c.Confluence.GetDownload(typ, id, file, version, date, api)
	if err != nil {
		c.error(ctx, err)
		return
	}

	ctx.Data(200, download.ContentType, download.Data)
}

func (c *Convergence) reset(ctx *gin.Context) {
	c.Confluence.Reset()

	referer := ctx.Request.Referer()
	if len(referer) <= 0 {
		referer = "/"
	}

	ctx.Redirect(http.StatusTemporaryRedirect, referer)
}

func (c *Convergence) processBody(body string, key string) template.HTML {
	body = strings.Replace(body, "/wiki/display/", "/page/", -1)
	body = strings.Replace(body, "/wiki/download/", "/download/", -1)

	for _, match := range linkRegex.FindAllStringSubmatch(body, -1) {
		body = strings.Replace(body, match[0], `"/page/`+key+`/`+match[1]+`"`, 1)
	}

	return template.HTML(body)
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
