package main

import (
	"bufio"
	"encoding/json"
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
	kubeCtl = "kubectl"
	gcloud  = "gcloud"
)

func main() {

	args := os.Args

	// print usage
	if len(args[1:]) < 1 {
		fmt.Println("Usage: ", args[0], `
		 -cc (change context)
		 -cn (change namespace)
		 -cp (change project)
		 -dc (delete context)
		 -dp (delete gcloud configuration)
		 -gc (get cluster credentials)
		 -lc (list contexts)
		 -lk (list GKE clusters)
		 -ln (list namespaces)
		 -lp (list google projects)
		 -rc (rename context)
		 -rp (rename gcloud configuration)
		 -sc (show current context, namespace, and project)
		 -t (generate token)`)
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
	case "-dp":
		project := getProjects(gcloud)
		deleteProject(project, gcloud)
		printCurrentProject(gcloud)
	case "-gc":
		getClusterCredentials(gcloud)
		printCurrentCluster(kubeCtl)
	case "-t":
		generateToken(kubeCtl)
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
	case "-lk":
		listClusters(gcloud)
	case "-lp":
		printCurrentProject(gcloud)
	case "-ln":
		printNameSpaces(kubeCtl)
	case "-sc":
		showCurrent(kubeCtl, gcloud)
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

func generateToken(kubeCtl string) {
	// List service accounts and let user pick one
	out, err := exec.Command(kubeCtl, "get", "serviceaccounts", "-o", "name").Output()
	if err != nil {
		log.Fatalf("Failed to list service accounts: %v\nIs your kubectl context connected to a cluster?", err)
	}

	lines := parseLines(string(out), "serviceaccount/")
	if len(lines) == 0 {
		fmt.Println("No service accounts found in the current namespace.")
		os.Exit(0)
	}

	items := buildNumberedMap(lines)

	t := tablewriter.NewTable(os.Stdout)
	t.Header("", "Service Accounts")
	var keys []int
	for k := range items {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		t.Append(strconv.Itoa(k), items[k][0])
	}
	t.Render()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Service account to generate token for? ")
	input, _ := reader.ReadString('\n')

	sa, err := validateSelection(input, items)
	if err != nil {
		log.Fatal(err)
	}

	token, err := exec.Command(kubeCtl, "create", "token", sa).Output()
	if err != nil {
		log.Fatalf("Failed to create token for %q: %v", sa, err)
	}
	fmt.Println("Copy and paste this:")
	fmt.Println(strings.TrimSpace(string(token)))
}

func deleteProject(name string, gcloud string) {
	out, err := exec.Command(gcloud, "config", "configurations", "delete", name, "--quiet").Output()
	if err != nil {
		log.Fatalf("Failed to delete configuration %q: %v\nNote: you cannot delete the active configuration.", name, err)
	}
	fmt.Println(string(out))
}

func showCurrent(kubeCtl string, gcloud string) {
	// current context
	ctx, err := exec.Command(kubeCtl, "config", "current-context").Output()
	if err != nil {
		fmt.Println("Context:   (none)")
	} else {
		fmt.Printf("Context:   %s", string(ctx))
	}

	// current namespace
	ns, err := exec.Command(kubeCtl, "config", "view", "--minify", "--output=jsonpath={.contexts[0].context.namespace}").Output()
	if err != nil || strings.TrimSpace(string(ns)) == "" {
		fmt.Println("Namespace: default")
	} else {
		fmt.Printf("Namespace: %s\n", strings.TrimSpace(string(ns)))
	}

	// current gcloud project
	proj, err := exec.Command(gcloud, "config", "get-value", "project").Output()
	if err != nil || strings.TrimSpace(string(proj)) == "" {
		fmt.Println("Project:   (unset)")
	} else {
		fmt.Printf("Project:   %s\n", strings.TrimSpace(string(proj)))
	}

	// current gcloud account
	acct, err := exec.Command(gcloud, "config", "get-value", "account").Output()
	if err != nil || strings.TrimSpace(string(acct)) == "" {
		fmt.Println("Account:   (unset)")
	} else {
		fmt.Printf("Account:   %s\n", strings.TrimSpace(string(acct)))
	}
}

func listClusters(gcloud string) {
	out, err := exec.Command(gcloud, "container", "clusters", "list").Output()
	if err != nil {
		log.Fatalf("Failed to list GKE clusters: %v\nIs gcloud configured with a project?", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		fmt.Println("No GKE clusters found in the current project.")
		return
	}
	fmt.Println(string(out))
}

func getClusterCredentials(gcloud string) {
	// list clusters in current project
	out, err := exec.Command(gcloud, "container", "clusters", "list", "--format=value(name,zone)").Output()
	if err != nil {
		log.Fatalf("Failed to list GKE clusters: %v\nIs gcloud configured with a project?", err)
	}

	lines := parseLines(string(out), "")
	if len(lines) == 0 {
		fmt.Println("No GKE clusters found in the current project.")
		os.Exit(0)
	}

	type clusterInfo struct {
		name string
		zone string
	}
	clusters := []clusterInfo{}
	items := map[int][]string{}
	for i, l := range lines {
		parts := strings.Fields(l)
		if len(parts) >= 2 {
			clusters = append(clusters, clusterInfo{name: parts[0], zone: parts[1]})
			items[i+1] = []string{parts[0] + " (" + parts[1] + ")"}
		}
	}

	t := tablewriter.NewTable(os.Stdout)
	t.Header("", "Cluster", "Zone")
	for i, c := range clusters {
		t.Append(strconv.Itoa(i+1), c.name, c.zone)
	}
	t.Render()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Cluster to get credentials for? ")
	input, _ := reader.ReadString('\n')

	selection, err := validateSelection(input, items)
	if err != nil {
		log.Fatal(err)
	}

	// find the matching cluster to get the zone
	var chosen clusterInfo
	for _, c := range clusters {
		if c.name+" ("+c.zone+")" == selection {
			chosen = c
			break
		}
	}

	fmt.Printf("Getting credentials for %s in %s...\n", chosen.name, chosen.zone)
	cred, err := exec.Command(gcloud, "container", "clusters", "get-credentials", chosen.name, "--zone", chosen.zone).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to get credentials: %v\n%s", err, string(cred))
	}
	fmt.Println(string(cred))
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
