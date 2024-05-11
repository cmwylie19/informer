package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeConfig := os.Getenv("KUBECONFIG")

	var clusterConfig *rest.Config
	var err error
	if kubeConfig != "" {
		clusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		clusterConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalln(err)
	}

	clusterClient, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatalln(err)
	}

	resource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, corev1.NamespaceAll, nil)
	informer := factory.ForResource(resource).Informer()

	mux := &sync.RWMutex{}
	synced := false
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			mux.RLock()
			defer mux.RUnlock()
			if !synced {
				return
			}

			//fmt.Printf("Pod created: %v\n", obj.(corev1.Pod).Name)
			fmt.Printf("%+v\n", obj)
		},
		// UpdateFunc: func(oldObj, newObj interface{}) {
		// 	mux.RLock()
		// 	defer mux.RUnlock()
		// 	if !synced {
		// 		return
		// 	}

		// 	fmt.Printf("Pod updated: %s/%s %s\n", oldObj.(corev1.Pod).Namespace, oldObj.(corev1.Pod).Name, newObj.(corev1.Pod).Status.Phase)
		// },
		// DeleteFunc: func(obj interface{}) {
		// 	mux.RLock()
		// 	defer mux.RUnlock()
		// 	if !synced {
		// 		return
		// 	}

		// 	// Handler logic
		// 	fmt.Printf("Pod deleted: %v\n", obj.(corev1.Pod).Name)
		// },
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go informer.Run(ctx.Done())

	isSynced := cache.WaitForCacheSync(ctx.Done(), informer.HasSynced)
	mux.Lock()
	synced = isSynced
	mux.Unlock()

	if !isSynced {
		log.Fatal("failed to sync")
	}

	<-ctx.Done()
}
