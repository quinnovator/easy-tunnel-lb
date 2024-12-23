package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// TunnelAnnotation is the annotation key for enabling tunnel load balancing on a Service
	TunnelAnnotation = "easy-tunnel-lb.quinnovator.com/enabled"
)

// K8sServiceClient interface for Kubernetes operations on Services
type K8sServiceClient interface {
	ListServices(ctx context.Context, namespace string, opts metav1.ListOptions) (*v1.ServiceList, error)
	WatchServices(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error)
	GetService(ctx context.Context, namespace, name string) (*v1.Service, error)
	SetServiceLoadBalancer(ctx context.Context, svc *v1.Service, externalIP, externalHost string) error
}

// ServiceWatcher watches Kubernetes Services for LoadBalancer type
type ServiceWatcher struct {
	k8sClient  K8sServiceClient
	reconciler *ServiceReconciler
	logger     *utils.Logger
	workqueue  workqueue.RateLimitingInterface
}

// NewServiceWatcher creates a new ServiceWatcher
func NewServiceWatcher(k8sClient K8sServiceClient, reconciler *ServiceReconciler, logger *utils.Logger) *ServiceWatcher {
	return &ServiceWatcher{
		k8sClient:  k8sClient,
		reconciler: reconciler,
		logger:     logger,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
	}
}

// Start begins watching Service resources
func (w *ServiceWatcher) Start(ctx context.Context) error {
	defer w.workqueue.ShutDown()

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return w.k8sClient.ListServices(ctx, "", options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return w.k8sClient.WatchServices(ctx, "", options)
			},
		},
		&v1.Service{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.handleService(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.handleService(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			w.handleServiceDelete(obj)
		},
	})

	go informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return fmt.Errorf("failed to sync service informer cache")
	}

	go wait.Until(w.runWorker, time.Second, ctx.Done())

	<-ctx.Done()
	return nil
}

func (w *ServiceWatcher) runWorker() {
	for w.processNextWorkItem() {
	}
}

func (w *ServiceWatcher) processNextWorkItem() bool {
	obj, shutdown := w.workqueue.Get()
	if shutdown {
		return false
	}
	defer w.workqueue.Done(obj)

	err := func(obj interface{}) error {
		key, ok := obj.(string)
		if !ok {
			w.workqueue.Forget(obj)
			return fmt.Errorf("expected string in workqueue but got %#v", obj)
		}

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return fmt.Errorf("invalid resource key: %s", key)
		}

		svc, err := w.k8sClient.GetService(context.Background(), namespace, name)
		if err != nil {
			return fmt.Errorf("failed to get service: %w", err)
		}
		if svc == nil {
			// Service might have been deleted
			return nil
		}

		if err := w.reconciler.Reconcile(context.Background(), svc); err != nil {
			return fmt.Errorf("failed to reconcile service: %w", err)
		}

		w.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Error processing service")
		w.workqueue.AddRateLimited(obj)
		return true
	}

	return true
}

func (w *ServiceWatcher) handleService(obj interface{}) {
	svc, ok := obj.(*v1.Service)
	if !ok {
		return
	}
	// We only care about LB-type services with our annotation
	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return
	}
	if _, found := svc.Annotations[TunnelAnnotation]; !found {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Error creating key for service")
		return
	}
	w.workqueue.Add(key)
}

func (w *ServiceWatcher) handleServiceDelete(obj interface{}) {
	svc, ok := obj.(*v1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			w.logger.Error("Error decoding object, invalid type")
			return
		}
		svc, ok = tombstone.Obj.(*v1.Service)
		if !ok {
			w.logger.Error("Error decoding service tombstone, invalid type")
			return
		}
	}

	// Only handle if we had the annotation
	if svc.Spec.Type == v1.ServiceTypeLoadBalancer {
		if _, found := svc.Annotations[TunnelAnnotation]; found {
			if err := w.reconciler.HandleDelete(context.Background(), svc); err != nil {
				w.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Error("Error handling service deletion")
			}
		}
	}
}