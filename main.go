package main

import (
	"fmt"
	kubexposev1 "kubexpose/pkg/apis/kubexpose/v1"
	kubexposeclientset "kubexpose/pkg/client/clientset/versioned"
	kubexposeinformer "kubexpose/pkg/client/informers/externalversions/kubexpose/v1"
	kubexposev1lister "kubexpose/pkg/client/listers/kubexpose/v1"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"os"

	log "github.com/Sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

var kubexposeClient *kubexposeclientset.Clientset
var k8sClient *kubernetes.Clientset

func main() {
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"

	// create the config from the path
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Fatalf("getClusterConfig: %v", err)
	}

	// generate the client based off of the config
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("getClusterConfig: %v", err)
	}

	kubexposeClient, err = kubexposeclientset.NewForConfig(config)

	if err != nil {
		log.Fatalf("getClusterConfig: %v", err)
	}

	log.Info("Successfully constructed k8s client")

	informer := kubexposeinformer.NewKubexposeInformer(kubexposeClient, meta_v1.NamespaceAll, 0, cache.Indexers{})
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer.AddEventHandler(kubexposehandler{q: queue})

	log.Info("informer and queue ready")

	defer queue.ShutDown()
	defer utilruntime.HandleCrash()
	stopChan := make(chan struct{})

	log.Info("starting informer")

	go informer.Run(stopChan)
	if !cache.WaitForCacheSync(stopChan, func() bool { return informer.HasSynced() }) {
		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
		return
	}
	log.Info("cache sync complete")
	wait.Until(processor{q: queue, i: informer}.process, time.Second, stopChan)

}

type processor struct {
	q workqueue.RateLimitingInterface
	i cache.SharedIndexInformer
}

func (p processor) process() {
	for {
		log.Info("processing next item START: ")
		key, quit := p.q.Get()
		if quit {
			log.Info("processing halted")
		}
		defer p.q.Done(key)
		keyRaw := key.(string)
		_, exists, err := p.i.GetIndexer().GetByKey(keyRaw)

		if err != nil {
			if p.q.NumRequeues(key) < 5 {
				log.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", key, err)
				p.q.AddRateLimited(key)
			} else {
				log.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", key, err)
				p.q.Forget(key)
				utilruntime.HandleError(err)
			}
		}

		if !exists {
			log.Infof("Controller.processNextItem: kubexpose deleted detected: %s", keyRaw)
			log.Info("will delete corresponding ngrok deployment ")
			p.q.Forget(key)
		} else {
			log.Infof("Controller.processNextItem: kubexpose created detected: %s", keyRaw)

			// Get the Foo resource with this namespace/name
			lister := kubexposev1lister.NewKubexposeLister(p.i.GetIndexer())
			_, name, err := cache.SplitMetaNamespaceKey(keyRaw)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
				return
			}

			log.Info("trying to list kubexpose resource for ", name)

			kubexposeResource, err := lister.Kubexposes("default").Get(name)
			if err != nil {
				// The Foo resource may no longer exist, in which case we stop
				// processing.
				if errors.IsNotFound(err) {
					utilruntime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
					return
				}

				return
			}
			log.Info("will create corresponding ngrok deployment ")
			createDeployment(kubexposeResource)
			p.q.Forget(key)
		}

		//time.Sleep(10 * time.Second)
		log.Info("processing next item END: ")

	}
}

type kubexposehandler struct {
	q workqueue.RateLimitingInterface
}

func (k kubexposehandler) OnAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	log.Infof("Add kubexpose: %s", key)
	if err == nil {
		// add the key to the queue for the handler to get
		k.q.Add(key)
	}
}

func (k kubexposehandler) OnUpdate(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	log.Infof("Update kubexpose: %s", key)
	if err == nil {
		k.q.Add(key)
	}
}

func (k kubexposehandler) OnDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	log.Infof("Delete kubexpose: %s", key)
	if err == nil {
		k.q.Add(key)
	}
}

const crd_Kind = "Kubexpose"

func createDeployment(ke *kubexposev1.Kubexpose) {
	port := strconv.Itoa(int(*ke.Spec.Port))
	label := ke.Spec.ServiceName + "-" + port
	ngrokDeploymentName := label
	numReplicas := int32(1)
	ngrokDeployment := &appsv1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      ngrokDeploymentName,
			Namespace: "default",
			OwnerReferences: []meta_v1.OwnerReference{
				*meta_v1.NewControllerRef(ke, kubexposev1.SchemeGroupVersion.WithKind(crd_Kind)),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numReplicas,
			Selector: &meta_v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": label,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{
					Labels: map[string]string{
						"app": label,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "ngrok",
							Image:   "wernight/ngrok",
							Command: []string{"ngrok"},
							Args:    []string{"http", ke.Spec.ServiceName + ":" + port},
							Ports:   []corev1.ContainerPort{corev1.ContainerPort{ContainerPort: 4040}},
						},
					},
				},
			},
		},
	}

	deployment, err := k8sClient.AppsV1().Deployments("default").Create(ngrokDeployment)
	if err != nil {
		log.Error("unable to create deployment", ngrokDeploymentName)
		return
	}

	log.Info("created deployment", deployment.GetName())
}
