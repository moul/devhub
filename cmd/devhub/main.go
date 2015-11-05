package main

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/scaleway/devhub/pkg/manifest"
)

func main() {
	logrus.Infof("Fetching manifest...")
	manifest, err := scwManifest.GetManifest()
	if err != nil {
		logrus.Fatalf("Cannot get manifest: %v", err)
	}
	logrus.Infof("Manifest fetched: %d images", len(manifest.Images))

	/*
		logrus.Infof("Initializing Docker client...")
		token := &oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")}
		ts := oauth2.StaticTokenSource(token)
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		client := github.NewClient(tc)
		logrus.Infof("Docker client initialized")
	*/

	for _, image := range manifest.Images {
		dockerfile, err := image.GetDockerfile()
		if err != nil {
			logrus.Errorf("Cannot get Dockerfile for %s:%s", image.Name, image.Tags)
		}

		fmt.Println(dockerfile)

		// break -> only 1 image for now
		break
	}
}
