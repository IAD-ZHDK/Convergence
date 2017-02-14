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
	confluence *Confluence
}

func NewConvergence(confluence *Confluence) *Convergence {
	return &Convergence{
		confluence: confluence,
	}
}

func (c *Convergence) Run() error {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.Static("/assets", "./assets")

	router.GET("/", c.viewRoot)
	router.GET("/page/:key", c.viewSpace)
	router.GET("/page/:key/:title", c.viewPage)
	router.GET("/download/:type/:id/:file", c.handleDownload)
	router.GET("/reset", c.handleReset)

	router.NoRoute(c.showNotFound)

	return router.Run()
}

func (c *Convergence) viewRoot(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"Title": "IAD Wiki",
	})
}

func (c *Convergence) viewSpace(ctx *gin.Context) {
	key := ctx.Param("key")

	space, err := c.confluence.GetSpace(key)
	if err != nil {
		c.showError(ctx, err)
		return
	}

	ctx.HTML(http.StatusOK, "page.html", gin.H{
		"Title": space.Name,
		"Body":  c.processBody(space.Homepage.Body, key),
		"Index": key,
		"Space": space.Name,
	})
}

func (c *Convergence) viewPage(ctx *gin.Context) {
	key := ctx.Param("key")
	title := ctx.Param("title")

	var err error
	var page *Page

	space, err := c.confluence.GetSpace(key)
	if err != nil {
		c.showError(ctx, err)
		return
	}

	if _, err := strconv.Atoi(title); err == nil {
		page, err = c.confluence.GetPageByID(key, title)
		if err != nil {
			c.showError(ctx, err)
			return
		}
	}

	if page == nil {
		page, err = c.confluence.GetPageByTitle(key, title)
		if err != nil {
			c.showError(ctx, err)
			return
		}
	}

	ctx.HTML(http.StatusOK, "page.html", gin.H{
		"Title": page.Title,
		"Body":  c.processBody(page.Body, key),
		"Index": key,
		"Space": space.Name,
	})
}

func (c *Convergence) handleDownload(ctx *gin.Context) {
	typ := ctx.Param("type")
	id := ctx.Param("id")
	file := ctx.Param("file")
	version := ctx.Query("version")
	date := ctx.Query("modificationDate")
	api := ctx.Query("api")

	download, err := c.confluence.GetDownload(typ, id, file, version, date, api)
	if err != nil {
		c.showError(ctx, err)
		return
	}

	ctx.Data(200, download.ContentType, download.Data)
}

func (c *Convergence) handleReset(ctx *gin.Context) {
	c.confluence.Reset()

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

func (c *Convergence) showError(ctx *gin.Context, err error) {
	fmt.Printf("Error: %s\n", err.Error())

	if err == errNotFound {
		c.showNotFound(ctx)
		return
	}

	ctx.HTML(http.StatusInternalServerError, "503.html", gin.H{
		"Title": "Internal Server Error",
	})
}

func (c *Convergence) showNotFound(ctx *gin.Context) {
	ctx.HTML(http.StatusNotFound, "404.html", gin.H{
		"Title": "Not Found",
	})
}
