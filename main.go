package main

//import "gopkg.in/yaml.v2"
import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	//"github.com/prometheus/alertmanager/config"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	typeAnnotationKey = "alertmanager-type"
	specAnnotationKey = "spec"
	configFileKey     = "alertmanager.yml"
	routeDefaultKey   = "alertmanager-default-route"
)

type (
	controller struct {
		client          *k8sClient
		targetNamespace string
		targetName      string
		selector        string
		namespaces      []string
	}
)

var rootCmd = &cobra.Command{
	Use:   "alertmanager-config-controller [target-namespace] [target-name]",
	Short: "Collects alertmanager configs as defined in configmaps and generates a single config",
	Run:   runController,
}

var (
	selector, endpoint string
	namespaces         []string
	onetime            bool
	syncInterval       time.Duration
)

func main() {
	rootCmd.PersistentFlags().StringVarP(&selector, "selector", "s", "", "label selector")
	rootCmd.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "http://127.0.0.1:8001", "kubernetes endpoint")
	rootCmd.PersistentFlags().StringArrayVarP(&namespaces, "namespace", "n", nil, "namespace to query. can be used multiple times. default is all namespaces")
	rootCmd.PersistentFlags().BoolVarP(&onetime, "onetime", "o", false, "run one time and exit.")
	rootCmd.PersistentFlags().DurationVarP(&syncInterval, "sync-interval", "i", (60 * time.Second), "the time duration between processing.")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runController(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		log.Fatal("namespace and name of target configmap is required")
	}

	if len(namespaces) == 0 {
		namespaces = append(namespaces, "")
	}
	c := &controller{
		client:          newk8sClient(endpoint),
		selector:        selector,
		namespaces:      namespaces,
		targetNamespace: args[0],
		targetName:      args[1],
	}

	log.Println("Starting configmap-aggregator...")

	if err := c.client.waitForKubernetes(); err != nil {
		log.Fatal(err)
	}

	if onetime {
		if err := c.process(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	go func() {
		wg.Add(1)
		for {
			if err := c.process(); err != nil {
				log.Printf("failed to process config maps: %v", err)
			}
			// TODO: info level?
			//else {
			//	log.Printf("configmap aggregation complete. Next sync in %v seconds.", syncInterval.Seconds())
			//}
			select {
			case <-time.After(syncInterval):
			case <-done:
				wg.Done()
				return
			}
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	log.Printf("Shutdown signal received, exiting...")
	close(done)
	wg.Wait()
	os.Exit(0)
}

func hashConfigMap(cm *ConfigMap) string {
	h := fnv.New64()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}

	// we only hash the data for now
	printer.Fprintf(h, "%#v", cm.Data)
	return hex.EncodeToString(h.Sum(nil))
}

// true if they are the same
func compareConfigMaps(a, b *ConfigMap) bool {
	return hashConfigMap(a) == hashConfigMap(b)
}

func readObject(cm *ConfigMap, o interface{}) (bool, error) {
	data := cm.Data[specAnnotationKey]
	if data == "" {
		log.Printf("no %s key for %s/%s", specAnnotationKey, cm.Metadata.Namespace, cm.Metadata.Name)
		return false, nil
	}
	err := yaml.Unmarshal([]byte(data), o)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse '%s' data for %s/%s", specAnnotationKey, cm.Metadata.Namespace, cm.Metadata.Name)
	}
	return true, nil
}

func (c *controller) process() error {
	cm, err := c.createConfigMap()
	if err != nil {
		return err
	}
	return c.upsertConfigMap(cm)
}

var routes []*Route
var defaultRoute *Route

func (c *controller) createConfigMap() (*ConfigMap, error) {
	cfg := Config{}

	for _, n := range c.namespaces {
		list, err := c.client.getConfigMaps(n, selector)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get config maps for %s %s", n, c.selector)
		}

	ITEMS:
		for i := range list.Items {
			cm := &list.Items[i]
			if cm.Metadata.Namespace == c.targetNamespace && cm.Metadata.Name == c.targetName {
				continue ITEMS
			}

			// XXX: this is clumsy
			switch strings.ToLower(cm.Metadata.Annotations[typeAnnotationKey]) {
			case "global":
				var g GlobalConfig
				rc, err := readObject(cm, &g)
				if err != nil {
					return nil, err
				}

				if rc {
					cfg.Global = &g
				}
			case "inhibitrule":
				var r InhibitRule
				rc, err := readObject(cm, &r)
				if err != nil {
					return nil, err
				}
				if rc {
					cfg.InhibitRules = append(cfg.InhibitRules, &r)
				}
			case "receiver":
				var r Receiver
				rc, err := readObject(cm, &r)
				if err != nil {
					return nil, err
				}
				if rc {
					cfg.Receivers = append(cfg.Receivers, &r)
				}

			case "template":
				var t string
				rc, err := readObject(cm, &t)
				if err != nil {
					return nil, err
				}

				if rc && t != "" {
					cfg.Templates = append(cfg.Templates, t)
				}

			case "route":
				var r Route
				rc, err := readObject(cm, &r)
				if err != nil {
					return nil, err
				}

				if rc {
					if len(r.Routes) > 0 {
						log.Printf("route %s/%s has child routes defined, they will be ignored", cm.Metadata.Namespace, cm.Metadata.Name)
					}
					if cm.Metadata.Annotations[routeDefaultKey] == "true" {
						if defaultRoute != nil {
							log.Printf("default route already set by %s/%s sets it again", cm.Metadata.Namespace, cm.Metadata.Name)
						}
						defaultRoute = &r
					} else {
						routes = append(routes, &r)
					}
				}
			}
		}
	}

	if defaultRoute == nil {
		return nil, errors.New("no default route found")
	}

	defaultRoute.Routes = routes
	cfg.Route = defaultRoute

	cm := newConfigMap(c.targetNamespace, c.targetName)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}
	cm.Data[configFileKey] = string(data)

	fmt.Println(cm.Data[configFileKey])
	return cm, nil
}

func (c *controller) upsertConfigMap(cm *ConfigMap) error {
	existing, err := c.client.getConfigMap(c.targetNamespace, c.targetName)
	if err == ErrNotExist {
		return c.client.createConfigMap(cm)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to get config map %s/%s", c.targetNamespace, c.targetName)
	}

	//copy labels, annotations, and version
	for k, v := range existing.Metadata.Annotations {
		cm.Metadata.Annotations[k] = v
	}
	for k, v := range existing.Metadata.Labels {
		cm.Metadata.Labels[k] = v
	}
	cm.Metadata.ResourceVersion = existing.Metadata.ResourceVersion

	// XXX: unset fields on existing that will cause to not match
	// currently we don't unmarshal any

	if compareConfigMaps(existing, cm) {
		return nil
	}
	return c.client.updateConfigMap(cm)
}
