package scwManifest

import (
	"bufio"
	"net/http"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/scaleway/devhub/pkg/image"
)

const ManifestURL = "https://raw.githubusercontent.com/scaleway/image-tools/master/public-images.manifest"

type Manifest struct {
	Images []scwImage.Image
}

func GetManifest() (*Manifest, error) {
	return GetManifestByURL(ManifestURL)
}

func GetManifestByURL(manifestURL string) (*Manifest, error) {
	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)

	re := regexp.MustCompile(`\ +`)

	manifest := Manifest{
		Images: make([]scwImage.Image, 0),
	}

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if len(line) < 1 || line[0] == '#' {
			continue
		}
		cols := re.Split(line, -1)
		if len(cols) < 4 {
			logrus.Warnf("Cannot parse manifest line %q: invalid amount of columns", line)
		}
		newEntry := scwImage.Image{
			Name:   cols[0],
			Tags:   strings.Split(cols[1], ","),
			Repo:   cols[2],
			Path:   cols[3],
			Branch: "master",
		}
		manifest.Images = append(manifest.Images, newEntry)
	}

	return &manifest, nil
}
