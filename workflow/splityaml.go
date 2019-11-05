package workflow

import (
	"regexp"
	"strings"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
)

var yamlSeparator = regexp.MustCompile("\\n---")

// SplitYAMLFile is a helper to split a body into multiple workflow objects
func SplitYAMLFile(body []byte) ([]wfv1.Workflow, error) {
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
