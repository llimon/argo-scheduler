package workflow

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
)

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func SubmitWorkflow(workflowBody []byte) error {
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

	checkErr(err)

	workFlows, err := SplitYAMLFile(workflowBody)
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

	return nil
}
