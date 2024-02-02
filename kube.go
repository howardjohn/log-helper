package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/howardjohn/log-helper/pkg/color"
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
	translateIPs bool
}

var _ Replacer = &KubeReplacer{}

func NewKubeReplacer(translateIPs bool) (*KubeReplacer, error) {
	cfg := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if c := os.Getenv("KUBECONFIG"); c != "" {
		cfg = c
	}
	config, err := clientcmd.BuildConfigFromFlags("", cfg)
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
		translateIPs: translateIPs,
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
		for _, cip := range s.Spec.ClusterIPs {
			if cip != "None" && cip != "" {
				m[cip] = s.Name
			}
		}
		for _, a := range s.Status.LoadBalancer.Ingress {
			m[a.IP] = s.Name + "-loadbalancer"
		}
		return m
	}))
	factory.Core().V1().Pods().Informer().AddEventHandler(r.ObjectHandler(func(o runtime.Object) map[string]string {
		p := o.(*v1.Pod)
		if p.Spec.HostNetwork {
			// Node will find it. Just return a map of ourself so the pod name is highlighted
			return map[string]string{
				p.Name: p.Name,
			}
		}
		m := map[string]string{}
		for _, i := range p.Status.PodIPs {
			m[i.IP]= p.Name
		}
		return m
	}))
	stop := make(chan struct{})
	factory.Start(stop)
	factory.WaitForCacheSync(stop)
	return r, nil
}

func (kr *KubeReplacer) Replace(s string) string {
	if !kr.translateIPs {
		return s
	}
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
	delete(extract, "")
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
	keys := make([]string, 0, len(kr.replacements))
	for k := range kr.replacements {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		// Ensure longest IP is first
		return len(keys[i]) > len(keys[j])
	})
	kvlist := make([]string, 0, len(kr.replacements)*2)
	for _, k := range keys {
		kvlist = append(kvlist, k, kr.replacements[k])
	}
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

type MatcherProvider interface {
	GetMatchers() []*Matcher
}

type StaticMatchers struct{ Matchers []*Matcher }

func (s StaticMatchers) GetMatchers() []*Matcher {
	return s.Matchers
}

type KubeMatcher struct {
	staticMatchers  []*Matcher
	dynamicMatchers map[string]*Matcher
	replacer        *KubeReplacer
	colors          []color.Color
}

func (s *KubeMatcher) GetMatchers() []*Matcher {
	s.replacer.mu.RLock()
	uniqReplacements := make(map[string]struct{}, len(s.replacer.replacements))
	if s.replacer.translateIPs {
		for _, name := range s.replacer.replacements {
			if len(name) < 3 {
				// Too small to be useful
				continue
			}
			uniqReplacements[name] = struct{}{}
		}
	} else {
		// Add name AND IP
		for ip, name := range s.replacer.replacements {
			uniqReplacements[ip] = struct{}{}
			if len(name) < 3 {
				// Too small to be useful
				continue
			}
			uniqReplacements[name] = struct{}{}
		}
	}
	s.replacer.mu.RUnlock()
	keys := make([]string, 0, len(uniqReplacements))
	for k := range uniqReplacements {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	replacementMatchers := make([]*Matcher, 0, len(keys))
	for i, r := range keys {
		if m, f := s.dynamicMatchers[r]; f {
			replacementMatchers = append(replacementMatchers, m)
			continue
		}
		rx := compileRegex(regexp.QuoteMeta(r))
		m := &Matcher{
			r:        rx,
			variants: map[string]int{},
			color:    ExtrapolateColorList(s.colors, i+len(s.staticMatchers), len(keys)+len(s.staticMatchers)),
		}
		replacementMatchers = append(replacementMatchers, m)
		s.dynamicMatchers[r] = m
	}
	matchers := make([]*Matcher, 0, len(s.staticMatchers)+len(uniqReplacements))
	matchers = append(matchers, s.staticMatchers...)
	matchers = append(matchers, replacementMatchers...)
	return matchers
}

func NewKubeMatcher(matchers []*Matcher, replacer *KubeReplacer, colors []color.Color) *KubeMatcher {
	return &KubeMatcher{
		staticMatchers:  matchers,
		dynamicMatchers: map[string]*Matcher{},
		replacer:        replacer,
		colors:          colors,
	}
}
