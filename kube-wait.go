package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

func main() {
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		printUsage()
		os.Exit(1)
	}

	timeout := time.Minute
	timeoutStr := os.Getenv("POD_TIMEOUT")
	if i, err := strconv.Atoi(timeoutStr); err == nil {
		timeout = time.Duration(i) * time.Second
	}

	log.Printf("Waiting in namespace: %v", ns)

	client, err := client.NewInCluster()
	if err != nil {
		log.Fatalf("Failed to get connection to Kubernetes master: %v", err)
	}

	getStatus(client, ns)

	watch, err := client.Deployments(ns).Watch(api.ListOptions{})
	if err != nil {
		log.Fatalf("Could not get watcher: %v", err)
	}

	watchCh := watch.ResultChan()

	for {
		select {
		case <-watchCh:
			getStatus(client, ns)
		case <-time.After(timeout):
			getStatus(client, ns)
			log.Fatal("Deployment did not converge in time.")
		}
	}
}

func getStatus(client *client.Client, ns string) {
	pending := []string{}

	deployments, err := client.Deployments(ns).List(api.ListOptions{})
	if err != nil {
		log.Fatalf("Could not get deployments: %v", err)
	}

	total := len(deployments.Items)
	available := 0

	for _, deployment := range deployments.Items {
		if deployment.Status.AvailableReplicas == deployment.Spec.Replicas {
			available++
		} else {
			pending = append(pending, deployment.Name)
		}
	}

	if available == total {
		log.Printf("%v/%v deployments available.", available, total)
		os.Exit(0)
	}
	log.Printf("%v/%v deployments available. Still waiting for: %v", available, total, strings.Join(pending, ", "))
}

func printUsage() {
	fmt.Print(`Could not get the current namespace. Ensure POD_NAMESPACE is set. For a Kubernetes PodSpec, add:
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace`)
}
