package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/scaleway/devhub/pkg/manifest"
)

type Cache struct {
	Manifest *scwManifest.Manifest
}

var cache Cache

func main() {
	router := gin.Default()

	router.LoadHTMLGlob("templates/*")

	router.GET("/", indexEndpoint)

	v1 := router.Group("/v1")
	{
		v1.GET("/images", imagesEndpoint)
		v1.GET("/images/:name", imageEndpoint)
		v1.GET("/images/:name/dockerfile", imageDockerfileEndpoint)
	}

	go updateManifestCron(&cache)

	router.Run(":4242")
}

func indexEndpoint(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{})
}

func imagesEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"images": cache.Manifest.Images,
	})
}

func imageDockerfileEndpoint(c *gin.Context) {
	name := c.Param("name")
	image := cache.Manifest.Images[name]
	dockerfile, err := image.GetDockerfile()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("%v", err),
		})
	}
	c.String(http.StatusOK, dockerfile)
}

func imageEndpoint(c *gin.Context) {
	name := c.Param("name")
	image := cache.Manifest.Images[name]
	c.JSON(http.StatusOK, gin.H{
		"image": image,
	})
}

func updateManifestCron(cache *Cache) {
	logrus.Infof("Fetching manifest...")
	manifest, err := scwManifest.GetManifest()
	if err != nil {
		logrus.Fatalf("Cannot get manifest: %v", err)
	}
	cache.Manifest = manifest
	logrus.Infof("Manifest fetched: %d images", len(manifest.Images))
	time.Sleep(30 * time.Second)
}
