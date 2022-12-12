package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	helmclient "github.com/mittwald/go-helm-client"

	errs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"

	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/go-playground/webhooks/v6/azuredevops"
	"github.com/go-playground/webhooks/v6/github"
)

var (
	webhooks_path   = "/webhooks"
	webhooks_secret = os.Getenv("WEBHOOKS_SECRET")
	debug           = true
)

var kubeconfig *string

func removeRelease(namespace string, branchName string) {

	log.Println("Namespace:", namespace)

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

func getBranchName(event any) (branch string) {
	// print event
	log.Printf("Event: %+v", event)

	// // Github and GitLab event ref
	// if event.Ref == "" {
	// 	branch = event.Ref
	// }
	// // Azure event ref
	// if event.Resource.RefUpdates[0].Name == "" {
	// 	branch = event.Resource.RefUpdates[0].Name
	// }

	// var re, _ = regexp.Compile(`^(refs\/heads\/)`)
	// branch = re.ReplaceAllString(branch, "")

	// re, _ = regexp.Compile(`^(refs\/)`)
	// branch = re.ReplaceAllString(branch, "")

	return branch
}

func main() {

	// TODO: Require webhook secret
	// if webhooks_secret == "" {
	// 	log.Println("Error: WEBHOOKS_SECRET is required")
	// 	os.Exit(1)
	// }

	// Try reading kubeconfig
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	log.Println("Starting webhook listener")

	// Start listener webhook
	githubhook, _ := github.New(github.Options.Secret(webhooks_secret))
	azurehook, _ := azuredevops.New()

	// Github webhook handler
	http.HandleFunc(webhooks_path, func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received webhook request ...")

		w.Header().Set("Content-Type", "application/json")

		payload, err := githubhook.Parse(r, github.CreateEvent, github.DeleteEvent, github.PushEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				// ok event wasn't one of the ones asked to be parsed
				log.Printf("Unknown event: %s", err)
			}
		}

		switch payload.(type) {

		// TODO: Webhook - repository created event (do nothing)
		// CreateEvent ?
		// RepositoryEvent ?
		// case github.CreatePayload:
		// 	event := payload.(github.CreatePayload)
		// 	// Do whatever you want from here...
		// 	fmt.Printf("%+v", event)

		// TODO: Webhook - delete event
		case github.DeletePayload:
			event := payload.(github.DeletePayload)

			var repository = event.Repository.Name
			var branch = getBranchName(event)

			// Thread release removal
			go removeRelease(repository, branch)

			resp := map[string]string{"message": "ok", "result": "ok"}
			err := json.NewEncoder(w).Encode(resp)
			if err != nil {
				http.Error(w, err.Error(), 500)
			}

		// Webhook - push event (https://developer.github.com/webhooks/#events)
		case github.PushPayload:
			event := payload.(github.PushPayload)

			var repository = event.Repository.Name
			var branch = getBranchName(event)

			if event.Deleted && event.After != "" {
				// Special commit state for when the branch was removed
				if (event.Deleted) && (event.After == "0000000000000000000000000000000000000000") {
					// Thread release removal
					go removeRelease(repository, branch)
				}
			}

			resp := map[string]string{"message": "ok", "result": "ok"}
			err := json.NewEncoder(w).Encode(resp)
			if err != nil {
				http.Error(w, err.Error(), 500)
			}

		default:
			log.Println("Unknown payload")

		}
	})

	// Azure DevOps handler
	http.HandleFunc(webhooks_path, func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received webhook request ...")

		w.Header().Set("Content-Type", "application/json")

		// payload, err := azurehook.Parse(r, azuredevops.GitPullRequestCreatedEventType)
		// if err != nil {
		// 	if err == azuredevops.ErrParsingPayload {
		// 		log.Printf("Error parsing payload: %s", err)
		// 	}
		// }

		// fmt.Printf("%+v", payload)

		// switch payload.(type) {

		// // TODO: Webhook - repository created event (do nothing)
		// // CreateEvent ?
		// // RepositoryEvent ?
		// // case github.CreatePayload:
		// // 	event := payload.(github.CreatePayload)
		// // 	// Do whatever you want from here...
		// // 	fmt.Printf("%+v", event)

		// // TODO: Webhook - delete event
		// case github.DeletePayload:
		// 	event := payload.(github.DeletePayload)

		// 	var repository = event.Repository.Name
		// 	var branch = getBranchName(event)

		// 	// Thread release removal
		// 	go removeRelease(repository, branch)

		// 	resp := map[string]string{"message": "ok", "result": "ok"}
		// 	err := json.NewEncoder(w).Encode(resp)
		// 	if err != nil {
		// 		http.Error(w, err.Error(), 500)
		// 	}

		// // Webhook - push event (https://developer.github.com/webhooks/#events)
		// case github.PushPayload:
		// 	event := payload.(github.PushPayload)

		// 	var repository = event.Repository.Name
		// 	var branch = getBranchName(event)

		// 	if event.Deleted && event.After != "" {
		// 		// Special commit state for when the branch was removed
		// 		if (event.Deleted) && (event.After == "0000000000000000000000000000000000000000") {
		// 			// Thread release removal
		// 			go removeRelease(repository, branch)
		// 		}
		// 	}

		// 	resp := map[string]string{"message": "ok", "result": "ok"}
		// 	err := json.NewEncoder(w).Encode(resp)
		// 	if err != nil {
		// 		http.Error(w, err.Error(), 500)
		// 	}

		// default:
		// 	log.Println("Unknown payload")

		// }
	})
	http.ListenAndServe(":8080", nil)
}
