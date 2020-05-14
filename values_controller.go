package main

import (
	"basic-gin2/pkg/util/flat"
	"bytes"
	"encoding/json"
	//"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/helm/pkg/chartutil"

	//"k8s.io/helm/pkg/getter"
	"fmt"

	"k8s.io/helm/pkg/timeconv"
	//vals "github.com/helm/cmd/helm"
	"github.com/ghodss/yaml"
	//"path/filepath"

	hgetter "k8s.io/helm/pkg/getter"
	//"fmt"
	//"k8s.io/helm/pkg/manifest"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
	//"k8s.io/helm/pkg/renderutil"
	//"k8s.io/helm/pkg/strvals"
	"regexp"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/Masterminds/sprig"
	"k8s.io/helm/pkg/engine"
	util "k8s.io/helm/pkg/releaseutil"
	"k8s.io/helm/pkg/tiller"
	tversion "k8s.io/helm/pkg/version"
)

var repoURL = "https://kubernetes-charts.storage.googleapis.com/redis-9.3.0.tgz"

// RemoteHelm defined
type RemoteHelm struct{}

// NewRemoteHelm defined
func NewRemoteHelm() RemoteHelm {
	return RemoteHelm{}
}

type templateCmd struct {
	values       []string
	stringValues []string
	namespace    string
	valueFiles   []byte
	//chartPath    string
	out          io.Writer
	nameTemplate string
	showNotes    bool
	releaseName  string
	renderFiles  []string
	kubeVersion  string
	outputDir    string
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

type valueFiles []string

func (v *valueFiles) String() string {
	return fmt.Sprint(*v)
}

func (v *valueFiles) Type() string {
	return "valueFiles"
}

func (v *valueFiles) Set(value string) error {
	for _, filePath := range strings.Split(value, ",") {
		*v = append(*v, filePath)
	}
	return nil
}

// Merges source and destination map, preferring values from the source map
func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

// vals merges values from files specified via -f/--values and
func vals(valueFiles []byte, values []string, stringValues []string) ([]byte, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, filePath := range valueFiles {
		currentMap := map[string]interface{}{}

		var bytes []byte
		var err error
		// if strings.TrimSpace(filePath) == "-" {
		// 	bytes, err = ioutil.ReadAll(os.Stdin)
		// }

		if err != nil {
			return []byte{}, err
		}

		if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
			return []byte{}, fmt.Errorf("failed to parse %s: %s", filePath, err)
		}
		// Merge with the previous map
		base = mergeValues(base, currentMap)
	}
	return yaml.Marshal(base)
}

