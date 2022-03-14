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

package executor

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	qtccv1alpha1 "github.com/quanxiang-cloud/qtcc/pkg/apis/qtcc/v1alpha1"
	"github.com/quanxiang-cloud/qtcc/pkg/misc"
)

// ExecutorReconciler reconciles a Executor object
type ExecutorReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	AppsV1Client v1.AppsV1Interface
}

//+kubebuilder:rbac:groups=qtcc.quanxiang.dev,resources=executors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qtcc.quanxiang.dev,resources=executors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qtcc.quanxiang.dev,resources=executors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Executor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ExecutorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithName("Executor")

	executor := &qtccv1alpha1.Executor{}
	err := r.Get(ctx, req.NamespacedName, executor)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "get executor")
		return ctrl.Result{}, err
	}

	if executor.IsFinish() {
		return ctrl.Result{}, nil
	}

	err = r.mutate(ctx, executor)

	executor.Status.Status = corev1.ConditionTrue
	if err != nil {
		executor.Status.Status = corev1.ConditionFalse
		executor.Status.Message = err.Error()
	}

	if err = r.Status().Update(ctx, executor); err != nil {
		logger.Error(err, "update status")
		return ctrl.Result{}, err
	}
	logger.Info("Success")
	return ctrl.Result{}, nil
}

func (r *ExecutorReconciler) mutate(ctx context.Context, executor *qtccv1alpha1.Executor) error {
	logger := log.FromContext(ctx)
	logger = logger.WithName("mutate")

	for _, cf := range executor.Spec.Configs {
		conf := qtccv1alpha1.Config{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: executor.Namespace,
			Name:      cf.ConfigName,
		}, &conf)
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Info("config not exist", "configName", cf.ConfigName)
				return err
			}
			logger.Error(err, "get config", "configName", cf.ConfigName)
			return err
		}

		if !r.IsChange(conf, executor) {
			continue
		}

		svc, deployment, err := r.getRefer(ctx, executor, conf)
		if err != nil {
			logger.Error(err, "get refer", "config", cf.ConfigName)
			return err
		}

		container, err := r.mutateContainer(ctx, conf)
		if err != nil {
			return err
		}

		container.Name = cf.Kind
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)
		deployment.Spec.Template.Labels = deployment.Labels

		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: deployment.Labels,
		}

		err = r.createDeployment(ctx, deployment)
		if err != nil {
			return err
		}

		svcPorts := make([]corev1.ServicePort, 0, len(conf.Spec.Ports))
		for _, port := range conf.Spec.Ports {
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       port.Name,
					TargetPort: intstr.FromString(port.Name),
					Port:       port.ContainerPort,
					Protocol:   port.Protocol,
				})
		}

		svc.Spec.Ports = svcPorts
		err = r.createService(ctx, svc)
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO if config is delete, will delete deployment and svc ?
func (r *ExecutorReconciler) IsChange(config qtccv1alpha1.Config, executor *qtccv1alpha1.Executor) bool {
	hash := misc.Hash(config)
	if len(executor.Status.Condition) == 0 {
		executor.Status.Condition = make(map[string]qtccv1alpha1.Condition)
	}

	condition := executor.Status.Condition[config.Spec.Kind]
	if condition.Hash != hash {
		condition.Hash = hash
		executor.Status.Condition[config.Spec.Kind] = condition
		return true
	}
	return false
}

func (r *ExecutorReconciler) getRefer(ctx context.Context, executor *qtccv1alpha1.Executor, cf qtccv1alpha1.Config) (*corev1.Service, *appsv1.Deployment, error) {
	var has = func(ctx context.Context, r *ExecutorReconciler, name, namesapce string) (*corev1.Service, *appsv1.Deployment, error) {
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{Namespace: namesapce, Name: name}, deployment)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, nil, err
			}
			err = nil
			deployment = nil
		}

		svc := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Namespace: namesapce, Name: name}, svc)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, nil, err
			}
			err = nil
			svc = nil
		}

		return svc, deployment, nil
	}

	name := fmt.Sprintf("%s-%s", executor.Name, cf.Spec.Kind)
	svc, deployment, err := has(ctx, r, name, executor.Namespace)
	if err != nil {
		return nil, nil, err
	}

	labels := map[string]string{
		"qtcc.quanxiang.dev/app": executor.Name,
	}
	labels["qtcc.quanxiang.dev/kind"] = cf.Spec.Kind

	if deployment == nil {
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: executor.Namespace,
				Labels:    labels,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: make([]corev1.Container, 0),
					},
				},
			},
		}
		deployment.SetOwnerReferences(nil)
		if err := ctrl.SetControllerReference(executor, deployment, r.Scheme); err != nil {
			return nil, nil, err
		}
	}

	if svc == nil {
		svc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: executor.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: deployment.Labels,
			},
		}
		svc.SetOwnerReferences(nil)
		if err := ctrl.SetControllerReference(executor, svc, r.Scheme); err != nil {
			return nil, nil, err
		}
	}
	return svc, deployment, nil
}

func (r *ExecutorReconciler) mutateContainer(ctx context.Context, conf qtccv1alpha1.Config) (corev1.Container, error) {
	confSepc := conf.Spec

	container := corev1.Container{
		Image: confSepc.Image,
		Ports: confSepc.Ports,
	}

	for _, param := range confSepc.Params {
		if param.Scope == "env" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  param.Name,
				Value: param.Value,
			})
			continue
		}
		container.Args = append(container.Args,
			fmt.Sprintf("--%s=%s", param.Name, param.Value),
		)
	}

	return container, nil
}

func (r *ExecutorReconciler) createDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	logger := log.FromContext(ctx)
	logger = logger.WithName("createDeployment")
	var err error
	if deployment.ResourceVersion == "" {
		err = r.Create(ctx, deployment)
	} else {
		err = r.Update(ctx, deployment)
	}

	if err != nil {
		logger.Error(err, "create deployment", "name", deployment.Name)
		return err
	}
	return nil
}

func (r *ExecutorReconciler) createService(ctx context.Context, svc *corev1.Service) error {
	logger := log.FromContext(ctx)
	logger = logger.WithName("createService")
	err := r.Create(ctx, svc)
	if err != nil {
		logger.Error(err, "create service", "name", svc.Name)
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExecutorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qtccv1alpha1.Executor{}).
		Complete(r)
}
