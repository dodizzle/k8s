package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

var (
	kubeCtl = "kubectl"
	gcloud  = "gcloud"
)

func main() {

	args := os.Args

	// print usage
	if len(args[1:]) < 1 {
		fmt.Println("Usage: ", args[0], `
		 -cn (change namespace)
		 -cc (change context)
		 -cp (change project)
		 -lc (list contexts)
		 -ln (list namespaces)
		 -lp (list google projects)
		 -t (generate token for proxy auth)`)
		os.Exit(0)
	}

	if args[1] == "-cc" {
		context := getContexts(kubeCtl)
		setContext(context, kubeCtl)
		printCurrentCluster(kubeCtl)
	} else if args[1] == "-t" {
		defaultSecret := getDefaultSecret(kubeCtl)
		defaultToken := getDefaultToken(defaultSecret, kubeCtl)
		decodeToken(defaultToken)
	} else if args[1] == "-cp" {
		project := getProjects(gcloud)
		setProject(project, gcloud)
		printCurrentProject(gcloud)
	} else if args[1] == "-cn" {
		context := currentContext(kubeCtl)
		namespace := getNameSpaces(kubeCtl, context)
		setNameSpace(kubeCtl, context, namespace)
	} else if args[1] == "-lc" {
		printCurrentCluster(kubeCtl)
	} else if args[1] == "-lp" {
		printCurrentProject(gcloud)
	} else if args[1] == "-ln" {
		printNameSpaces(kubeCtl)
	}
}

