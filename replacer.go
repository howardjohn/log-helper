package main

import (
	"path/filepath"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Replacer interface {
	Replace(s string) string
}

type KubeReplacer struct {
	*strings.Replacer
	mu           sync.RWMutex
	replacements map[string]string
}

var _ Replacer = &KubeReplacer{}

func NewKubeReplacer() (*KubeReplacer, error) {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	r := &KubeReplacer{
		replacements: map[string]string{},
		Replacer:     strings.NewReplacer(),
	}
	factory := informers.NewSharedInformerFactory(client, 0)
	factory.Core().V1().Nodes().Informer().AddEventHandler(r.ObjectHandler(func(o runtime.Object) map[string]string {
		n := o.(*v1.Node)
		m := map[string]string{}
		for _, a := range n.Status.Addresses {
			if a.Type == v1.NodeInternalIP {
				m[a.Address] = n.Name
			}
		}
		return m
	}))
	factory.Core().V1().Services().Informer().AddEventHandler(r.ObjectHandler(func(o runtime.Object) map[string]string {
		s := o.(*v1.Service)
		m := map[string]string{}
		if s.Spec.ClusterIP != "None" && s.Spec.ClusterIP != "" {
			m[s.Spec.ClusterIP] = s.Name
		}
		for _, a := range s.Status.LoadBalancer.Ingress {
			m[a.IP] = s.Name + "-loadbalancer"
		}
		return m
	}))
	factory.Core().V1().Pods().Informer().AddEventHandler(r.ObjectHandler(func(o runtime.Object) map[string]string {
		p := o.(*v1.Pod)
		return map[string]string{
			p.Status.PodIP: p.Name,
		}
	}))
	stop := make(chan struct{})
	factory.Start(stop)
	factory.WaitForCacheSync(stop)
	return r, nil
}

func (kr *KubeReplacer) Replace(s string) string {
	kr.mu.RLock()
	repl := kr.Replacer
	kr.mu.RUnlock()
	return repl.Replace(s)
}

func (kr *KubeReplacer) ObjectHandler(extract func(o runtime.Object) map[string]string) cache.ResourceEventHandler {

	single := func(obj interface{}) {
		o := extractObject(obj)
		if o == nil {
			return
		}
		kr.handle(extract(o))
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: single,
		UpdateFunc: func(oldInterface, newInterace interface{}) {
			oldObj := extractObject(oldInterface)
			if oldObj == nil {
				return
			}
			newObj := extractObject(newInterace)
			if newObj == nil {
				return
			}
			kr.handle(extract(newObj))
		},
		DeleteFunc: single,
	}
}

func (kr *KubeReplacer) handle(extract map[string]string) {
	kr.mu.RLock()
	update := false
	for k, v := range extract {
		if kr.replacements[k] != v {
			update = true
			break
		}
	}
	kr.mu.RUnlock()
	if !update {
		return
	}
	kr.mu.Lock()
	defer kr.mu.Unlock()
	for k, v := range extract {
		kr.replacements[k] = v
	}
	kvlist := make([]string, 0, len(kr.replacements)*2)
	for k, v := range kr.replacements {
		kvlist = append(kvlist, k, v)
	}
	// TODO: better ordering? Key is IP so probably no risk for now
	kr.Replacer = strings.NewReplacer(kvlist...)
}

func extractObject(obj interface{}) runtime.Object {
	o, ok := obj.(runtime.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return nil
		}
		o, ok = tombstone.Obj.(runtime.Object)
		if !ok {
			return nil
		}
	}
	return o
}
