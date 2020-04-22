package main

import (
	"basic-gin2/pkg/util/flat"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"k8s.io/helm/pkg/chartutil"

	"fmt"

	"k8s.io/helm/pkg/timeconv"

	hgetter "k8s.io/helm/pkg/getter"
	//"fmt"
	"k8s.io/helm/pkg/manifest"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
)

var repoURL = "https://kubernetes-charts.storage.googleapis.com/redis-9.3.0.tgz"

// RemoteHelm defined
type RemoteHelm struct{}

// NewRemoteHelm defined
func NewRemoteHelm() RemoteHelm {
	return RemoteHelm{}
}

// Values defined
func (controller RemoteHelm) Values(c *gin.Context) {
	u, err := url.Parse(repoURL)
	if err != nil {
		c.Error(err)
	}
	httpgetter, err := hgetter.NewHTTPGetter(u.String(), "", "", "")

	if err != nil {
		c.Error(err)
	}

	data, err := httpgetter.Get(u.String())

	if err != nil {
		c.Error(err)
	}

	r := bytes.NewReader(data.Bytes())

	chart, err := chartutil.LoadArchive(r)

	if err != nil {
		c.Error(err)
	}

	// print values
	vl := chartutil.FromYaml(chart.Values.Raw)
	f, err := flat.Flatten(vl, nil)
	if err != nil {
		c.Error(err)
	}
	v, err := json.MarshalIndent(f, " ", "\t")
	if err != nil {
		c.Error(err)
	}
	fmt.Println(string(v))
	c.String(http.StatusOK, "json已打印！")

}


// Manifest defined
func (controller RemoteHelm) Manifest(c *gin.Context) {
		u, err := url.Parse(repoURL)
	if err != nil {
		c.Error(err)
	}
	httpgetter, err := hgetter.NewHTTPGetter(u.String(), "", "", "")

	if err != nil {
		c.Error(err)
	}

	data, err := httpgetter.Get(u.String())

	if err != nil {
		c.Error(err)
	}

	r := bytes.NewReader(data.Bytes())

	chart, err := chartutil.LoadArchive(r)

	//accept the json paramaters
	tobeconverted, _ := ioutil.ReadFile("./d.json")

	// Declared an empty map interface
	var result map[string]*helmchart.Value
 
	// Unmarshal or Decode the JSON to the interface.
	_ = json.Unmarshal([]byte(tobeconverted), &result)

	now := timeconv.Now()
	// test Render
	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "test_release",
			IsInstall: true,
			IsUpgrade: false,
			Time:      now,
			Namespace: "defult",
		},
		KubeVersion: "1.11.5",
	}
	config := &helmchart.Config{Raw: string(chart.Values.Raw), Values: result}
	renderedTemplates, err := renderutil.Render(chart, config, renderOpts)
	if err != nil {
		fmt.Println(err)
	}
	listManifests := manifest.SplitManifests(renderedTemplates)
	for _, manifest := range listManifests {
		fmt.Println(chartutil.ToYaml(manifest))
	}
	c.String(http.StatusOK, "yaml已打印")
}
