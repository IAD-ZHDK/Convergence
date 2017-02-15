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
	proxy      http.Handler
}

func NewConvergence(confluence *Confluence) *Convergence {
	return &Convergence{
		confluence: confluence,
		proxy:      confluence.Proxy(),
	}
}

func (c *Convergence) Run() error {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.GET("/", c.viewRoot)
	router.GET("/page/:key", c.viewSpace)
	router.GET("/page/:key/:title", c.viewPage)
	router.Static("/assets", "./assets")
	router.GET("/reset", c.handleReset)
	router.NoRoute(c.handleProxy)

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

func (c *Convergence) handleReset(ctx *gin.Context) {
	c.confluence.Reset()

	referrer := ctx.Request.Referer()
	if len(referrer) <= 0 {
		referrer = "/"
	}

	ctx.Redirect(http.StatusTemporaryRedirect, referrer)
}

func (c *Convergence) handleProxy(ctx *gin.Context) {
	// proxy request if begins with /wiki
	if strings.HasPrefix(ctx.Request.URL.Path, "/wiki") {
		c.proxy.ServeHTTP(ctx.Writer, ctx.Request)
		return
	}

	c.showNotFound(ctx)
}

func (c *Convergence) processBody(body string, key string) template.HTML {
	body = strings.Replace(body, "/wiki/display/", "/page/", -1)

	for _, match := range linkRegex.FindAllStringSubmatch(body, -1) {
		body = strings.Replace(body, match[0], `"/page/`+key+`/`+match[1]+`"`, 1)
	}

	return template.HTML(body)
}

func (c *Convergence) showError(ctx *gin.Context, err error) {
	fmt.Printf("Error: %s\n", err.Error())

	if err == ErrNotFound {
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
