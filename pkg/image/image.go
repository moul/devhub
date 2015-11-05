package scwImage

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
)

type Image struct {
	Name   string
	Tags   []string
	Repo   string
	Path   string
	Branch string
}

func (i *Image) RepoHost() string {
	return strings.Split(i.Repo, "/")[0]
}

func (i *Image) RepoPath() string {
	return strings.Join(strings.Split(i.Repo, "/")[1:], "/")
}

func (i *Image) RawContentUrl(path string) (string, error) {
	switch i.RepoHost() {
	case "github.com":
		prefix := i.Path
		if prefix == "." {
			prefix = ""
		}
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", i.RepoPath(), i.Branch, prefix, path), nil
	}
	return "", fmt.Errorf("Unhandled repository service: %q", i.RepoHost())
}

func (i *Image) FullName() string {
	return fmt.Sprintf("%s:%s", i.Name, i.Tags[0])
}

func (i *Image) GetDockerfile() (string, error) {
	url, err := i.RawContentUrl("Dockerfile")
	if err != nil {
		return "", err
	}
	logrus.Infof("Fetching Dockerfile for %s (%s)", i.FullName(), url)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	return string(body), nil
}
