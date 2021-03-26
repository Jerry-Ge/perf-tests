/*
Copyright 2016 The Kubernetes Authors.

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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/perftype"
)

// DownloaderOptions is an options for Downloader.
type DownloaderOptions struct {
	Mode               string
	ConfigPaths        []string
	GithubConfigDirs   []string
	DefaultBuildsCount int
	// Development-only flag.
	// Overrides build count from "perfDashBuildsCount" label with DefaultBuildsCount.
	OverrideBuildCount bool
}

// Downloader that gets data about results from a storage service (GCS) repository.
type Downloader struct {
	MetricsBkt MetricsBucket
	Options    *DownloaderOptions
}

// NewDownloader creates a new Downloader.
func NewDownloader(opt *DownloaderOptions, bkt MetricsBucket) *Downloader {
	return &Downloader{
		MetricsBkt: bkt,
		Options:    opt,
	}
}

// TODO(random-liu): Only download and update new data each time.
func (g *Downloader) getData() (JobToCategoryData, error) {
	result := make(JobToCategoryData)
	var resultLock sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1)
	go g.getJobData(&wg, result, &resultLock)
	wg.Wait()
	return result, nil
}

/*
getJobData fetches build numbers, reads metrics data from GCS and
updates result with parsed metrics for a given prow job. Assumptions:
- metric files are in /artifacts directory
- metric file names have following prefix: {{OutputFilePrefix}}_{{Name}},
  where OutputFilePrefix and Name are parts of test description (specified in prefdash config)
- if there are multiple files with a given prefix, then expected format is
  {{OutputFilePrefix}}_{{Name}}_{{SuiteId}}. SuiteId is prepended to the category label,
  which allows comparing metrics across several runs in a given suite
*/
func (g *Downloader) getJobData(wg *sync.WaitGroup, result JobToCategoryData, resultLock *sync.Mutex) {
	defer wg.Done()

	// Hack: hardcode the fileName to be our fileName, update
	buildNumber := 123456789
	fileName := "/root/.go/src/perf-tests/perfdash/jerry/PodStartupLatency_PodStartupLatency_node-throughput_2020.json"
	jsonFile, err := os.Open(fileName)
	defer jsonFile.Close()
	testDataResponse, _ := ioutil.ReadAll(jsonFile)

	JerryPrefix := "E2E"
	JerryResultCategory := "configmap_vol_per_node_E2E"
	JerryTestLabel := "PodStartup"
	JerryJob := "ci-kubernetes-storage-scalability"

	if err != nil {
		klog.Infof("Error when reading response Body for %q: %v", fileName, err)
	}

	buildData := getBuildData(result, JerryPrefix, JerryResultCategory, JerryTestLabel, JerryJob, resultLock)
	// fmt.Println(buildData)
	// fmt.Println(testDataResponse)
	// testDescription.Parser = parsePerfData
	parsePerfData(testDataResponse, buildNumber, buildData)

}

func (g *Downloader) artifactName(jobAttrs Tests, file string) string {
	return path.Join(jobAttrs.ArtifactsDir, file)
}

func getResultCategory(metricsFileName string, filePrefix string, category string, artifacts []string) string {
	if len(artifacts) <= 1 {
		return category
	}
	// If there are more artifacts, assume that this is a test suite run.
	trimmed := strings.TrimPrefix(metricsFileName, filePrefix+"_")
	suiteID := strings.Split(trimmed, "_")[0]
	return fmt.Sprintf("%v_%v", suiteID, category)
}

func getBuildData(result JobToCategoryData, prefix string, category string, label string, job string, resultLock *sync.Mutex) *BuildData {
	resultLock.Lock()
	defer resultLock.Unlock()
	if _, found := result[prefix]; !found {
		result[prefix] = make(CategoryToMetricData)
	}
	if _, found := result[prefix][category]; !found {
		result[prefix][category] = make(MetricToBuildData)
	}
	if _, found := result[prefix][category][label]; !found {
		result[prefix][category][label] = &BuildData{Job: job, Version: "", Builds: map[string][]perftype.DataItem{}}
	}
	return result[prefix][category][label]
}

// MetricsBucket is the interface that fetches data from a storage service.
type MetricsBucket interface {
	GetBuildNumbers(job string) ([]int, error)
	ListFilesInBuild(job string, buildNumber int, prefix string) ([]string, error)
	ReadFile(job string, buildNumber int, path string) ([]byte, error)
}

func joinStringsAndInts(pathElements ...interface{}) string {
	var parts []string
	for _, e := range pathElements {
		switch t := e.(type) {
		case string:
			parts = append(parts, t)
		case int:
			parts = append(parts, strconv.Itoa(t))
		default:
			panic(fmt.Sprintf("joinStringsAndInts only accepts ints and strings as path elements, but was passed %#v", t))
		}
	}
	return path.Join(parts...)
}
