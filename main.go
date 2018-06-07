package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

var domain string
var org string
var pythonVersion string
var internalRecipesRepo string
var wg = sync.WaitGroup{}
var mux = sync.Mutex{}
var forcedDeps forcing
var repos astringVar
var allowDouble astringVar
var externalChannel string
var condaBuildVersion string

type forcing []dependency

func (i *forcing) Set(value string) error {
	parts := strings.Split(value, "=")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	if len(parts) == 1 {
		parts = append(parts, "*")
	}
	*i = append(*i, parts)
	return nil
}
func (i *forcing) String() string { return "" }

type astringVar []string

func (i *astringVar) Set(value string) error {
	*i = append(*i, value)
	return nil
}
func (i *astringVar) String() string { return "" }

type dependency []string
type depInfo struct {
	dep dependency
	url string
}

func (d *depInfo) scanned() bool {
	return d.url != ""
}

type CondaPackageInfo struct {
	Depends []string `json:"depends"`
	Version string   `json:"version"`
	Url     string   `json:"url"`
}
type Meta struct {
	Package struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"package"`
	Source struct {
		Url string `yaml:"url"`
		Fn  string `yaml:"fn"`
	} `yaml:"source"`
	Requirements struct {
		Build []string `yaml:"build"`
		Run   []string `yaml:"run"`
	} `yaml:"requirements"`
}
type Env struct {
	Name         string   `yaml:"name"`
	Channels     []string `yaml:"channels"`
	Dependencies []string `yaml:"dependencies"`
}

var condarc = `channels:
  - local
  - bioconda
  - conda-forge
  - defaults
`

func cleanup() {
	for _, repo := range repos {
		exec.Command("rm", "-rf", repo).Run()
	}
	exec.Command("rm", "-rf", internalRecipesRepo).Run()
	exec.Command("rm", "-rf", "miniconda2")
}

func main() {
	defer cleanup()
	flag.StringVar(&domain, "domain", "https://github.com", "Domain to pull from")
	flag.StringVar(&org, "org", "venicegeo", "Org")
	flag.StringVar(&pythonVersion, "pyversion", "2.7.13", "Version of python to use")
	flag.StringVar(&internalRecipesRepo, "inrecipesrepo", "venicegeo-conda-recipes", "Internal Recipes Repo")
	flag.StringVar(&externalChannel, "echannel", "", "External channel local path")
	flag.StringVar(&condaBuildVersion, "build", "3.0.6", "Which version of conda-build to use")
	flag.Var(&forcedDeps, "force", "Force a dependency version")
	flag.Var(&repos, "repo", "Add a repository to scan")
	flag.Var(&allowDouble, "allow", "Allow duplicate package versions")
	flag.Parse()
	fmt.Println(repos, forcedDeps, allowDouble)
	for _, repo := range append(repos, internalRecipesRepo) {
		execute("git", "clone", f("%s/%s/%s", domain, org, repo))
	}
	execute("curl", "-L", "https://repo.continuum.io/miniconda/Miniconda2-4.3.21-Linux-x86_64.sh", "-o", "miniconda.sh")
	execute("bash", "miniconda.sh", "-b", "-p", "./miniconda2")
	execute("rm", "miniconda.sh")
	ioutil.WriteFile(os.Getenv("HOME")+"/.condarc", []byte(condarc), 0644)
	conda("config", "--set", "auto_update_conda", "false")
	if externalChannel != "" {
		conda("config", "--add", "channels", externalChannel)
	}
	allDependencies := []depInfo{}
	directDeps := []dependency{}

	for _, repo := range repos {
		dat, err := ioutil.ReadFile(repo + "/environment.yml")
		check(err)
		var env Env
		check(yaml.Unmarshal(dat, &env))
		for _, d := range env.Dependencies {
			addTo(strings.Split(d, "="), &allDependencies)
		}
	}
	recipes, err := ioutil.ReadDir(internalRecipesRepo + "/recipes")
	check(err)
	recipeNames := make([]string, len(recipes), len(recipes))
	for i, recipe := range recipes {
		dat, err := ioutil.ReadFile(internalRecipesRepo + "/recipes/" + recipe.Name() + "/meta.yaml")
		check(err)
		var yml Meta
		check(yaml.Unmarshal(dat, &yml))
		for _, dep := range append(yml.Requirements.Build, yml.Requirements.Run...) {
			addTo(strings.Split(dep, " "), &allDependencies)
		}
		recipeNames[i] = yml.Package.Name
	}
	for _, rn := range recipeNames {
		for i, dep := range allDependencies {
			if dep.dep[0] == rn {
				allDependencies = append(allDependencies[:i], allDependencies[i+1:]...)
				break
			}
		}
	}
	allDependencies = append(allDependencies, depInfo{dependency{"conda-build", condaBuildVersion}, ""})
	log.Println("First pass - direct dependencies")
	for _, d := range allDependencies {
		fmt.Println(d.dep)
		directDeps = append(directDeps, d.dep)
	}

	i := 2
	p := func() {
		log.Println("Pass", i)
		for _, d := range allDependencies {
			fmt.Println(d.dep, "\t\t", d.url)
		}
	}
	for {
		for ; scan(&allDependencies, false); i++ {
			p()
		}
		p()
		if !scan(&allDependencies, true) {
			break
		}
		p()
	}
	log.Println("RESULT")
	for _, d := range allDependencies {
		fmt.Println(d.dep, "\t\t", d.url)
	}
	execute("mkdir", "-p", "output/linux-64")
	execute("mkdir", "-p", "output/noarch")
	csvDat := make([][]string, len(allDependencies), len(allDependencies))
	for i, d := range allDependencies {
		if d.url == "" {
			log.Fatalln("Was unable to solve this problem")
		}
		toAdd := make([]string, 0, 4)
		toAdd = append(toAdd, d.dep[0], d.dep[1])
		for _, di := range directDeps {
			if di[0] == d.dep[0] && di[1] == d.dep[1] {
				toAdd = append(toAdd, "direct")
				break
			}
		}
		if len(toAdd) != 3 {
			toAdd = append(toAdd, "transitive")
		}
		if strings.HasPrefix(d.url, "file://") {
			output("cp", strings.TrimPrefix(d.url, "file:/"), "output/linux-64/")
			toAdd = append(toAdd, f("%s/%s/conda-provisioning/tree/master/recipes", domain, org))
		} else {
			executeDir("output/linux-64", "wget", d.url)
			toAdd = append(toAdd, d.url)
		}
		csvDat[i] = toAdd
	}
	file, err := os.Create("output/index.csv")
	check(err)
	w := csv.NewWriter(file)
	check(w.WriteAll(csvDat))
	w.Flush()
	conda("install", "conda-build="+condaBuildVersion, "-y")
	conda("index", "output/linux-64")
	conda("index", "output/noarch")
}

func scan(deps *[]depInfo, scanPatterns bool) bool {
	somethingSuccessfullyScanned := false
	for i, dep := range *deps {
		if dep.scanned() {
			continue
		}
		log.Println("scanning", dep.dep)
		if !versionIsNotPattern(dep.dep[1]) && !scanPatterns {
			log.Println("version is a pattern and this isnt a pattern scan")
			continue
		}
		name := strings.Replace(strings.Join(dep.dep, "="), "=>=", ">=", -1)
		dat, err := oconda("info", "--json", name)
		if err != nil {
			log.Println("Error running against", name)
			continue
		}
		var info map[string][]CondaPackageInfo
		check(json.Unmarshal(dat, &info))
		var channelToUse *CondaPackageInfo = nil
		for _, channelInfo := range info[name] {
			pygood := false
			pyfound := false
			for _, dep := range channelInfo.Depends {
				parts := strings.SplitN(dep, " ", 2)
				if parts[0] != "python" || len(parts) == 1 {
					continue
				}
				pyfound = true
				if len(parts) == 2 && (parts[1] == "2.7*" || strings.HasPrefix(parts[1], ">=2.7")) {
					pygood = true
				} else {
					pygood = false
				}
				break
			}
			if pygood || !pyfound {
				channelToUse = &channelInfo
				break
			}
		}
		if channelToUse != nil {
			somethingSuccessfullyScanned = true
			(*deps)[i].url = channelToUse.Url
			if scanPatterns {
				(*deps)[i].dep[1] = channelToUse.Version
			}
			for _, d := range channelToUse.Depends {
				addTo(strings.Split(d, " "), deps)
			}
		} else {
			log.Println("No good channel found for", name)
		}
	}
	return somethingSuccessfullyScanned
}

func versionIsNotPattern(version string) bool {
	return !strings.ContainsAny(version, "*><|")
}

func addTo(dep dependency, deps *[]depInfo) {
	for i, p := range dep {
		dep[i] = strings.ToLower(p)
	}
	dep[0] = strings.Replace(dep[0], ".", "", -1)
	if dep[0] == "python" || dep[0] == "conda" {
		return
	}
	for _, d := range forcedDeps {
		if dep[0] == d[0] {
			dep = d
		}
	}
	if len(dep) == 1 {
		dep = append(dep, "*")
	}
	log.Println("attempting to add", dep)
	for i, e := range *deps {
		if e.dep[0] != dep[0] {
			continue
		}
		erro := func() {
			log.Fatalf("The package %s wants to use both versions %s and %s. Cannot continue until this is resolved.\n", e.dep[0], e.dep[1], dep[1])
		}
		existingVersionPattern := e.dep[1]
		newVersionPattern := dep[1]
		if existingVersionPattern == newVersionPattern {
			return
		} else if versionIsNotPattern(existingVersionPattern) && versionIsNotPattern(newVersionPattern) {
			for _, a := range allowDouble {
				if a == e.dep[0] {
					return
				}
			}
			erro()
		} else if versionIsNotPattern(existingVersionPattern) {
			if !testVersionMatchesPattern(existingVersionPattern, newVersionPattern) {
				erro()
			}
		} else if versionIsNotPattern(newVersionPattern) {
			if !testVersionMatchesPattern(newVersionPattern, existingVersionPattern) {
				erro()
			}
		} else {
			tmp, ok := minPattern(existingVersionPattern, newVersionPattern)
			if ok {
				(*deps)[i].dep[1] = tmp
				return
			}
			for _, a := range allowDouble {
				if a == e.dep[0] {
					return
				}
			}
			erro()
		}
		return
	}
	*deps = append(*deps, depInfo{dep, ""})
}

func check(err error, a ...interface{}) {
	if err != nil {
		defer cleanup()
		log.Fatalln(append([]interface{}{err.Error()}, a...))
	}
}
func execute(name string, args ...string) {
	dat, err := exec.Command(name, args...).CombinedOutput()
	check(err, string(dat))
}
func executeDir(dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	check(cmd.Run())
}
func output(name string, args ...string) string {
	dat, err := exec.Command(name, args...).CombinedOutput()
	check(err, string(dat))
	return string(dat)
}

func f(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

func conda(args ...string) {
	execute("./miniconda2/bin/conda", args...)
}
func oconda(args ...string) ([]byte, error) {
	return exec.Command("./miniconda2/bin/conda", args...).Output()
}
func testVersionMatchesPattern(version, pattern string) bool {
	version, pattern = strings.ToLower(version), strings.ToLower(pattern)
	if strings.Contains(pattern, "|") {
		parts := strings.SplitN(pattern, "|", 2)
		return testVersionMatchesPattern(version, parts[0]) || testVersionMatchesPattern(version, parts[1])
	} else if strings.Contains(pattern, "*") {
		tmp := []byte(version)
		for i, l := range []byte(pattern) {
			if i == len(tmp) {
				if pattern[i:] == ".*" {
					return true
				} else {
					return false
				}
			}
			if l == 42 {
				return true
			}
			if l != tmp[i] {
				return false
			}
		}
	} else if strings.ContainsAny(pattern, "><") {
		parts := strings.Split(pattern, ",")
		gtes := strings.TrimPrefix(parts[0], ">=")
		if len(parts) > 1 {
			lts := strings.TrimPrefix(parts[1], "<")
			return gte(convertPattern(version), convertPattern(gtes)) && lt(convertPattern(version), convertPattern(lts))
		} else {
			return gte(convertPattern(version), convertPattern(gtes))
		}
	} else {
		return version == pattern
	}
	return false
}
func convertPattern(pattern string) []uint64 {
	parts := strings.Split(pattern, ".")
	conv := make([]uint64, len(parts), len(parts))
	var err error = nil
	for i, p := range parts {
		conv[i], err = strconv.ParseUint(p, 36, 64)
		check(err)
	}
	return conv
}
func gte(version, expected []uint64) bool {
	min := len(version)
	if min > len(expected) {
		min = len(expected)
	}
	for i := 0; i < min; i++ {
		if version[i] > expected[i] {
			return true
		} else if version[i] == expected[i] {
			continue
		} else {
			return false
		}
	}
	return true
}
func lt(version, expected []uint64) bool {
	min := len(version)
	if min > len(expected) {
		min = len(expected)
	}
	for i := 0; i < min; i++ {
		if version[i] < expected[i] {
			return true
		} else if version[i] == expected[i] {
			continue
		} else {
			return false
		}
	}
	return false
}
func minPattern(old, neww string) (string, bool) {
	if old == "*" {
		return neww, true
	} else if neww == "*" {
		return old, true
	}
	return "", false
}
