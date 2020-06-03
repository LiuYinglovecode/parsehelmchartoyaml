package main

import (
	"basic-gin2/pkg/util/flat"
	"bytes"
	"encoding/json"
	"log"
	"regexp"
	"io"
	"net/http"
	"net/url"
	"strings"
	"github.com/gin-gonic/gin"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/timeconv"
	"fmt"
	"path/filepath"
	"github.com/ghodss/yaml"
	hgetter "k8s.io/helm/pkg/getter"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
	"text/template"
	"github.com/Masterminds/sprig"
	util "k8s.io/helm/pkg/releaseutil"
	tversion "k8s.io/helm/pkg/version"
)

type Link struct {
	URL string
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

// GetValues defined
func GetValues(c *gin.Context) {
	var link Link
	if c.ShouldBind(&link) == nil {
		log.Println(link.URL)
	}
	u, err := url.Parse(link.URL)
	//for debug:fmt.Print("the u string is: s%",u)
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
	vls := chartutil.FromYaml(chart.Values.Raw)
	f, err := flat.Flatten(vls, nil)
	if err != nil {
		c.Error(err)
	}
	v, err := json.MarshalIndent(f, " ", "\t")
	if err != nil {
		c.Error(err)
	}
	//fmt.Println(string(v))
	fmt.Println("json printed successful!")
	//c.String(http.StatusOK, "ok")
	c.String(http.StatusOK, string(v))
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
			return []byte{}, fmt.Errorf("failed to parse %T: %s", filePath, err)
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


// Manifest defined
func Manifest(c *gin.Context) {
	var t templateCmd
	//var link Link
	//Set up engine.
	renderer := engine.New()

	caps := &chartutil.Capabilities{
		APIVersions:   chartutil.DefaultVersionSet,
		KubeVersion:   chartutil.DefaultKubeVersion,
		TillerVersion: tversion.GetVersionProto(),
	}

	// Check chart requirements to make sure all dependencies are present in /charts
	// l, err := chartutil.Load("G:/redis-9.3.0.tgz")
	
	id := c.Query("chart")
	u, err := url.Parse(id)
	//fmt.Print("the u string is:\n",u)
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

	chart1, err := chartutil.LoadArchive(r)

	if err != nil {
		c.Error(err)
	}

	l := chart1
	if err != nil {
		c.Error(err)
	} 
	//fmt.Print("for debug, this is what loadarchive function output:\n",l)

	
	message := c.PostForm("message")
	//var jsonStr = []byte(message)
	fmt.Printf(message)

	// Declared an empty map interface
	var result map[string]*helmchart.Value

	// // Unmarshal or Decode the JSON to the interface.
	json.Unmarshal([]byte(message), &result)
	// get combined values and create config
	rawVals, err := vals(t.valueFiles, t.values, t.stringValues)
	if err != nil {
		c.Error(err)
	}
	fmt.Println(rawVals)

	config := &helmchart.Config{Raw: string(rawVals), Values: result}

	options := chartutil.ReleaseOptions{
		Name:      t.releaseName,
		Time:      timeconv.Now(),
		Namespace: t.namespace,
	}
	vals, err := chartutil.ToRenderValuesCaps(l, config, options, caps)
	if err != nil {
		c.Error(err)
	}

	out, err := renderer.Render(l, vals)
	if err != nil {
		c.Error(err)
	}

	listManifests := []tiller.Manifest{}

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

	for _, m := range tiller.SortByKind(listManifests) {
		data := m.Content
		b := filepath.Base(m.Name)
		if !t.showNotes && b == "NOTES.txt" {
			continue
		}
		if strings.HasPrefix(b, "_") {
			continue
		}
		c.String(http.StatusOK, string("---\n# Source: "))
		c.String(http.StatusOK, "%s\n", m.Name)
		c.String(http.StatusOK, string(data))
		//fmt.Printf("---\n# Source: %s\n", m.Name)
		//fmt.Println(data)
	}

	// fmt.Println(listManifests)
	//c.String(http.StatusOK, "yaml已打印")
	fmt.Println("yaml printed successful!")
	c.JSON(200, gin.H{
		"status":  "posted",
		"message": message,
	})
	//c.String(http.StatusOK, string(v))
}
