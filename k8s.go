package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

var (
	kubeCtl = "/Users/daveo/KUBE/google-cloud-sdk/bin/kubectl"
)

func main() {

	context := getContexts(kubeCtl)
	setContext(context)
}

func setContext(context string) {
	out, err := exec.Command("/Users/daveo/KUBE/google-cloud-sdk/bin/kubectl", "config", "use-context", context).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func getContexts(kubeCtl string) string {
	out, err := exec.Command("/Users/daveo/KUBE/google-cloud-sdk/bin/kubectl", "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		log.Fatal(err)
	}
	dat := string(out)

	lines := strings.Split(dat, "\n")

	contextsReturn := map[int][]string{}
	startingNum := 0
	// remove header from cli output
	lines = append(lines[:0], lines[0+1:]...)
	for _, l := range lines {
		startingNum++
		//fmt.Println(startingNum, l)
		contextsReturn[startingNum] = []string{l}
	}
	//fmt.Println(contextsReturn[1])
	contextsTable := tablewriter.NewWriter(os.Stdout)
	contextsTable.SetHeader([]string{"", "Contexts"})
	// now
	var keys []int
	for k := range contextsReturn {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		contextsTable.Append([]string{strconv.Itoa(k), contextsReturn[k][0]})
	}
	contextsTable.Render()
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Context to use? ")
	contextsIndex, _ := reader.ReadString('\n')

	contextChoice, erry := strconv.Atoi(strings.TrimSpace(contextsIndex))
	if erry != nil {
		fmt.Println("Error:", err)
	}
	return (contextsReturn[contextChoice][0])
}
