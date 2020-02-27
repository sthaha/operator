package v1alpha1

import (
	mf "github.com/jcrossley3/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("extn").WithName("registry")

// +kxxx8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Extension interface {
	Transformer(s *runtime.Scheme) mf.Transformer
}

// RegistryExtension defines image overrides of tekton images.
// The override values are specific to each tekton deployment.
// +kxx8s:openapi-gen=true
type RegistryExtension struct {
	// A map of a container name or arg key to the full image location of
	// the individual tekton container.
	Override map[string]string `json:"override,omitempty"`
}

func (r *RegistryExtension) Transformer(scheme *runtime.Scheme) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" {
			return nil
		}
		var deploy = &appsv1.Deployment{}
		if err := scheme.Convert(u, deploy, nil); err != nil {
			return err
		}
		//registry := r.Registry
		err := r.UpdateDeployment(deploy)
		if err != nil {
			return err
		}
		return scheme.Convert(deploy, u, nil)
	}
}

func (reg *RegistryExtension) UpdateDeployment(deploy *appsv1.Deployment) error {
	containers := deploy.Spec.Template.Spec.Containers
	for index := range containers {
		container := &containers[index]
		log := log.WithValues("container", container.Name)
		log.V(1).Info("Processing")

		override, ok := reg.Override[container.Name]
		if !ok {
			continue
		}

		if override == "" {
			log.Info("skipping invalid empty override")
		}

		log.Info("Updating container image", "new", override)
		container.Image = override

		// replace image in args
		args := container.Args
		for i, v := range args {
			if img, ok := reg.Override[v]; ok && img != "" {
				args[i] = img
			}
		}
	}

	return nil
}
