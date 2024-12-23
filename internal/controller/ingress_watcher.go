package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/quinnovator/easy-tunnel-lb/internal/k8s"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// TunnelAnnotation is the annotation key for enabling tunnel load balancing
	TunnelAnnotation = "easy-tunnel-lb.quinnovator.com/enabled"
)

// IngressWatcher watches Kubernetes Ingress resources
type IngressWatcher struct {
	k8sClient  *k8s.Client
	reconciler *Reconciler
	logger     *utils.Logger
	workqueue  workqueue.RateLimitingInterface
}

// NewIngressWatcher creates a new ingress watcher
func NewIngressWatcher(k8sClient *k8s.Client, reconciler *Reconciler, logger *utils.Logger) *IngressWatcher {
	return &IngressWatcher{
		k8sClient:  k8sClient,
		reconciler: reconciler,
		logger:     logger,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Ingresses"),
	}
}

// Start begins watching Ingress resources
func (w *IngressWatcher) Start(ctx context.Context) error {
	defer w.workqueue.ShutDown()

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return w.k8sClient.ListIngresses(ctx, "")
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return w.k8sClient.WatchIngresses(ctx, "")
			},
		},
		&networkingv1.Ingress{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.handleIngress(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			w.handleIngress(new)
		},
		DeleteFunc: func(obj interface{}) {
			w.handleIngressDelete(obj)
		},
	})

	go informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return fmt.Errorf("failed to sync informer cache")
	}

	go wait.Until(w.runWorker, time.Second, ctx.Done())

	<-ctx.Done()
	return nil
}

func (w *IngressWatcher) runWorker() {
	for w.processNextWorkItem() {
	}
}

func (w *IngressWatcher) processNextWorkItem() bool {
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

		ingress, err := w.k8sClient.GetIngress(context.Background(), namespace, name)
		if err != nil {
			return fmt.Errorf("failed to get ingress: %w", err)
		}

		if err := w.reconciler.Reconcile(context.Background(), ingress); err != nil {
			return fmt.Errorf("failed to reconcile ingress: %w", err)
		}

		w.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Error processing ingress")
		w.workqueue.AddRateLimited(obj)
		return true
	}

	return true
}

func (w *IngressWatcher) handleIngress(obj interface{}) {
	ingress := obj.(*networkingv1.Ingress)
	if _, ok := ingress.Annotations[TunnelAnnotation]; !ok {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Error creating key for ingress")
		return
	}

	w.workqueue.Add(key)
}

func (w *IngressWatcher) handleIngressDelete(obj interface{}) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			w.logger.Error("Error decoding object, invalid type")
			return
		}
		ingress, ok = tombstone.Obj.(*networkingv1.Ingress)
		if !ok {
			w.logger.Error("Error decoding object tombstone, invalid type")
			return
		}
	}

	if _, ok := ingress.Annotations[TunnelAnnotation]; !ok {
		return
	}

	if err := w.reconciler.HandleDelete(context.Background(), ingress); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Error handling ingress deletion")
	}
} 