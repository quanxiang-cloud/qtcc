/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package injector

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/quanxiang-cloud/qtcc/pkg/apis/qtcc/v1alpha1"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *injector) getDefault(ctx context.Context, namespace string) (*v1alpha1.ConfigList, error) {
	cc := i.client.QtccV1alpha1().Configs(namespace)

	return cc.List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=default", v1alpha1.DefaultLable),
	})
}

func (i *injector) getPatchOperations(ctx context.Context, ar *v1.AdmissionReview) ([]PatchOperation, error) {
	configList, err := i.getDefault(ctx, ar.Request.Namespace)
	if err != nil {
		return nil, err
	}

	executor := &v1alpha1.Executor{}
	err = json.Unmarshal(ar.Request.Object.Raw, executor)
	if err != nil {
		return nil, err
	}

	var has = func(kind string, configs []v1alpha1.ConfigCite) bool {
		for _, config := range configs {
			if config.Kind == kind {
				return true
			}
		}
		return false
	}

	cs := make([]v1alpha1.ConfigCite, 0)
	for _, item := range configList.Items {
		if !has(item.Spec.Kind, executor.Spec.Configs) {
			log.Info("append config", "kind", item.Spec.Kind)
			cs = append(cs, v1alpha1.ConfigCite{
				Kind:       item.Spec.Kind,
				ConfigName: item.Name,
			})
		}
	}

	patchOpt := make([]PatchOperation, 0)
	if len(executor.Spec.Configs) == 0 {
		return append(patchOpt, PatchOperation{
			Op:   "add",
			Path: "/spec",
			Value: v1alpha1.ExecutorSpec{
				Configs: cs,
			},
		}), nil
	}

	for _, cite := range cs {
		patchOpt = append(patchOpt, PatchOperation{
			Op:    "add",
			Path:  "/spec/configs/-",
			Value: cite,
		})
	}

	return patchOpt, nil
}
