/*
Copyright 2023.

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

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8snoviceoperatorv1 "github.com/jnytnai0613/k8snovice-operator/api/v1"
)

// K8sNoviceOperatorReconciler reconciles a K8sNoviceOperator object
type K8sNoviceOperatorReconciler struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
	Scheme    *runtime.Scheme
}

// Create OwnerReference with CR as Owner
func createOwnerReferences(
	log logr.Logger,
	k8snovice k8snoviceoperatorv1.K8sNoviceOperator,
	scheme *runtime.Scheme,
) (*metav1apply.OwnerReferenceApplyConfiguration, error) {
	gvk, err := apiutil.GVKForObject(&k8snovice, scheme)
	if err != nil {
		log.Error(err, "Unable get GVK")
		return nil, fmt.Errorf("unable to get GVK: %w", err)
	}

	owner := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(k8snovice.GetName()).
		WithUID(k8snovice.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)

	return owner, nil
}

func (r *K8sNoviceOperatorReconciler) applyDeployment(k8snovice k8snoviceoperatorv1.K8sNoviceOperator, logger logr.Logger) error {
	var (
		deploymentClient = r.Clientset.AppsV1().Deployments("k8snovice-operator-system")
		labels           = map[string]string{"apps": "nginx"}
		fieldMgr         = "k8snovice"
	)

	// Create applyconfiguration for Deployment
	nextDeploymentApplyConfig := appsv1apply.Deployment(
		"k8snovice-operator",
		"k8snovice-operator-system").
		WithSpec(appsv1apply.DeploymentSpec().
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(labels)))

	if k8snovice.Spec.DeploymentSpec.Replicas != nil {
		replicas := *k8snovice.Spec.DeploymentSpec.Replicas
		nextDeploymentApplyConfig.Spec.WithReplicas(replicas)
	}

	if k8snovice.Spec.DeploymentSpec.Strategy != nil {
		types := *k8snovice.Spec.DeploymentSpec.Strategy.Type
		rollingUpdate := k8snovice.Spec.DeploymentSpec.Strategy.RollingUpdate
		nextDeploymentApplyConfig.Spec.WithStrategy(appsv1apply.DeploymentStrategy().
			WithType(types).
			WithRollingUpdate(rollingUpdate))
	}

	podTemplate := k8snovice.Spec.DeploymentSpec.Template
	podTemplate.WithLabels(labels)
	if podTemplate != nil {
		nextDeploymentApplyConfig.Spec.WithTemplate(podTemplate)
	}

	// Create OwnerReference
	owner, err := createOwnerReferences(logger, k8snovice, r.Scheme)
	if err != nil {
		return fmt.Errorf("unable to create OwnerReference: %w", err)
	}
	nextDeploymentApplyConfig.WithOwnerReferences(owner)

	// If EmptyDir is not set to Medium or SizeLimit, applyconfiguration
	// returns an empty pointer address. Therefore, a comparison between
	// applyconfiguration in subsequent steps will always detect a difference.
	// In the following process, if neither Medium nor SizeLimit is set,
	// explicitly set nil to prevent the above problem.
	for i, v := range nextDeploymentApplyConfig.Spec.Template.Spec.Volumes {
		e := v.EmptyDir
		if e != nil {
			if v.EmptyDir.Medium != nil || v.EmptyDir.SizeLimit != nil {
				break
			}
			nextDeploymentApplyConfig.Spec.Template.Spec.Volumes[i].
				WithEmptyDir(nil)
		}
	}

	deployment, err := deploymentClient.Get(
		context.Background(),
		k8snovice.Spec.DeploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get Deployment: %w", err)
		}
	}
	currDeploymentMapApplyConfig, err := appsv1apply.ExtractDeployment(deployment, fieldMgr)
	if err != nil {
		return fmt.Errorf("failed to extract Deployment: %w", err)
	}

	// If there is no difference between the current and next applyconfigurations,
	if equality.Semantic.DeepEqual(currDeploymentMapApplyConfig, nextDeploymentApplyConfig) {
		return nil
	}

	// Apply the changes
	applied, err := deploymentClient.Apply(
		context.Background(),
		nextDeploymentApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		return fmt.Errorf("unable to apply Deployment: %w", err)
	}

	logger.Info("Deployment applied", "status", applied.Status)

	return nil
}

//+kubebuilder:rbac:groups=k8snoviceoperator.jnytnai0613.github.io,resources=k8snoviceoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8snoviceoperator.jnytnai0613.github.io,resources=k8snoviceoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8snoviceoperator.jnytnai0613.github.io,resources=k8snoviceoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *K8sNoviceOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var k8snovice k8snoviceoperatorv1.K8sNoviceOperator
	if err := r.Client.Get(ctx, req.NamespacedName, &k8snovice); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to fetch K8sNoviceOperator")
			return ctrl.Result{}, err
		}
	}

	// Resources in secondary clusters are considered external resources.
	// Therefore, they are deleted by finalizer.
	finalizerName := "plumber.jnytnai0613.github.io/finalizer"
	if !k8snovice.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&k8snovice, finalizerName) {
			// any finalizer logic here
			controllerutil.RemoveFinalizer(&k8snovice, finalizerName)
			if err := r.Client.Update(ctx, &k8snovice); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if !controllerutil.ContainsFinalizer(&k8snovice, finalizerName) {
		controllerutil.AddFinalizer(&k8snovice, finalizerName)
		if err := r.Client.Update(ctx, &k8snovice); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create Deployment
	if err := r.applyDeployment(k8snovice, logger); err != nil {
		logger.Error(err, "unable to apply Deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sNoviceOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8snoviceoperatorv1.K8sNoviceOperator{}).
		Complete(r)
}
