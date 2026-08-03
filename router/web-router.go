package router

import (
	"bytes"
	"embed"
	"html"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

const defaultSeoDescription = "A unified AI model hub for aggregation and distribution."

func seoOption(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(common.OptionMap[key])
}

func absoluteSeoURL(value string, c *gin.Context) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + "/" + strings.TrimLeft(value, "/")
}

func renderSeoIndex(indexPage []byte, c *gin.Context) []byte {
	title := seoOption("SeoTitle")
	if title == "" {
		title = common.SystemName
	}
	description := seoOption("SeoDescription")
	if description == "" {
		description = defaultSeoDescription
	}
	robots := seoOption("SeoRobots")
	if robots == "" {
		robots = "index,follow"
	}
	canonicalURL := seoOption("SeoCanonicalUrl")
	if canonicalURL == "" {
		canonicalURL = absoluteSeoURL("/", c)
	}
	values := map[string]string{
		"__SEO_TITLE__":         title,
		"__SEO_DESCRIPTION__":   description,
		"__SEO_KEYWORDS__":      seoOption("SeoKeywords"),
		"__SEO_CANONICAL_URL__": canonicalURL,
		"__SEO_OG_IMAGE__":      absoluteSeoURL(seoOption("SeoOgImage"), c),
		"__SEO_ROBOTS__":        robots,
	}
	page := indexPage
	for placeholder, value := range values {
		page = bytes.ReplaceAll(page, []byte(placeholder), []byte(html.EscapeString(value)))
	}
	return page
}

func serveRobots(c *gin.Context) {
	robots := seoOption("SeoRobots")
	disallow := strings.Contains(strings.ToLower(robots), "noindex")
	body := "User-agent: *\n"
	if disallow {
		body += "Disallow: /\n"
	} else {
		body += "Disallow:\n"
		canonicalURL := strings.TrimRight(seoOption("SeoCanonicalUrl"), "/")
		if canonicalURL != "" {
			body += "Sitemap: " + canonicalURL + "/sitemap.xml\n"
		}
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

func serveSitemap(c *gin.Context) {
	baseURL := strings.TrimRight(seoOption("SeoCanonicalUrl"), "/")
	if baseURL == "" {
		baseURL = absoluteSeoURL("/", c)
		baseURL = strings.TrimRight(baseURL, "/")
	}
	body := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>` + html.EscapeString(baseURL) + `</loc></url>
</urlset>`
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(body))
}

func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.GET("/robots.txt", serveRobots)
	router.GET("/sitemap.xml", serveSitemap)
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, "web/dist")))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if tryServeLocalStorageByRequestPath(c) {
			return
		}
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", renderSeoIndex(indexPage, c))
	})
}
