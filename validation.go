package main

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type validationViolation struct {
	TargetDesc string
	Message    string
}

type validationViolationSet struct {
	Violations []validationViolation
}

type objectValidation struct {
	Kind               string
	ObjMeta            *metav1.ObjectMeta
	ContainerResources *validationViolationSet
}

func validateContainerResources(validation *objectValidation, targetDesc string, container *corev1.Container, config *config) {
	validateResourceIsSet(validation.ContainerResources, targetDesc,
		container.Resources.Limits, "limit", corev1.ResourceCPU, config.RuleResourceLimitCPURequired)
	validateResourceIsSet(validation.ContainerResources, targetDesc,
		container.Resources.Limits, "limit", corev1.ResourceMemory, config.RuleResourceLimitMemoryRequired)
	validateResourceIsSet(validation.ContainerResources, targetDesc,
		container.Resources.Requests, "request", corev1.ResourceCPU, config.RuleResourceRequestCPURequired)
	validateResourceIsSet(validation.ContainerResources, targetDesc,
		container.Resources.Requests, "request", corev1.ResourceMemory, config.RuleResourceRequestMemoryRequired)
}

func validateResourceIsSet(violationSet *validationViolationSet, targetDesc string, resList corev1.ResourceList,
	listName string, name corev1.ResourceName, enabled bool) {
	if enabled && !isResourceSet(resList, name) {
		msg := fmt.Sprintf("'%s' resource %s must be specified.", name, listName)
		violationSet.add(validationViolation{targetDesc, msg})
	}
}

func isResourceSet(resList corev1.ResourceList, name corev1.ResourceName) bool {
	var missing = resList == nil
	if !missing {
		if _, ok := resList[name]; !ok {
			missing = true
		}
	}
	return !missing
}

func (violationSet *validationViolationSet) add(violation validationViolation) {
	violationSet.Violations = append(violationSet.Violations, violation)
}

// Returns the textual representation of a validation set. It groups
// violation messages by their target. If there are no violations, returns an
// empty string.
func (violationSet *validationViolationSet) message() string {
	m := make(map[string]string)
	targetDescs := []string{} // to keep ordering
	for _, v := range violationSet.Violations {
		if _, ok := m[v.TargetDesc]; !ok {
			targetDescs = append(targetDescs, v.TargetDesc)
			m[v.TargetDesc] = ""
		}
		m[v.TargetDesc] = strings.TrimSpace(fmt.Sprintf("%s %s ", m[v.TargetDesc], v.Message))
	}

	var message = ""
	for _, targetDesc := range targetDescs {
		message = strings.TrimSpace(fmt.Sprintf("%s %s: [%s] ", message, targetDesc, m[targetDesc]))
	}
	return message
}

func (validation *objectValidation) message(config *config) string {
	var message = ""

	containerResourcesViolationMessage := validation.ContainerResources.message()
	if len(containerResourcesViolationMessage) > 0 {
		message = fmt.Sprintf("One or more container resource specifications are invalid: [%s]", containerResourcesViolationMessage)
		if len(config.RuleResourceViolationMessage) > 0 {
			message = fmt.Sprintf("%s %s", message, config.RuleResourceViolationMessage)
		}
	}

	if len(message) > 0 && validation.ObjMeta != nil {
		message = fmt.Sprintf("Validation errors for %s '%s/%s': %s",
			validation.Kind, validation.ObjMeta.GetNamespace(), validation.ObjMeta.GetName(), message)
	}

	return message
}