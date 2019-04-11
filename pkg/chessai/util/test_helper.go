package util

import (
	"io/ioutil"
	"log"
	"strings"
)

func LoadBoardFile(fileName string) (lines []string, skip bool) {
	fileData, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}
	fileStr := strings.ReplaceAll(string(fileData), "\r", "")
	lines = strings.Split(fileStr, "\n")
	if strings.Contains(lines[0], "skip") {
		log.Printf("WARNING Skipping test %s\n", fileName)
		skip = true
	}
	return
}
