package gzip

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type gzipHandler struct {
	*Options
	gzPool sync.Pool
}

func newGzipHandler() *gzipHandler {
	var gzPool sync.Pool
	gzPool.New = func() interface{} {
		gz, err := gzip.NewWriterLevel(ioutil.Discard, gzip.DefaultCompression)
		if err != nil {
			panic(err)
		}
		return gz
	}
	handler := &gzipHandler{
		Options: DefaultOptions,
		gzPool:  gzPool,
	}
	for _, setter := range []Option{WithExcludedPaths([]string{"/api/"})}{
		setter(handler.Options)
	}
	return handler
}

func (g *gzipHandler) Handle(c *gin.Context) {
	if fn := g.DecompressFn; fn != nil && c.Request.Header.Get("Content-Encoding") == "gzip" {
		fn(c)
	}

	if !g.shouldCompress(c.Request) {
		return
	}

	gz := g.gzPool.Get().(*gzip.Writer)
	defer g.gzPool.Put(gz)
	defer gz.Reset(ioutil.Discard)
	gz.Reset(c.Writer)

	if strings.Contains(c.Request.URL.Path,"/static/js/main.") && strings.HasSuffix(c.Request.URL.Path,".chunk.js") {
	    c.Header("Content-Encoding", "gzip")
	    c.Header("Vary", "Accept-Encoding")
	    c.Writer = &gzipWriter{c.Writer, gz}
	    defer func() {
	        gz.Close()
	        c.Header("Content-Length", fmt.Sprint(c.Writer.Size()))
	    }()
	    c.Next()
	    return
	}

	c.Header("Content-Encoding", "gzip")
	c.Header("Vary", "Accept-Encoding")
	c.Writer = &gzipWriter{c.Writer, gz}
	defer func() {
		gz.Close()
		c.Header("Content-Length", fmt.Sprint(c.Writer.Size()))
	}()
	c.Next()
}

func (g *gzipHandler) shouldCompress(req *http.Request) bool {
	if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") ||
		strings.Contains(req.Header.Get("Connection"), "Upgrade") ||
		strings.Contains(req.Header.Get("Content-Type"), "text/event-stream") {

		return false
	}

	extension := filepath.Ext(req.URL.Path)
	if g.ExcludedExtensions.Contains(extension) {
		return false
	}

	if g.ExcludedPaths.Contains(req.URL.Path) {
		return false
	}
	if g.ExcludedPathesRegexs.Contains(req.URL.Path) {
		return false
	}

	return true
}
