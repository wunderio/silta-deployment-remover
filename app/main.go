package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"context"

	helmclient "github.com/mittwald/go-helm-client"

	errs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"

	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	webhooks_path   = "/webhooks"
	webhooks_secret = os.Getenv("WEBHOOKS_SECRET")
	debug           = debugEnabled()
)

var kubeconfig *string

func debugEnabled() bool {
	value, ok := os.LookupEnv("DEBUG")
	if ok {
		return value == "true"
	}
	return false
}

func removeRelease(namespace string, branchName string) {

	if namespace == "" || branchName == "" {
		log.Println("Namespace or branch name not found in request, exiting")
		return
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		// Fall back to in-cluster config
		// use token at /var/run/secrets/kubernetes.io/serviceaccount/token
		// KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined
		config, err = rest.InClusterConfig()
		if err != nil {
			// Still fails, might as well trigger panic() to fail pod
			log.Println("Error loading kubernetes cluster configuration:", err)
		}
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println("Error creating clientset:", err)
	}

	// Get pods to verify kube connection
	// pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	log.Println("Error listing pods:", err)
	// }
	// log.Printf("There are %d pods in the namespace\n", len(pods.Items))

	// Use helm via rest config
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        namespace,
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            false,
			Linting:          true,
		},
		RestConfig: config,
	}

	helmClient, err := helmclient.NewClientFromRestConf(opt)
	if err != nil {
		log.Printf("Kubernetes connection failure: %s", err)
	}
	_ = helmClient

	// Find kubernetes configmap by name
	// TODO: Change silta-release subchart, add special label or annotation to silta-release configmaps
	cm, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println("Error loading configmaps:", err)
	}

	var releasesFound = 0

	// Iterate cm.Items
	for _, cm := range cm.Items {
		if cm.Data["branchName"] == branchName {

			releasesFound++
			releaseName := cm.Labels["release"]

			log.Println("Found silta-release configmap for branchName:", branchName)
			log.Println("Release name:", cm.Labels["release"])

			// Delete helm release
			if debug {
				log.Println("Debug mode, not removing release")
			} else {
				uninstallErr := helmClient.UninstallReleaseByName(cm.Labels["release"])
				if uninstallErr != nil {
					log.Fatalf("Error removing a release:%s", uninstallErr)
				}
			}

			// Remove post-install job
			if debug {
				// List jobs
				postrelease, err := clientset.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "release=" + releaseName})
				if err != nil {
					log.Printf("Error listing post-release job: %s", err)
				} else {
					log.Printf("There are %d jobs with label %s in the namespace", len(postrelease.Items), "release="+releaseName)
				}
			} else {
				// Actually delete job
				deleteErr := clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), releaseName+"-post-release", metav1.DeleteOptions{})
				if deleteErr != nil {
					if errs.IsNotFound(deleteErr) {
						//Resource doesnt exist, skip printing a message
					} else {
						log.Printf("Cannot delete post-release job: %s", deleteErr)
					}
				}
			}

			// Find PVC's by release name label
			PVC_client := clientset.CoreV1().PersistentVolumeClaims(namespace)
			list, err := PVC_client.List(context.TODO(), metav1.ListOptions{
				LabelSelector: "release=" + releaseName,
			})
			if err != nil {
				log.Fatalf("Error getting the list of PVCs: %s", err)
			}

			// Iterate pvc's
			for _, v := range list.Items {
				log.Printf("PVC name: %s", v.Name)
				if debug {
					log.Println("  Debug mode, not removing PVC")
				} else {
					// Delete PVC's
					PVC_client.Delete(context.TODO(), v.Name, metav1.DeleteOptions{})
					log.Println("  PVC deleted:", v.Name)
				}
			}

			if debug {
				log.Printf("Debug mode, not removing release %s/%s", namespace, releaseName)
			} else {
				log.Printf("Release %s/%s removed", namespace, releaseName)
			}
		}
	}

	if releasesFound == 0 {
		log.Printf("No releases found for branch %s", branchName)
	}
}

func getBranchName(webhookData RequestData) (branch string) {

	// Github and GitLab event ref
	if webhookData.Ref != "" {
		branch = webhookData.Ref
	}

	// Azure event ref
	if len(webhookData.Resource.RefUpdates) > 0 {
		branch = webhookData.Resource.RefUpdates[0].Name
	}

	var re, _ = regexp.Compile(`^(refs\/heads\/)`)
	branch = re.ReplaceAllString(branch, "")

	re, _ = regexp.Compile(`^(refs\/)`)
	branch = re.ReplaceAllString(branch, "")

	return branch
}

