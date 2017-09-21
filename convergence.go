package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pressly/chi"
	"github.com/unrolled/render"
)

type Convergence struct {
	HomeSpaceKey  string
	HomePageTitle string

	confluence *Confluence
	proxy      http.Handler
	router     *chi.Mux
	render     *render.Render
}

func NewConvergence(confluence *Confluence, homeSpaceKey, homePageTitle string) *Convergence {
	return &Convergence{
		HomeSpaceKey:  homeSpaceKey,
		HomePageTitle: homePageTitle,

		confluence: confluence,
		proxy:      confluence.Proxy(),
		router:     chi.NewRouter(),
		render: render.New(render.Options{
			Extensions: []string{".html"},
			Layout:     "layout",
		}),
	}
}

func (c *Convergence) Run() {
	c.router.Use(c.proxyMiddleware)

	c.router.Get("/", c.viewRoot)
	c.router.Get("/:key", c.viewSpace)
	c.router.Get("/:key/:id/:title", c.viewPage)
	c.router.Get("/reset", c.handleReset)
	c.router.FileServer("/assets", http.Dir("./assets"))

	c.router.NotFound(c.handleNotFound)

	fmt.Printf("Running on 0.0.0.0:%s...\n", os.Getenv("PORT"))
	http.ListenAndServe(":"+os.Getenv("PORT"), c.router)
}

func (c *Convergence) viewRoot(w http.ResponseWriter, r *http.Request) {
	var err error
	var page *Page

	if _, err := strconv.Atoi(c.HomePageTitle); err == nil {
		page, err = c.confluence.GetPageByID(c.HomeSpaceKey, c.HomePageTitle)
		if err != nil {
			c.showError(w, err)
			return
		}
	}

	if page == nil {
		page, err = c.confluence.GetPageByTitle(c.HomeSpaceKey, c.HomePageTitle)
		if err != nil {
			c.showError(w, err)
			return
		}
	}

	c.render.HTML(w, http.StatusOK, "index", map[string]interface{}{
		"Title": page.Title,
		"Body":  c.processBody(page.Body, c.HomeSpaceKey),
	})
}

func (c *Convergence) viewSpace(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	space, err := c.confluence.GetSpace(key)
	if err != nil {
		c.showError(w, err)
		return
	}

	c.render.HTML(w, http.StatusOK, "page", map[string]interface{}{
		"Title": space.Name,
		"Body":  c.processBody(space.Homepage.Body, key),
		"Index": key,
		"Space": space.Name,
	})
}

func (c *Convergence) viewPage(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	id := chi.URLParam(r, "id")
	//title := chi.URLParam(r, "title")

	var err error
	var page *Page

	space, err := c.confluence.GetSpace(key)
	if err != nil {
		c.showError(w, err)
		return
	}

	page, err = c.confluence.GetPageByID(key, id)
	if err != nil {
		c.showError(w, err)
		return
	}

	c.render.HTML(w, http.StatusOK, "page", map[string]interface{}{
		"Title": page.Title,
		"Body":  c.processBody(page.Body, key),
		"Index": key,
		"Space": space.Name,
	})
}

func (c *Convergence) handleReset(w http.ResponseWriter, r *http.Request) {
	c.confluence.Reset()

	referrer := r.Referer()
	if len(referrer) <= 0 {
		referrer = "/"
	}

	http.Redirect(w, r, referrer, http.StatusTemporaryRedirect)
}

func (c *Convergence) handleNotFound(w http.ResponseWriter, r *http.Request) {
	c.showNotFound(w)
}

func (c *Convergence) proxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// proxy request if begins with /wiki
		if strings.HasPrefix(r.URL.Path, "/wiki") {
			c.proxy.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *Convergence) showError(w http.ResponseWriter, err error) {
	fmt.Printf("Error: %s\n", err.Error())

	if err == ErrNotFound {
		c.showNotFound(w)
		return
	}

	c.render.HTML(w, http.StatusInternalServerError, "503", map[string]interface{}{
		"Title": "Internal Server Error",
	})
}

func (c *Convergence) showNotFound(w http.ResponseWriter) {
	c.render.HTML(w, http.StatusNotFound, "404", map[string]interface{}{
		"Title": "Not Found",
	})
}

var linkRegex = regexp.MustCompile(`"/wiki/spaces/([A-z0-9]+)/pages/([0-9]+)/?(\S*)"`)

func (c *Convergence) processBody(body string, key string) template.HTML {
	for _, match := range linkRegex.FindAllStringSubmatch(body, -1) {
		body = strings.Replace(body, match[0], `"/`+key+`/`+match[2]+`/`+match[3]+`"`, 1)
	}

	body = strings.Replace(body, "/wiki/display/", "/", -1)
	body = strings.Replace(body, "/wiki/spaces/", "/", -1)

	return template.HTML(body)
}
