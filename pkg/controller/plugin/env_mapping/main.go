package main

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/rigdev/rig/pkg/api/v1alpha2"
	"github.com/rigdev/rig/pkg/controller/pipeline"
	"github.com/rigdev/rig/pkg/controller/plugin"
	"github.com/rigdev/rig/pkg/controller/plugin/env_mapping/types"
	"github.com/rigdev/rig/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type envMapping struct{}

func (p *envMapping) LoadConfig(_ []byte) error {
	return nil
}

func (p *envMapping) Run(_ context.Context, req pipeline.CapsuleRequest, _ hclog.Logger) error {
	value, ok := req.Capsule().Annotations[types.AnnotationEnvMapping]
	if !ok {
		return nil
	}

	var data types.AnnotationValue
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	if err := req.GetNew(deployment); errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, source := range data.Sources {
		container := source.Container
		if container == "" {
			container = req.Capsule().GetName()
		}

		for i, c := range deployment.Spec.Template.Spec.Containers {
			if c.Name != container {
				continue
			}

			for _, m := range source.Mappings {
				envVar := corev1.EnvVar{
					Name: m.Env,
				}
				switch {
				case source.ConfigMap != "":
					envVar.ValueFrom = &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: source.ConfigMap,
							},
							Key: m.Key,
						},
					}
					req.MarkUsedResource(v1alpha2.UsedResource{
						Ref: &corev1.TypedLocalObjectReference{
							Kind: "ConfigMap",
							Name: source.ConfigMap,
						},
					})
				case source.Secret != "":
					envVar.ValueFrom = &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: source.Secret,
							},
							Key: m.Key,
						},
					}
					req.MarkUsedResource(v1alpha2.UsedResource{
						Ref: &corev1.TypedLocalObjectReference{
							Kind: "Secret",
							Name: source.Secret,
						},
					})
				}

				c.Env = append(c.Env, envVar)
			}

			deployment.Spec.Template.Spec.Containers[i] = c

			break
		}
	}

	return req.Set(deployment)
}

func main() {
	plugin.StartPlugin("rigdev.env_mapping", &envMapping{})
}
