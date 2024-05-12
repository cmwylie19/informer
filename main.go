package main

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
)

type PodLoggingController struct {
	clientset       *kubernetes.Clientset
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
}

// Run starts shared informers and waits for the shared informer cache to synchronize.
func (c *PodLoggingController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	return nil
}

func (c *PodLoggingController) podAdd(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.Infof("POD CREATED: %s/%s", pod.Namespace, pod.Name)

	if pod.Namespace != "default" {
		return
	}
	err := c.clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("Failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
	} else {
		klog.Infof("POD DELETED: %s/%s", pod.Namespace, pod.Name)
	}
}

func (c *PodLoggingController) podUpdate(old, new interface{}) {
	oldPod := old.(*v1.Pod)
	newPod := new.(*v1.Pod)
	klog.Infof("POD UPDATED: %s/%s %s", oldPod.Namespace, oldPod.Name, newPod.Status.Phase)
}

func (c *PodLoggingController) podDelete(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.Infof("POD DELETED: %s/%s", pod.Namespace, pod.Name)
}

func NewPodLoggingController(clientset *kubernetes.Clientset, informerFactory informers.SharedInformerFactory) *PodLoggingController {
	podInformer := informerFactory.Core().V1().Pods()

	c := &PodLoggingController{
		clientset:       clientset,
		informerFactory: informerFactory,
		podInformer:     podInformer,
	}
	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.podAdd,
			UpdateFunc: c.podUpdate,
			DeleteFunc: c.podDelete,
		},
	)
	return c
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	// Use in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to get cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create clientset: %v", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Hour*24)
	controller := NewPodLoggingController(clientset, factory)
	if err != nil {
		klog.Fatalf("Failed to create controller: %v", err)
	}

	stop := make(chan struct{})
	defer close(stop)
	if err = controller.Run(stop); err != nil {
		klog.Fatalf("Failed to run controller: %v", err)
	}
	<-stop
}
