/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"github.com/spf13/pflag"
	"k8s.io/klog"
)

type JobToCategoryData map[string]CategoryToMetricData

type CategoryToMetricData map[string]MetricToBuildData

type MetricToBuildData map[string]*BuildData

type BuildData struct {
	Builds  map[string][]DataItem `json:"builds"`
	Job     string                `json:"job"`
	Version string                `json:"version"`
}

type JerryBuildData struct {
	Builds  []JerryDataItem `json:"builds"`
	Job     string          `json:"job"`
	Version string          `json:"version"`
}

// DataItem is the data point.
type DataItem struct {
	// Data is a map from bucket to real data point (e.g. "Perc90" -> 23.5). Notice
	// that all data items with the same label combination should have the same buckets.
	Data map[string]float64 `json:"data"`
	// Unit is the data unit. Notice that all data items with the same label combination
	// should have the same unit.
	Unit string `json:"unit"`
	// Labels is the labels of the data item.
	Labels Label `json:"labels,omitempty"`
}

type JerryDataItem struct {
	// Data is a map from bucket to real data point (e.g. "Perc90" -> 23.5). Notice
	// that all data items with the same label combination should have the same buckets.
	Data JerryData `json:"data"`
	// Unit is the data unit. Notice that all data items with the same label combination
	// should have the same unit.
	Unit string `json:"unit"`
	// Labels is the labels of the data item.
	Labels Label `json:"labels,omitempty"`
}

type JerryData struct {
	Perc50 float64 `json:"Perc50"`
	Perc90 float64 `json:"Perc90"`
	Perc99 float64 `json:"Perc99"`
}

type Label struct {
	Metric string `json:"Metric"`
}

var (
	addr   = pflag.String("address", ":8080", "The address to serve web data on")
	www    = pflag.Bool("www", false, "If true, start a web-server to server performance data")
	wwwDir = pflag.String("dir", "www", "If non-empty, add a file server for this directory at the root of the web server")
)

