package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/renstrom/fuzzysearch/fuzzy"

	"github.com/scaleway/devhub/pkg/manifest"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

var cache Cache

type Cache struct {
	Mapping struct {
		Images []ImageMapping `json:"mapped_images"`
	} `json:"mapping"`
	Manifest *scwManifest.Manifest `json:"manifest"`
	Api      struct {
		Images      *[]api.ScalewayImage      `json:"api_images"`
		Bootscripts *[]api.ScalewayBootscript `json:"api_bootscripts"`
	} `json:"api"`
}

func NewCache() Cache {
	return Cache{}
}

func (c *Cache) GetImageByName(name string) []*ImageMapping {
	found := []*ImageMapping{}
	for idx, image := range c.Mapping.Images {
		if image.MatchName(name) {
			fmt.Println(image, name)
			found = append(found, &c.Mapping.Images[idx])
		}
	}
	return found
}

func (c *Cache) MapImages() {
	// FIXME: add mutex
	if c.Manifest == nil || c.Api.Images == nil {
		return
	}

	c.Mapping.Images = make([]ImageMapping, 0)

	logrus.Infof("Mapping images")
	for _, manifestImage := range c.Manifest.Images {
		imageMapping := ImageMapping{
			ManifestName: manifestImage.Name,
		}
		manifestImageName := ImageCodeName(manifestImage.Name)
		for _, apiImage := range *c.Api.Images {
			apiImageName := ImageCodeName(apiImage.Name)
			if rankMatch := fuzzy.RankMatch(manifestImageName, apiImageName); rankMatch > -1 {
				imageMapping.ApiUUID = apiImage.Identifier
				imageMapping.RankMatch = rankMatch
				imageMapping.Found++
			}
		}
		c.Mapping.Images = append(c.Mapping.Images, imageMapping)
	}
	logrus.Infof("Images mapped")
}

type ImageMapping struct {
	ApiUUID      string
	ManifestName string
	RankMatch    int
	Found        int
}

func (i *ImageMapping) MatchName(input string) bool {
	if input == i.ApiUUID {
		return true
	}
	if fuzzy.RankMatch(i.ManifestName, input) > -1 {
		return true
	}
	return false
}

func ImageCodeName(inputName string) string {
	name := strings.ToLower(inputName)
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = regexp.MustCompile(`--+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

func main() {
	router := gin.Default()

	router.LoadHTMLGlob("templates/*")

	router.GET("/", indexEndpoint)

	router.GET("/images", imagesEndpoint)
	router.GET("/images/:name", imageEndpoint)
	router.GET("/images/:name/dockerfile", imageDockerfileEndpoint)

	router.GET("/cache", cacheEndpoint)

	// router.GET("/images/:name/new", newServerEndpoint)
	// router.GET("/images/:name/badge", imageBadgeEndpoint)

	Api, err := api.NewScalewayAPI("https://api.scaleway.com", "", os.Getenv("SCALEWAY_ORGANIZATION"), os.Getenv("SCALEWAY_TOKEN"))
	if err != nil {
		logrus.Fatalf("Failed to initialize Scaleway Api: %v", err)
	}

	go updateManifestCron(&cache)
	go updateScwApiImages(Api, &cache)
	// go updateScwApiBootscripts(Api, &cache)

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
	images := cache.GetImageByName(name)
	switch len(images) {
	case 0:
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No such image",
		})
	case 1:
		c.JSON(http.StatusOK, gin.H{
			"image": images[0],
		})
	default:
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Too much images are matching your request",
			"images": images,
		})
	}
}

func updateScwApiImages(Api *api.ScalewayAPI, cache *Cache) {
	for {
		logrus.Infof("Fetching images from the Api...")
		images, err := Api.GetImages()
		if err != nil {
			logrus.Errorf("Failed to retrieve images list from the Api: %v", err)
		} else {
			cache.Api.Images = images
			logrus.Infof("Images fetched: %d images", len(*images))
			cache.MapImages()
		}
		time.Sleep(5 * time.Minute)
	}
}

func updateScwApiBootscripts(Api *api.ScalewayAPI, cache *Cache) {
	for {
		logrus.Infof("Fetching bootscripts from the Api...")
		bootscripts, err := Api.GetBootscripts()
		if err != nil {
			logrus.Errorf("Failed to retrieve bootscripts list from the Api: %v", err)
		} else {
			cache.Api.Bootscripts = bootscripts
			logrus.Infof("Bootscripts fetched: %d bootscripts", len(*bootscripts))
		}
		time.Sleep(5 * time.Minute)
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
			logrus.Infof("Manifest fetched: %d images", len(manifest.Images))
			cache.MapImages()
		}
		time.Sleep(5 * time.Minute)
	}
}