func getEventType(req *http.Request, webhookData RequestData) (event string) {

	// Github event type based on header
	if req.Header.Get("x-github-event") != "" {
		event = req.Header.Get("x-github-event")
	}

	// Github push event with branch deletion
	if webhookData.Deleted && webhookData.After != "" {
		// Special commit state for when the branch was removed
		if (webhookData.Deleted) && (webhookData.After == "0000000000000000000000000000000000000000") {
			// Thread release removal
			event = "delete"
		}
	}

	// Azure event ref
	if req.Header.Get("x-vss-activityid") != "" {
		if len(webhookData.Resource.RefUpdates) > 0 {
			// Create event
			if webhookData.Resource.RefUpdates[0].OldObjectId == "0000000000000000000000000000000000000000" {
				event = "create"
			}
			// Delete event
			if webhookData.Resource.RefUpdates[0].NewObjectId == "0000000000000000000000000000000000000000" {
				event = "delete"
			}
		}
	}

	return event
}

func getRepositoryName(webhookData RequestData) (repository string) {

	// Github and GitLab event repository name
	if webhookData.Repository.Name != "" {
		repository = webhookData.Repository.Name
	}

	// Azure event repository name
	if webhookData.Resource.Repository.Name != "" {
		repository = webhookData.Resource.Repository.Name
	}

	return repository
}

func isValidSignature(req *http.Request, key string) bool {

	var body []byte

	// Assuming a non-empty header
	gotHash := strings.SplitN(req.Header.Get("X-Hub-Signature"), "=", 2)
	if gotHash[0] != "sha1" {
		return false
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Cannot read the request body: %s\n", err)
		return false
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	hash := hmac.New(sha1.New, []byte(key))
	if _, err := hash.Write(body); err != nil {
		log.Printf("Cannot compute the HMAC for request: %s\n", err)
		return false
	}

	// TODO: signature256

	expectedHash := hex.EncodeToString(hash.Sum(nil))

	// Allow invalid signatures in debug mode
	if debug {
		log.Println("Debug mode, allowing invalid signature")
		log.Println("EXPECTED HASH:", expectedHash)
		log.Println("GOT HASH:     ", gotHash[1])
		return true
	}

	return gotHash[1] == expectedHash
}

func handleWebhook(w http.ResponseWriter, req *http.Request) {

	log.Println("Received webhook request ...")
	w.Header().Set("Content-Type", "application/json")

	signature := req.Header.Get("x-hub-signature")
	signature256 := req.Header.Get("x-hub-signature-256")

	// Validate Github signature
	if signature != "" || signature256 != "" {
		log.Println("Processing github request")

		// Check signature
		if isValidSignature(req, webhooks_secret) {
			fmt.Println("Github signature is valid")
		} else {
			fmt.Println("Github signature is invalid. You might need to switch deliveries to application/json.")
			return
		}
	} else {
		// Fall back to basic auth and validate it
		// Azure DevOps provides this
		_, password, ok := req.BasicAuth()
		if ok {
			// Calculate SHA-256 hashes for the provided and expected
			passwordHash := sha256.Sum256([]byte(password))
			expectedPasswordHash := sha256.Sum256([]byte(webhooks_secret))
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)
			// Compare hashes
			if passwordMatch {
				fmt.Println("Basic authentication is valid")
			} else {
				fmt.Println("Basic authentication is invalid")
				return
			}
		} else {
			fmt.Println("Authentication is invalid")
			if debug {
				log.Println("Debug mode, skipping authentication validation")
			} else {
				return
			}
		}
	}

	// Decode request body
	var body []byte
	defer req.Body.Close()

	// Read request body
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Failed to read request body: %s", err)
		return
	}

	// Replace body with original body
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	// Marshal request body
	var webhookData RequestData
	err = json.Unmarshal(body, &webhookData)
	if err != nil {
		log.Printf("Failed to parse request body: %s", err)
	}

	var repository = getRepositoryName(webhookData)
	var branch = getBranchName(webhookData)
	var event = getEventType(req, webhookData)

	fmt.Printf("Event: %s, Repository: %s, Branch: %s \n", event, repository, branch)

	// Respond to ping event
	if event == "ping" {
		// Respond with pong
		resp := map[string]string{"message": "pong", "result": "ok"}
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	// Remove release when delete event is received
	if event == "delete" {
		// Thread release removal
		go removeRelease(repository, branch)
	}

	// Always return 200
	resp := map[string]string{"message": "ok", "result": "ok"}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

}

func main() {
	// Try reading kubeconfig
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	if debug {
		log.Println("Debug mode enabled")
	}
	log.Println("Starting webhook listener")
	http.HandleFunc(webhooks_path, handleWebhook)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
