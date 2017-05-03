package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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
		if err != nil {
			continue
		}
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

func interval() float64 {
	b, err := ioutil.ReadFile("/etc/collectd/collectd.conf")
	check(err)

	s := string(b)
	i, _ := regexp.Compile(`Interval`)

	if i.MatchString(s) == true {
		str := strings.Split(s, "\"")[1]
		in, _ := strconv.Atoi(str)
		if in == 0 {
			return float64(60)
		}
		return float64(in)
	}

	return float64(60)
}

func hostname() string {
	b, err := ioutil.ReadFile("/etc/collectd/collectd.conf")
	check(err)

	s := string(b)
	h, _ := regexp.Compile(`Hostname`)

	if h.MatchString(s) == true {
		return strings.Split(s, "\"")[1]
	}
	return ""
}

func main() {
	var username = flag.String("u", "user", "open files of username")
	flag.Parse()
	dirs := listProcs(username)
	openFiles := countOpenFiles(dirs)
	percent := countPercent(openFiles)
	fmt.Printf("PUTVAL %s/system/openfiles interval=%f N:%f\n", hostname(), interval(), percent)
}
