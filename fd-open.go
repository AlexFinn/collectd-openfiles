package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"collectd.org/api"
	"collectd.org/exec"
)

// check for errors and bail out if one is hit
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// find all directories in /proc with a numeric name corresponding to a process id
// return a slice consisting of directory names for all found numeric process ids
func listProcs(u *string) []string {
	mainDir := "/proc"
	procFiles, err := ioutil.ReadDir(mainDir)
	check(err)
	dirs := []string{}
	username := *u
	user, err := user.Lookup(username)
	check(err)
	userid := user.Uid

	for _, fileInfo := range procFiles {
		isDir := fileInfo.IsDir()
		fileName := fileInfo.Name()
		stats, _ := os.Stat(mainDir + "/" + fileName)
		check(err)
		fileUid := stats.Sys().(*syscall.Stat_t).Uid
		matched, _ := regexp.MatchString("^[0-9]+$", fileName)
		if isDir == true && matched == true && userid == strconv.Itoa(int(fileUid)) {
			dirs = append(dirs, fileName)
		}
	}
	return dirs
}

// count the number of files in /proc/n/fd where n is a process id like 12345
// return the length as an int of the ioutil.ReadDir result
func countFds(dir string) int {
	fds, err := ioutil.ReadDir("/proc/" + dir + "/fd")
	check(err)
	return len(fds)
}

// take a slice of dirs, iterate over and find number of open files in each
// add the result and return the total number of open files found in /proc/**/fd
func countOpenFiles(dirs []string) int {
	openFiles := 0
	for _, dir := range dirs {
		openFiles = openFiles + countFds(dir)
	}
	return openFiles
}

func countPercent(opened int) float64 {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	check(err)
	return float64(opened) * 100 / float64(rLimit.Max)
}

// get a list of dirs in /proc corresponding to process ids
// recurse through each and calculate number of open files
// print the result
func main() {
	var username = flag.String("u", "user", "open files of username")
	flag.Parse()
	dirs := listProcs(username)
	openFiles := countOpenFiles(dirs)
	percent := countPercent(openFiles)
	vl := api.ValueList{
		Identifier: api.Identifier{
			Host:   exec.Hostname(),
			Plugin: "system",
			Type:   "openfiles",
		},
		Time:     time.Now(),
		Interval: exec.Interval(),
		Values:   []api.Value{api.Gauge(percent)},
	}
	exec.Putval.Write(context.Background(), &vl)
}