func main() {
	klog.InitFlags(nil)
	klog.Infof("Starting perfdash...")
	if err := run(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}

func run() error {

	pflag.Parse()

	// Open our jsonFile
	fmt.Println("trying to open customized data")
	jsonFile, err := os.Open("www/custom_data.json")

	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened users.json")

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// 1. init buildData
	var jerryBuildData JerryBuildData

	json.Unmarshal([]byte(byteValue), &jerryBuildData)

	// fmt.Println(jerryBuildData.Builds)

	// === custom data ====
	customData := make(map[string]float64)
	customData["Perc50"] = 12.55
	customData["Perc90"] = 39.55
	customData["Perc99"] = 42.55

	// customData2 := make(map[string]float64)
	// customData2["Perc50"] = 14.55
	// customData2["Perc90"] = 34.55
	// customData2["Perc99"] = 48.55

	customLabel := Label{
		Metric: "test-metric",
	}

	customDataItem := DataItem{
		Data:   customData,
		Unit:   "ms",
		Labels: customLabel,
	}

	// customDataItem2 := DataItem{
	// 	Data:   customData2,
	// 	Unit:   "ms",
	// 	Labels: customLabel,
	// }

	customBuilds := make(map[string][]DataItem)
	customDataItemSlice := []DataItem{customDataItem}
	// customDataItemCollection2 := []DataItem{customDataItem2}

	customBuilds["data-1"] = customDataItemSlice
	// customDataItemMap["data-2"] = customDataItemCollection2

	newBuildData := BuildData{
		Builds:  customBuilds,
		Job:     "newJob",
		Version: "v2",
	}

	fmt.Println(newBuildData)
	// === custom data ====

	// iterate through jerryBuildData and convert that to metricToBuildData
	jerryBuilds := make(map[string][]DataItem)
	for i, temp := range jerryBuildData.Builds {
		tempCustomData := make(map[string]float64)
		tempCustomData["Perc50"] = temp.Data.Perc50
		tempCustomData["Perc90"] = temp.Data.Perc90
		tempCustomData["Perc99"] = temp.Data.Perc99
		tempDataItem := DataItem{
			Data:   tempCustomData,
			Unit:   "ms",
			Labels: customLabel,
		}
		jerryDataItemSlice := []DataItem{tempDataItem}
		jerryBuilds[string(i)] = jerryDataItemSlice
	}

	newnewBuildData := BuildData{
		Builds:  jerryBuilds,
		Job:     "jerryJob",
		Version: "vjerry",
	}

	// 2. init metricToBuildData map
	metricToBuildData := MetricToBuildData{
		"customMetricNew": &newnewBuildData,
	}

	// 3. init categoryToMetricData map
	categoryToMetricData := CategoryToMetricData{
		"customCategory": metricToBuildData,
	}

	// 4. init jobToCategoryData map
	jobToCategoryData := JobToCategoryData{
		"customJob": categoryToMetricData,
	}

	klog.Infof("Starting server...")
	http.Handle("/", http.FileServer(http.Dir(*wwwDir)))
	http.HandleFunc("/jobnames", jobToCategoryData.ServeJobNames)
	http.HandleFunc("/metriccategorynames", jobToCategoryData.ServeCategoryNames)
	http.HandleFunc("/metricnames", jobToCategoryData.ServeMetricNames)
	http.HandleFunc("/buildsdata", jobToCategoryData.ServeBuildsData)
	// http.HandleFunc("/config", serveConfig)
	fmt.Println("Serving Successful")
	return http.ListenAndServe(*addr, nil)
	return nil
}

// func serveConfig(res http.ResponseWriter, req *http.Request) {
// 	serveHTTPObject(res, req, &globalConfig)
// }

func serveHTTPObject(res http.ResponseWriter, req *http.Request, obj interface{}) {
	data, err := json.Marshal(obj)
	// fmt.Println("==========serveHTTPOBject==========")
	// fmt.Println(data)
	if err != nil {
		res.Header().Set("Content-type", "text/html")
		res.WriteHeader(http.StatusInternalServerError)
		_, err = res.Write([]byte(fmt.Sprintf("<h3>Internal Error</h3><p>%v", err)))
		if err != nil {
			klog.Errorf("unable to write error %v", err)
		}
		return
	}
	res.Header().Set("Content-type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(data)
	if err != nil {
		klog.Errorf("unable to write response data %v", err)
	}
}

func getURLParam(req *http.Request, name string) (string, bool) {
	params, ok := req.URL.Query()[name]
	if !ok || len(params) < 1 {
		return "", false
	}
	return params[0], true
}

// ServeJobNames serves all available job names.
func (j *JobToCategoryData) ServeJobNames(res http.ResponseWriter, req *http.Request) {
	jobNames := make([]string, 0)
	if j != nil {
		for k := range *j {
			jobNames = append(jobNames, k)
		}
	}
	sort.Strings(jobNames)
	serveHTTPObject(res, req, &jobNames)
}

// ServeCategoryNames serves all available category names for given job.
func (j *JobToCategoryData) ServeCategoryNames(res http.ResponseWriter, req *http.Request) {
	jobname, ok := getURLParam(req, "jobname")
	if !ok {
		klog.Warningf("url Param 'jobname' is missing")
		return
	}

	tests, ok := (*j)[jobname]
	if !ok {
		klog.Infof("unknown jobname - %v", jobname)
		return
	}

	categorynames := make([]string, 0)
	for k := range tests {
		categorynames = append(categorynames, k)
	}
	sort.Strings(categorynames)
	serveHTTPObject(res, req, &categorynames)
}

// ServeMetricNames serves all available metric names for given job and category.
func (j *JobToCategoryData) ServeMetricNames(res http.ResponseWriter, req *http.Request) {
	jobname, ok := getURLParam(req, "jobname")
	if !ok {
		klog.Warningf("Url Param 'jobname' is missing")
		return
	}
	categoryname, ok := getURLParam(req, "metriccategoryname")
	if !ok {
		klog.Warningf("Url Param 'metriccategoryname' is missing")
		return
	}

	categories, ok := (*j)[jobname]
	if !ok {
		klog.Infof("unknown jobname - %v", jobname)
		return
	}
	tests, ok := categories[categoryname]
	if !ok {
		klog.Infof("unknown metriccategoryname - %v", categoryname)
		return
	}

	metricnames := make([]string, 0)
	for k := range tests {
		metricnames = append(metricnames, k)
	}
	sort.Strings(metricnames)
	serveHTTPObject(res, req, &metricnames)
}

// ServeBuildsData serves builds data for given job name, category name and test name.
func (j *JobToCategoryData) ServeBuildsData(res http.ResponseWriter, req *http.Request) {
	//fmt.Printf("---------------1---------------")

	jobname, ok := getURLParam(req, "jobname")
	if !ok {
		klog.Warningf("Url Param 'jobname' is missing")
		return
	}
	//fmt.Printf("---------------2---------------")
	categoryname, ok := getURLParam(req, "metriccategoryname")
	if !ok {
		klog.Warningf("Url Param 'metriccategoryname' is missing")
		return
	}
	//fmt.Printf("---------------3---------------")
	metricname, ok := getURLParam(req, "metricname")
	if !ok {
		klog.Warningf("Url Param 'metricname' is missing")
		return
	}
	//fmt.Printf("---------------4---------------")
	categories, ok := (*j)[jobname]
	if !ok {
		klog.Infof("unknown jobname - %v", jobname)
		return
	}
	//fmt.Printf("---------------5---------------")
	tests, ok := categories[categoryname]
	if !ok {
		klog.Infof("unknown metriccategoryname - %v", categoryname)
		return
	}
	//mt.Printf("---------------6---------------")
	builds, ok := tests[metricname]
	if !ok {
		klog.Infof("unknown metricname - %v", metricname)
		return
	}
	// fmt.Printf("---------------7---------------")
	serveHTTPObject(res, req, builds)
}
