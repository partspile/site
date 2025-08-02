package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <make-year-model.json>\n", os.Args[0])
		os.Exit(1)
	}
	filename := os.Args[1]
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	var makes map[string]interface{}
	if err := json.Unmarshal(data, &makes); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}
	makeList := make([]string, 0, len(makes))
	for make := range makes {
		makeList = append(makeList, make)
	}
	sort.Strings(makeList)
	for _, make := range makeList {
		fmt.Println(make)
	}
}
