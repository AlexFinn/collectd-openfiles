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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

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
		stats, err := os.Stat(mainDir + "/" + fileName)
		check(err)
		fileUid := stats.Sys().(*syscall.Stat_t).Uid
		matched, _ := regexp.MatchString("^[0-9]+$", fileName)
		if isDir == true && matched == true && userid == strconv.Itoa(int(fileUid)) {
			dirs = append(dirs, fileName)
		}
	}
	return dirs
}

func countFds(dir string) int {
	fds, err := ioutil.ReadDir("/proc/" + dir + "/fd")
	check(err)
	return len(fds)
}

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