func setNameSpace(kubeCtl string, context string, namespace string) {
	newNamespace := "--namespace=" + namespace
	fmt.Println(newNamespace)
	fmt.Println(context)
	out, err := exec.Command(kubeCtl, "config", "set-context", strings.TrimSpace(context), strings.TrimSpace(newNamespace)).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func currentContext(kubeCtl string) string {
	out, err := exec.Command(kubeCtl, "config", "current-context").Output()
	if err != nil {
		log.Fatal(err)
	}
	return (string(out))
}

func getNameSpaces(kubeCtl string, context string) string {
	out, err := exec.Command(kubeCtl, "get", "namespaces", "-o", "name").Output()
	if err != nil {
		log.Fatal(err)
	}
	dat := string(out)

	lines := strings.Split(dat, "\n")

	contextsReturn := map[int][]string{}
	startingNum := 0
	// remove header from cli output
	// lines = append(lines[:0], lines[0+1:]...)
	// remove the last slice which is empty
	lines = lines[:len(lines)-1]

	for _, l := range lines {
		startingNum++
		contextsReturn[startingNum] = []string{strings.Replace(l, "namespace/", "", -1)}
	}

	contextsTable := tablewriter.NewWriter(os.Stdout)
	contextsTable.SetHeader([]string{"", "NameSpaces"})

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
	fmt.Print("NameSpace to use? ")
	contextsIndex, _ := reader.ReadString('\n')

	contextChoice, erry := strconv.Atoi(strings.TrimSpace(contextsIndex))
	if erry != nil {
		fmt.Println("Error:", err)
	}
	return (contextsReturn[contextChoice][0])
}

// gcloud config set project $gcloud_project
func setGoogleProject(gcloud string, projectname string) {
	out, err := exec.Command(gcloud, "config", "set", "project", projectname).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func printGcloudAuth(gcloud string) {
	out, err := exec.Command(gcloud, "auth", "list").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func activateServiceAccount(gcloud string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Service Account .json file to use? ")
	text, _ := reader.ReadString('\n')
	fmt.Println(strings.TrimSpace(text))
	jsonFile := "--key-file=" + strings.TrimSpace(text)
	out, err := exec.Command(gcloud, "auth", "activate-service-account", jsonFile).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func setProject(project string, gcloud string) {
	out, err := exec.Command(gcloud, "config", "configurations", "activate", project).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func printCurrentCluster(kubeCtl string) {
	out, err := exec.Command(kubeCtl, "config", "get-contexts").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func printNameSpaces(kubeCtl string) {
	out, err := exec.Command(kubeCtl, "get", "namespaces").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func printCurrentProject(gcloud string) {
	out, err := exec.Command(gcloud, "config", "configurations", "list").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func getProjects(gcloud string) string {
	filter := `--format=value(name.scope())`
	//out, err := exec.Command(gcloud, "config", "configurations", "list", "--format='value(name.scope())'").Output()
	out, err := exec.Command(gcloud, "config", "configurations", "list", filter).Output()
	if err != nil {
		log.Fatal(err)
	}
	dat := string(out)
	lines := strings.Split(dat, "\n")
	contextsReturn := map[int][]string{}
	startingNum := 0
	// remove the last slice which is empty
	lines = lines[:len(lines)-1]
	for _, l := range lines {
		startingNum++
		contextsReturn[startingNum] = []string{l}
	}
	contextsTable := tablewriter.NewWriter(os.Stdout)
	contextsTable.SetHeader([]string{"", "Configurations"})

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

// BytesToString converts bytes to a string
func BytesToString(data []byte) string {
	return string(data[:])
}

func decodeToken(defaultToken string) {
	data, err := base64.StdEncoding.DecodeString(defaultToken)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	cleanToken := BytesToString(data)
	fmt.Println("Copy and paste this:")
	fmt.Println(cleanToken)
}

func getDefaultToken(defaultSecret string, kubeCtl string) string {
	out, err := exec.Command(kubeCtl, "get", "secret", defaultSecret, "-o", "json").Output()
	if err != nil {
		log.Fatal(err)
	}
	type AutoGenerated struct {
		APIVersion string `json:"apiVersion"`
		Data       struct {
			CaCrt     string `json:"ca.crt"`
			Namespace string `json:"namespace"`
			Token     string `json:"token"`
		} `json:"data"`
		Kind     string `json:"kind"`
		Metadata struct {
			Annotations struct {
				KubernetesIoServiceAccountName string `json:"kubernetes.io/service-account.name"`
				KubernetesIoServiceAccountUID  string `json:"kubernetes.io/service-account.uid"`
			} `json:"annotations"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			ResourceVersion   string    `json:"resourceVersion"`
			SelfLink          string    `json:"selfLink"`
			UID               string    `json:"uid"`
		} `json:"metadata"`
		Type string `json:"type"`
	}
	var result AutoGenerated
	json.Unmarshal([]byte(out), &result)
	return result.Data.Token
}

func getDefaultSecret(kubeCtl string) string {
	out, err := exec.Command(kubeCtl, "get", "sa", "default", "-o", "json").Output()
	if err != nil {
		log.Fatal(err)
	}
	type AutoGenerated struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Metadata   struct {
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			ResourceVersion   string    `json:"resourceVersion"`
			SelfLink          string    `json:"selfLink"`
			UID               string    `json:"uid"`
		} `json:"metadata"`
		Secrets []struct {
			Name string `json:"name"`
		} `json:"secrets"`
	}
	var result AutoGenerated
	json.Unmarshal([]byte(out), &result)
	return result.Secrets[0].Name
}

func setContext(context string, kubeCtl string) {
	out, err := exec.Command(kubeCtl, "config", "use-context", context).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func getContexts(kubeCtl string) string {
	out, err := exec.Command(kubeCtl, "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		log.Fatal(err)
	}
	dat := string(out)

	lines := strings.Split(dat, "\n")

	contextsReturn := map[int][]string{}
	startingNum := 0
	// remove header from cli output
	// lines = append(lines[:0], lines[0+1:]...)
	// remove the last slice which is empty
	lines = lines[:len(lines)-1]

	for _, l := range lines {
		startingNum++
		contextsReturn[startingNum] = []string{l}
	}

	contextsTable := tablewriter.NewWriter(os.Stdout)
	contextsTable.SetHeader([]string{"", "Contexts"})

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
