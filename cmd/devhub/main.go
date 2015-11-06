package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/scaleway/devhub/pkg/manifest"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

type Cache struct {
	Manifest       *scwManifest.Manifest     `json:"manifest"`
	APIImages      *[]api.ScalewayImage      `json:"api_images"`
	APIBootscripts *[]api.ScalewayBootscript `json:"api_bootscripts"`
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

	router.GET("/cache", cacheEndpoint)

	// router.GET("/images/:name/new", newServerEndpoint)
	// router.GET("/images/:name/badge", imageBadgeEndpoint)

	API, err := api.NewScalewayAPI("https://api.scaleway.com", "", os.Getenv("SCALEWAY_ORGANIZATION"), os.Getenv("SCALEWAY_TOKEN"))
	if err != nil {
		logrus.Fatalf("Failed to initialize Scaleway API: %v", err)
	}

	go updateManifestCron(&cache)
	go updateScwAPIImages(API, &cache)
	go updateScwAPIBootscripts(API, &cache)

	router.Run(":4242")
}

func indexEndpoint(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{})
}

func cacheEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"cache": cache,
	})
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

func updateScwAPIImages(API *api.ScalewayAPI, cache *Cache) {
	for {
		images, err := API.GetImages()
		if err == nil {
			cache.APIImages = images
		} else {
			logrus.Errorf("Failed to retrieve images list from the API: %v", err)
		}
		time.Sleep(3 * time.Minute)
	}
}

func updateScwAPIBootscripts(API *api.ScalewayAPI, cache *Cache) {
	for {
		bootscripts, err := API.GetBootscripts()
		if err == nil {
			cache.APIBootscripts = bootscripts
		} else {
			logrus.Errorf("Failed to retrieve bootscripts list from the API: %v", err)
		}
		time.Sleep(3 * time.Minute)
	}
}

func updateManifestCron(cache *Cache) {
	for {
		logrus.Infof("Fetching manifest...")
		manifest, err := scwManifest.GetManifest()
		if err != nil {
			logrus.Errorf("Cannot get manifest: %v", err)
		} else {
			cache.Manifest = manifest
		}
		logrus.Infof("Manifest fetched: %d images", len(manifest.Images))
		time.Sleep(3 * time.Minute)
	}
}
