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
	//data,err := httpgetter.Get(v)
	//k := bytes.NewReader(data.Bytes())
	// chart, err := chartutil.LoadArchive(v)
	// now := timeconv.Now()
	// // test Render
	// renderOpts := renderutil.Options{
	// 	ReleaseOptions: chartutil.ReleaseOptions{
	// 		Name:      "test_release",
	// 		IsInstall: true,
	// 		IsUpgrade: false,
	// 		Time:      now,
	// 		Namespace: "defult",
	// 	},
	// 	KubeVersion: "1.11.5",
	// }

	// //config := &helmchart.Config{Raw: string(chart.Values.Raw), Values: map[string]*helmchart.Value{}}

	// renderedTemplates, err := renderutil.Render(chart, config, renderOpts)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// listManifests := manifest.SplitManifests(renderedTemplates)
	// for _, manifest := range listManifests {
	// 	fmt.Println(chartutil.ToYaml(manifest))
	// }
	// c.String(http.StatusOK, "yaml已打印")
}

// Manifest defined
func (controller RemoteHelm) Manifest(c *gin.Context) {
	//var vv = `G:\d.json`
	// xx, err := chartutil.LoadChartfile("./d.json")
	// if err != nil {
	// 	c.Error(err)
	// }
	// fmt.Println(xx);
	// ohttpgetter, err := hgetter.NewHTTPGetter(xx.String(), "", "", "")

	// if err != nil {
	// 	c.Error(err)
	// }

	// data, err := ohttpgetter.Get(xx.String())

	// r := bytes.NewReader(data.Bytes())

	tobeconverted, err := ioutil.ReadFile("./d.json")

	// Declared an empty map interface
	var result map[string]interface{}

	// Unmarshal or Decode the JSON to the interface.
	chart := json.Unmarshal([]byte(tobeconverted), &result)

	if err != nil {
		c.Error(err)
	}

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
	config := &helmchart.Config{Raw: string(chart.Values.Raw), Values: map[string]*helmchart.Value{}}
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
