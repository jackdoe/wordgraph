package main

import (
	"encoding/gob"
	"flag"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strings"
)

type QueryInput struct {
	Text       string  `form:"text" json:"text" binding:"required"`
	Percentile float64 `form:"percentile" json:"percentile"`
}

func init() {
	gob.Register(Index{})
}

func main() {
	var indexFile = flag.String("indexFile", "./computed_index/index.gob.gz", "load the index from")
	var parseRoot = flag.String("parseRoot", "", "parse all txt files from this directory")
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	idx := NewIndex()
	if *parseRoot != "" {
		for _, root := range strings.Split(*parseRoot, ",") {
			idx.parse(root)
			runtime.GC()
		}
		idx.save(*indexFile)
		runtime.GC()
	}
	idx.load(*indexFile)

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(gin.Recovery())
	r.Static("/assets", "./assets")
	r.LoadHTMLGlob("./t/*.tmpl")

	r.POST("/query", func(c *gin.Context) {
		var json QueryInput
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		out := idx.query(json.Text, json.Percentile)
		c.JSON(http.StatusOK, out)
	})

	r.POST("/flow", func(c *gin.Context) {
		var json QueryInput
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		out := idx.flow(json.Text)
		c.JSON(http.StatusOK, out)
	})

	r.GET("/test", func(c *gin.Context) {
		c.HTML(http.StatusOK, "test.tmpl", map[string]interface{}{})
	})

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", map[string]interface{}{})
	})

	r.Run()

}
