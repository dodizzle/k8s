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
		 -dc (delete context)
		 -lc (list contexts)
		 -ln (list namespaces)
		 -lp (list google projects)
		 -rc (rename context)
		 -rp (rename gcloud configuration)
		 -t (generate token for proxy auth)`)
		os.Exit(0)
	}

	switch args[1] {
	case "-cc":
		context := getContexts(kubeCtl)
		setContext(context, kubeCtl)
		printCurrentCluster(kubeCtl)
	case "-dc":
		context := getContexts(kubeCtl)
		deleteContext(context, kubeCtl)
		printCurrentCluster(kubeCtl)
	case "-rc":
		context := getContexts(kubeCtl)
		renameContext(context, kubeCtl)
		printCurrentCluster(kubeCtl)
	case "-t":
		defaultSecret := getDefaultSecret(kubeCtl)
		defaultToken := getDefaultToken(defaultSecret, kubeCtl)
		decodeToken(defaultToken)
	case "-cp":
		project := getProjects(gcloud)
		setProject(project, gcloud)
		printCurrentProject(gcloud)
	case "-rp":
		project := getProjects(gcloud)
		renameProject(project, gcloud)
		printCurrentProject(gcloud)
	case "-cn":
		context := currentContext(kubeCtl)
		namespace := getNameSpaces(kubeCtl, context)
		setNameSpace(kubeCtl, context, namespace)
	case "-lc":
		printCurrentCluster(kubeCtl)
	case "-lp":
		printCurrentProject(gcloud)
	case "-ln":
		printNameSpaces(kubeCtl)
	default:
		fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[1])
		os.Exit(1)
	}
}

func renameContext(context string, kubeCtl string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("New context name? ")
	input, _ := reader.ReadString('\n')
	out, err := exec.Command(kubeCtl, "config", "rename-context", strings.TrimSpace(context), strings.TrimSpace(input)).Output()
	if err != nil {
		log.Fatalf("Failed to rename context %q: %v", context, err)
	}
	return string(out)
}

func setNameSpace(kubeCtl string, context string, namespace string) {
	newNamespace := "--namespace=" + namespace
	fmt.Println(newNamespace)
	fmt.Println(context)
	out, err := exec.Command(kubeCtl, "config", "set-context", strings.TrimSpace(context), strings.TrimSpace(newNamespace)).Output()
	if err != nil {
		log.Fatalf("Failed to set namespace on context %q: %v", context, err)
	}
	fmt.Println(string(out))
}

func currentContext(kubeCtl string) string {
	out, err := exec.Command(kubeCtl, "config", "current-context").Output()
	if err != nil {
		log.Fatalf("Failed to get current context: %v\nSet one with: kubectl config use-context <name>", err)
	}
	return string(out)
}

func getNameSpaces(kubeCtl string, context string) string {
	out, err := exec.Command(kubeCtl, "get", "namespaces", "-o", "name").Output()
	if err != nil {
		log.Fatalf("Failed to get namespaces: %v\nIs your kubectl context connected to a cluster?", err)
	}

	lines := parseLines(string(out), "namespace/")
	if len(lines) == 0 {
		fmt.Println("No namespaces found in the current cluster.")
		os.Exit(0)
	}

	items := buildNumberedMap(lines)

	contextsTable := tablewriter.NewTable(os.Stdout)
	contextsTable.Header("", "NameSpaces")
	var keys []int
	for k := range items {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		contextsTable.Append(strconv.Itoa(k), items[k][0])
	}
	contextsTable.Render()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("NameSpace to use? ")
	input, _ := reader.ReadString('\n')

	selection, err := validateSelection(input, items)
	if err != nil {
		log.Fatal(err)
	}
	return selection
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
		log.Fatalf("Failed to activate configuration %q: %v", project, err)
	}
	fmt.Println(string(out))
}

func renameProject(oldName string, gcloud string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("New configuration name? ")
	input, _ := reader.ReadString('\n')
	newName := strings.TrimSpace(input)

	// create new configuration
	_, err := exec.Command(gcloud, "config", "configurations", "create", newName).Output()
	if err != nil {
		log.Fatalf("Failed to create configuration %q: %v", newName, err)
	}

	// get properties from old configuration to copy over
	props, err := exec.Command(gcloud, "config", "configurations", "describe", oldName, "--format=json").Output()
	if err != nil {
		log.Fatalf("Failed to read configuration %q: %v", oldName, err)
	}

	type ConfigProps struct {
		Properties struct {
			Compute struct {
				Region string `json:"region"`
				Zone   string `json:"zone"`
			} `json:"compute"`
			Core struct {
				Account string `json:"account"`
				Project string `json:"project"`
			} `json:"core"`
		} `json:"properties"`
	}
	var cfg ConfigProps
	json.Unmarshal(props, &cfg)

	// activate new configuration before setting properties
	exec.Command(gcloud, "config", "configurations", "activate", newName).Run()

	// copy properties to new configuration
	if cfg.Properties.Core.Account != "" {
		exec.Command(gcloud, "config", "set", "account", cfg.Properties.Core.Account).Run()
	}
	if cfg.Properties.Core.Project != "" {
		exec.Command(gcloud, "config", "set", "project", cfg.Properties.Core.Project).Run()
	}
	if cfg.Properties.Compute.Zone != "" {
		exec.Command(gcloud, "config", "set", "compute/zone", cfg.Properties.Compute.Zone).Run()
	}
	if cfg.Properties.Compute.Region != "" {
		exec.Command(gcloud, "config", "set", "compute/region", cfg.Properties.Compute.Region).Run()
	}

	// delete old configuration
	_, err = exec.Command(gcloud, "config", "configurations", "delete", oldName, "--quiet").Output()
	if err != nil {
		log.Fatalf("Failed to delete old configuration %q: %v", oldName, err)
	}

	fmt.Printf("Renamed configuration %q to %q\n", oldName, newName)
}

func printCurrentCluster(kubeCtl string) {
	out, err := exec.Command(kubeCtl, "config", "get-contexts").Output()
	if err != nil {
		log.Fatalf("Failed to get kubectl contexts: %v", err)
	}
	fmt.Println(string(out))
}

func printNameSpaces(kubeCtl string) {
	out, err := exec.Command(kubeCtl, "get", "namespaces").Output()
	if err != nil {
		log.Fatalf("Failed to list namespaces: %v\nIs your kubectl context connected to a cluster?", err)
	}
	fmt.Println(string(out))
}

func printCurrentProject(gcloud string) {
	out, err := exec.Command(gcloud, "config", "configurations", "list").Output()
	if err != nil {
		log.Fatalf("Failed to list gcloud configurations: %v", err)
	}
	fmt.Println(string(out))
}

func getProjects(gcloud string) string {
	filter := `--format=value(name.scope())`
	out, err := exec.Command(gcloud, "config", "configurations", "list", filter).Output()
	if err != nil {
		log.Fatalf("Failed to list gcloud configurations: %v\nIs gcloud installed and configured?", err)
	}

	lines := parseLines(string(out), "")
	if len(lines) == 0 {
		fmt.Println("No gcloud configurations found. Create one with: gcloud config configurations create <name>")
		os.Exit(0)
	}

	items := buildNumberedMap(lines)

	contextsTable := tablewriter.NewTable(os.Stdout)
	contextsTable.Header("", "Configurations")
	var keys []int
	for k := range items {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		contextsTable.Append(strconv.Itoa(k), items[k][0])
	}
	contextsTable.Render()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Configuration to use? ")
	input, _ := reader.ReadString('\n')

	selection, err := validateSelection(input, items)
	if err != nil {
		log.Fatal(err)
	}
	return selection
}

// BytesToString converts bytes to a string
func BytesToString(data []byte) string {
	return string(data[:])
}

// parseLines splits command output into non-empty lines, optionally stripping a prefix.
func parseLines(output string, stripPrefix string) []string {
	lines := strings.Split(output, "\n")
	// remove trailing empty element from split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if stripPrefix != "" {
		for i, l := range lines {
			lines[i] = strings.Replace(l, stripPrefix, "", -1)
		}
	}
	return lines
}

// buildNumberedMap converts a slice of strings into a 1-indexed numbered map.
func buildNumberedMap(lines []string) map[int][]string {
	m := map[int][]string{}
	for i, l := range lines {
		m[i+1] = []string{l}
	}
	return m
}

// validateSelection parses user input and looks it up in the numbered map.
func validateSelection(input string, items map[int][]string) (string, error) {
	choice, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return "", fmt.Errorf("invalid selection: %v", err)
	}
	val, ok := items[choice]
	if !ok {
		return "", fmt.Errorf("selection %d is out of range (1-%d)", choice, len(items))
	}
	return val[0], nil
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
		log.Fatalf("Failed to get secret %q: %v", defaultSecret, err)
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
		log.Fatalf("Failed to get default service account: %v\nIs your kubectl context connected to a cluster?", err)
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
		log.Fatalf("Failed to switch to context %q: %v", context, err)
	}
	fmt.Println(string(out))
}

func deleteContext(context string, kubeCtl string) {
	out, err := exec.Command(kubeCtl, "config", "delete-context", context).Output()
	if err != nil {
		log.Fatalf("Failed to delete context %q: %v", context, err)
	}
	fmt.Println(string(out))
}

func getContexts(kubeCtl string) string {
	out, err := exec.Command(kubeCtl, "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		log.Fatalf("Failed to get kubectl contexts: %v\nIs kubectl configured?", err)
	}

	lines := parseLines(string(out), "")
	if len(lines) == 0 {
		fmt.Println("No kubectl contexts found. Add one with: kubectl config set-context <name>")
		os.Exit(0)
	}

	items := buildNumberedMap(lines)

	contextsTable := tablewriter.NewTable(os.Stdout)
	contextsTable.Header("", "Contexts")
	var keys []int
	for k := range items {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		contextsTable.Append(strconv.Itoa(k), items[k][0])
	}
	contextsTable.Render()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Context to use? ")
	input, _ := reader.ReadString('\n')

	selection, err := validateSelection(input, items)
	if err != nil {
		log.Fatal(err)
	}
	return selection
}
