package main

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
)

// Controller struct defines how a controller should encapsulate
// logging, client connectivity, informing (list and watching)
// queueing, and handling of resource changes
type Controller struct {
	logger    *log.Entry
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
	handler   Handler
}

var yamlSeparator = regexp.MustCompile("\\n---")

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// splitYAMLFile is a helper to split a body into multiple workflow objects
func splitYAMLFile(body []byte) ([]wfv1.Workflow, error) {
	manifestsStrings := yamlSeparator.Split(string(body), -1)
	manifests := make([]wfv1.Workflow, 0)
	for _, manifestStr := range manifestsStrings {
		if strings.TrimSpace(manifestStr) == "" {
			continue
		}
		var wf wfv1.Workflow
		err := yaml.Unmarshal([]byte(manifestStr), &wf)
		//if wf.Kind != "" && wf.Kind != workflow.Kind {
		//	// If we get here, it was a k8s manifest which was not of type 'Workflow'
		//	// We ignore these since we only care about validating Workflow manifests.
		//	continue
		//}
		if err != nil {
			return nil, errors.New(errors.CodeBadRequest, err.Error())
		}
		manifests = append(manifests, wf)
	}
	return manifests, nil
}

// Run is the main path of execution for the controller loop
func (c *Controller) Run(stopCh <-chan struct{}) {
	// handle a panic with logging and exiting
	defer utilruntime.HandleCrash()
	// ignore new items in the queue but when all goroutines
	// have completed existing items then shutdown
	defer c.queue.ShutDown()

	c.logger.Info("Controller.Run: initiating")

	cj := cron.New()
	cj.AddFunc("*/1 * * * *", func() {
		// get current user to determine home directory
		usr, err := user.Current()
		checkErr(err)

		kubeconfig := filepath.Join(usr.HomeDir, ".kube", "config")
		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		checkErr(err)

		namespace := "default"

		// create the workflow client
		wfClient := wfclientset.NewForConfigOrDie(config).ArgoprojV1alpha1().Workflows(namespace)

		body, err := ioutil.ReadFile("workflows/hello-world.yaml")
		fmt.Printf(" reading file %v", &body)
		checkErr(err)

		workFlows, err := splitYAMLFile(body)
		checkErr(err)
		// submit the hello world workflow
		wfParameters := []string{"hello=one", "bye=two", "what=is"}
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range wfParameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) != 2 {
				// Ignore invalid parameters
				continue
			}
			param := wfv1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}
		for _, param := range workFlows[0].Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}

		workFlows[0].Spec.Arguments.Parameters = newParams
		createdWf, err := wfClient.Create(&workFlows[0])

		checkErr(err)
		fmt.Printf("Workflow %s submitted\n", createdWf.Name)

		c.logger.Info("Every hour on the minute")
	})
	c.logger.Info("Start Cron scheduler")
	cj.Start()

	// run the informer to start listing and watching resources
	go c.informer.Run(stopCh)

	// do the initial synchronization (one time) to populate resources
	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
		return
	}
	c.logger.Info("Controller.Run: cache sync complete")

	// run the runWorker method every second with a stop channel
	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced allows us to satisfy the Controller interface
// by wiring up the informer's HasSynced method to it
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// runWorker executes the loop to process new items added to the queue
func (c *Controller) runWorker() {
	log.Info("Controller.runWorker: starting")

	// invoke processNextItem to fetch and consume the next change
	// to a watched or listed resource
	for c.processNextItem() {
		log.Info("Controller.runWorker: processing next item")
	}

	log.Info("Controller.runWorker: completed")
}

// processNextItem retrieves each queued item and takes the
// necessary handler action based off of if the item was
// created or deleted
func (c *Controller) processNextItem() bool {
	log.Info("Controller.processNextItem: start")

	// fetch the next item (blocking) from the queue to process or
	// if a shutdown is requested then return out of this to stop
	// processing
	key, quit := c.queue.Get()

	// stop the worker loop from running as this indicates we
	// have sent a shutdown message that the queue has indicated
	// from the Get method
	if quit {
		return false
	}

	defer c.queue.Done(key)

	// assert the string out of the key (format `namespace/name`)
	keyRaw := key.(string)

	// take the string key and get the object out of the indexer
	//
	// item will contain the complex object for the resource and
	// exists is a bool that'll indicate whether or not the
	// resource was created (true) or deleted (false)
	//
	// if there is an error in getting the key from the index
	// then we want to retry this particular queue key a certain
	// number of times (5 here) before we forget the queue key
	// and throw an error
	item, exists, err := c.informer.GetIndexer().GetByKey(keyRaw)
	if err != nil {
		if c.queue.NumRequeues(key) < 5 {
			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", key, err)
			c.queue.AddRateLimited(key)
		} else {
			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", key, err)
			c.queue.Forget(key)
			utilruntime.HandleError(err)
		}
	}

	// if the item doesn't exist then it was deleted and we need to fire off the handler's
	// ObjectDeleted method. but if the object does exist that indicates that the object
	// was created (or updated) so run the ObjectCreated method
	//
	// after both instances, we want to forget the key from the queue, as this indicates
	// a code path of successful queue key processing
	if !exists {
		c.logger.Infof("Controller.processNextItem: object deleted detected: %s", keyRaw)
		c.handler.ObjectDeleted(item)
		c.queue.Forget(key)
	} else {
		c.logger.Infof("Controller.processNextItem: object created detected: %s", keyRaw)
		c.handler.ObjectCreated(item)
		c.queue.Forget(key)
	}

	// keep the worker loop running by returning true
	return true
}