func generateName(nameTemplate string) (string, error) {
	t, err := template.New("name-template").Funcs(sprig.TxtFuncMap()).Parse(nameTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = t.Execute(&b, nil)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func checkDependencies(ch *helmchart.Chart, reqs *chartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}

// func (t *templateCmd) run() error {

func (controller RemoteHelm) Mani(c *gin.Context) {
	// verify specified templates exist relative to chart
	rf := []string{}
	//var err error
	var t templateCmd
	// 临时加的，先试试
	//accept the json paramaters
	tobeconverted, _ := ioutil.ReadFile("./d.json")

	var result map[string]*helmchart.Value

	// Unmarshal or Decode the JSON to the interface.
	json.Unmarshal([]byte(tobeconverted), &result)

	// get combined values and create config
	rawVals, err := vals(t.valueFiles, t.values, t.stringValues)
	if err != nil {
		c.Error(err)
	}
	config := &helmchart.Config{Raw: string(rawVals), Values: result}

	// If template is specified, try to run the template.
	if t.nameTemplate != "" {
		t.releaseName, err = generateName(t.nameTemplate)
		// if err != nil {
		// 	return err
		// }
		if err != nil {
		c.Error(err)
	}
	}

	var chartPath = "./redis-9.3.0.tgz"
	// Check chart requirements to make sure all dependencies are present in /charts
	l, err := chartutil.Load(chartPath)
	if err != nil {
		 c.Error(err)
	}

	if req, err := chartutil.LoadRequirements(l); err == nil {
		if err := checkDependencies(l, req); err != nil {
			 c.Error(err)
		}
	} else if err != chartutil.ErrRequirementsNotFound {
			c.Error(err)
	}
	options := chartutil.ReleaseOptions{
		Name:      t.releaseName,
		Time:      timeconv.Now(),
		Namespace: t.namespace,
	}

	err = chartutil.ProcessRequirementsEnabled(l, config)
	if err != nil {
		c.Error(err) 
	}

	err = chartutil.ProcessRequirementsImportValues(l)
	if err != nil {
		c.Error(err)
	}

	// Set up engine.
	renderer := engine.New()

	caps := &chartutil.Capabilities{
		APIVersions:   chartutil.DefaultVersionSet,
		KubeVersion:   chartutil.DefaultKubeVersion,
		TillerVersion: tversion.GetVersionProto(),
	}

	// kubernetes version
	kv, err := semver.NewVersion(t.kubeVersion)
	// if err != nil {
	// 	return fmt.Errorf("could not parse a kubernetes version: %v", err)
	// }
	if err != nil {
		c.Error(err)
	}
	caps.KubeVersion.Major = fmt.Sprint(kv.Major())
	caps.KubeVersion.Minor = fmt.Sprint(kv.Minor())
	caps.KubeVersion.GitVersion = fmt.Sprintf("v%d.%d.0", kv.Major(), kv.Minor())

	vals, err := chartutil.ToRenderValuesCaps(l, config, options, caps)
	// if err != nil {
	// 	return err
	// }
	if err != nil {
		c.Error(err)
	}

	out, err := renderer.Render(l, vals)
	listManifests := []tiller.Manifest{}
	// if err != nil {
	// 	return err
	// }
	if err != nil {
		c.Error(err)
	}

	// extract kind and name
	re := regexp.MustCompile("kind:(.*)\n")
	for k, v := range out {
		match := re.FindStringSubmatch(v)
		h := "Unknown"
		if len(match) == 2 {
			h = strings.TrimSpace(match[1])
		}
		m := tiller.Manifest{Name: k, Content: v, Head: &util.SimpleHead{Kind: h}}
		listManifests = append(listManifests, m)
	}
	in := func(needle string, haystack []string) bool {
		// make needle path absolute
		d := strings.Split(needle, string(os.PathSeparator))
		dd := d[1:]
		an := filepath.Join(chartPath, strings.Join(dd, string(os.PathSeparator)))

		for _, h := range haystack {
			if h == an {
				return true
			}
		}
		return false
	}
	// if settings.Debug {
	// 	rel := &release.Release{
	// 		Name:      t.releaseName,
	// 		Chart:     c,
	// 		Config:    config,
	// 		Version:   1,
	// 		Namespace: t.namespace,
	// 		Info:      &release.Info{LastDeployed: timeconv.Timestamp(time.Now())},
	// 	}
	// 	printRelease(os.Stdout, rel)
	// }

	for _, m := range tiller.SortByKind(listManifests) {
		if len(t.renderFiles) > 0 && !in(m.Name, rf) {
			continue
		}
		data := m.Content
		b := filepath.Base(m.Name)
		if !t.showNotes && b == "NOTES.txt" {
			continue
		}
		if strings.HasPrefix(b, "_") {
			continue
		}

		fmt.Printf("---\n# Source: %s\n", m.Name)
		fmt.Println(data)
	}
	//return err
}

// Manifest defined
// func (controller RemoteHelm) Manifest(c *gin.Context) {
// 	var t templateCmd 

// 	// Set up engine.
// 	renderer := engine.New()

// 	caps := &chartutil.Capabilities{
// 		APIVersions:   chartutil.DefaultVersionSet,
// 		KubeVersion:   chartutil.DefaultKubeVersion,
// 		TillerVersion: tversion.GetVersionProto(),
// 	}

// 	// Check chart requirements to make sure all dependencies are present in /charts
// 	l, err := chartutil.Load(t.chartPath)
// 	if err != nil {
// 		c.Error(err)
// 	}
// 	// u, err := url.Parse(repoURL)
// 	// if err != nil {
// 	// 	c.Error(err)
// 	// }
// 	// httpgetter, err := hgetter.NewHTTPGetter(u.String(), "", "", "")

// 	// if err != nil {
// 	// 	c.Error(err)
// 	// }

// 	// data, err := httpgetter.Get(u.String())

// 	// if err != nil {
// 	// 	c.Error(err)
// 	// }

// 	// r := bytes.NewReader(data.Bytes())

// 	//chart, err := chartutil.LoadArchive(r)

// 	//accept the json paramaters
// 	tobeconverted, _ := ioutil.ReadFile("./d.json")

// 	// Declared an empty map interface
// 	var result map[string]*helmchart.Value

// 	// Unmarshal or Decode the JSON to the interface.
// 	json.Unmarshal([]byte(tobeconverted), &result)
// 	// get combined values and create config
// 	rawVals, err := vals(t.valueFiles, t.values, t.stringValues)
// 	if err != nil {
// 		c.Error(err)
// 	}

// 	config := &helmchart.Config{Raw: string(rawVals), Values: result}

// 	options := chartutil.ReleaseOptions{
// 		Name:      t.releaseName,
// 		Time:      timeconv.Now(),
// 		Namespace: t.namespace,
// 	}
// 	vals, err := chartutil.ToRenderValuesCaps(l, config, options, caps)
// 	if err != nil {
// 		c.Error(err)
// 	}

// 	out, err := renderer.Render(l, vals)
// 	if err != nil {
// 		c.Error(err)
// 	}

// 	listManifests := []tiller.Manifest{}

// 	// extract kind and name
// 	re := regexp.MustCompile("kind:(.*)\n")
// 	for k, v := range out {
// 		match := re.FindStringSubmatch(v)
// 		h := "Unknown"
// 		if len(match) == 2 {
// 			h = strings.TrimSpace(match[1])
// 		}
// 		m := tiller.Manifest{Name: k, Content: v, Head: &util.SimpleHead{Kind: h}}
// 		listManifests = append(listManifests, m)
// 	}
// 	// If template is specified, try to run the template.
// 	if t.nameTemplate != "" {
// 		t.releaseName, _ = generateName(t.nameTemplate)

// 	}
// 	c.String(http.StatusOK, "yaml已打印")
 //}