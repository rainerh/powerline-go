package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"fmt"

	"gopkg.in/yaml.v2"
)

const K8sShowCluster = "K8S_SHOW_CLUSTER"
const K8sShowContext = "K8S_SHOW_CONTEXT"
const K8sShowNamespace = "K8S_SHOW_NAMESPACE"
const K8sShowUser = "K8S_SHOW_USER"

type KubeContext struct {
	Context struct {
		Cluster   string
		Namespace string
		User      string
	}
	Name string
}

type KubeConfig struct {
	Contexts       []KubeContext `yaml:"contexts"`
	CurrentContext string        `yaml:"current-context"`
}

func homePath() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	}
	return os.Getenv(env)
}

func readKubeConfig(config *KubeConfig, path string) (err error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return
	}
	fileContent, err := ioutil.ReadFile(absolutePath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(fileContent, config)
	if err != nil {
		return
	}

	return
}

func segmentKube(p *powerline) {
	paths := append(strings.Split(os.Getenv("KUBECONFIG"), ":"), path.Join(homePath(), ".kube", "config"))
	config := &KubeConfig{}
	for _, configPath := range paths {
		temp := &KubeConfig{}
		if readKubeConfig(temp, configPath) == nil {
			config.Contexts = append(config.Contexts, temp.Contexts...)
			if config.CurrentContext == "" {
				config.CurrentContext = temp.CurrentContext
			}
		}
	}

	if config.CurrentContext != "" {
		cluster := ""
		namespace := ""
		user := ""

		for _, context := range config.Contexts {
			if context.Name == config.CurrentContext {
				cluster = context.Context.Cluster
				namespace = context.Context.Namespace
				user = context.Context.User
				break
			}
		}

		// When you use gke your clusters may look something like gke_projectname_availability-zone_cluster-01
		// instead I want it to read as `cluster-01`
		// So we remove the first 3 segments of this string, if the flag is set, and there are enough segments
		if strings.HasPrefix(cluster, "gke") && *p.args.ShortenGKENames {
			segments := strings.Split(cluster, "_")
			if len(segments) > 3 {
				cluster = strings.Join(segments[3:], "_")
			}
		}

		// With AWS EKS, cluster names are ARNs; it makes more sense to shorten them
		// so "eks-infra" instead of "arn:aws:eks:us-east-1:XXXXXXXXXXXX:cluster/eks-infra
		const arnRegexString string = "^arn:aws:eks:[[:alnum:]-]+:[[:digit:]]+:cluster/(.*)$"
		arnRe := regexp.MustCompile(arnRegexString)

		if arnMatches := arnRe.FindStringSubmatch(cluster); arnMatches != nil && *p.args.ShortenEKSNames {
			cluster = arnMatches[1]
		}

		p.appendSegment("kube-icon", segment{
			content:    fmt.Sprintf("⎈"),
			foreground: p.theme.KubeIconFg,
			background: p.theme.KubeIconBg,
		})

		// Only draw the icon once
		if cluster != "" && getPreference(K8sShowCluster, p.theme.KubeShowCluster) {
			p.appendSegment("kube-cluster", segment{
				content:    cluster,
				foreground: p.theme.KubeClusterFg,
				background: p.theme.KubeClusterBg,
			})
		}

		if getPreference(K8sShowContext, p.theme.KubeShowNamespace) {
			content := config.CurrentContext
			if user != "" && user != content && getPreference(K8sShowUser, p.theme.KubeShowUser) {
				content = fmt.Sprintf("%s@%s", user, content)
			}
			p.appendSegment("kube-context", segment{
				content:    content,
				foreground: p.theme.KubeContextFg,
				background: p.theme.KubeContextBg,
			})
		}

		if namespace != "" && getPreference(K8sShowNamespace, p.theme.KubeShowNamespace) {
			content := namespace
			p.appendSegment("kube-namespace", segment{
				content:    content,
				foreground: p.theme.KubeNamespaceFg,
				background: p.theme.KubeNamespaceBg,
			})
		}
	}
}

func getPreference(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if s, err := strconv.ParseBool(value); err == nil {
			return s
		}
	}
	return fallback
}
