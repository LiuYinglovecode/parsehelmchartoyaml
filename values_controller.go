package main

import (
	"basic-gin2/pkg/util/flat"
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"k8s.io/helm/pkg/chartutil"

	//"strconv"
	"fmt"

	hgetter "k8s.io/helm/pkg/getter"
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

	// b, err := json.MarshalIndent(chart.Metadata, "", "\t")
	// if err != nil {
	// 	c.Error(err)
	// }
	// fmt.Println(string(b))

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
	//fmt.Println(string(v))
	// m := pretty.Pretty(v)
	//m := "[\n" + (v) + "]"
	fmt.Println(string(v))
	c.String(http.StatusOK, "json已打印！")
}

// func (controller RemoteHelm) Manifest(c *gin.Context) {
// 	c.String(http.StatusOK, "yaml已打印")
// 	now := timeconv.Now()
// 	// test Render
// 	renderOpts := renderutil.Options{
// 		ReleaseOptions: chartutil.ReleaseOptions{
// 			Name:      "test_release",
// 			IsInstall: true,
// 			IsUpgrade: false,
// 			Time:      now,
// 			Namespace: "defult",
// 		},
// 		KubeVersion: "1.11.5",
// 	}
// 	config := &helmchart.Config{Raw: string(chart.Values.Raw), Values: map[string]*helmchart.Value{}}
// 	renderedTemplates, err := renderutil.Render(chart, config, renderOpts)
// 	//if err != nil {
// 	//	c.Fatal(err)
// 	//}
// 	listManifests := manifest.SplitManifests(renderedTemplates)
// 	for _, manifest := range listManifests {
// 		fmt.Println(chartutil.ToYaml(manifest))
// 	}
// }
