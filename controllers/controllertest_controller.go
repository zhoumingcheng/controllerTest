/*


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

package controllers

import (
	"context"
	webappv1 "controllerTest/api/v1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1be "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControllerTestReconciler reconciles a ControllerTest object
type ControllerTestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.com.bolingcavalry,resources=controllertests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.com.bolingcavalry,resources=controllertests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;update;patch;delete

func (r *ControllerTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("controllertest", req.NamespacedName)

	// your logic here
	//按照namespace加载controllerTest
	controllerTest := &webappv1.ControllerTest{}
	err := r.Get(ctx, req.NamespacedName, controllerTest)
	if err != nil {
		//如果没有查询到实例则直接返回
		if errors.IsNotFound(err) {
			log.Info("未查询到实例！")
			return ctrl.Result{}, nil
		}
		log.Error(err, "")
		return ctrl.Result{}, err
	}

	//查询到实例后查询该实例下的deploy，svc，ing
	if err := r.ListResource(log,ctx,controllerTest);err != nil {
		return ctrl.Result{}, err
	}

	// 声明 finalizer 字段，类型为字符串
	myFinalizerName := "storage.finalizers.tutorial.kubebuilder.io"

	// 通过检查 DeletionTimestamp 字段是否为0 判断资源是否被删除
	if controllerTest.ObjectMeta.DeletionTimestamp.IsZero() {
		// 如果为0 ，则资源未被删除，我们需要检测是否存在 finalizer，如果不存在，则添加，并更新到资源对象中
		if !containsString(controllerTest.ObjectMeta.Finalizers, myFinalizerName) {
			controllerTest.ObjectMeta.Finalizers = append(controllerTest.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), controllerTest); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// 如果不为 0 ，则对象处于删除中
		if containsString(controllerTest.ObjectMeta.Finalizers, myFinalizerName) {
			// 如果存在 finalizer 且与上述声明的 finalizer 匹配，那么执行对应 hook 逻辑
			if err := r.deleteExternalResources(ctx, controllerTest); err != nil {
				// 如果删除失败，则直接返回对应 err，controller 会自动执行重试逻辑
				return ctrl.Result{}, err
			}

			// 如果对应 hook 执行成功，那么清空 finalizers， k8s 删除对应资源
			controllerTest.ObjectMeta.Finalizers = removeString(controllerTest.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), controllerTest); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ControllerTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.ControllerTest{}).
		Complete(r)
}

// 查询该实例下的deploy，svc，ing ,若没有则进行创建

func (r *ControllerTestReconciler) ListResource(log logr.Logger, ctx context.Context, controllerTest *webappv1.ControllerTest) error {
	//查询deploy
	deploymentList := &v1.DeploymentList{}
	if err := r.List(ctx, deploymentList, client.InNamespace(controllerTest.Namespace),client.MatchingLabelsSelector{},
		client.MatchingLabels{"app": controllerTest.Spec.Selector.MatchLabels["app"]}); err != nil {
		log.Error(err, "unable to list child deployments")
		return err
	}
	//没有deploy 则创建deploy
	if len(deploymentList.Items) == 0 {
		for _, item := range controllerTest.Spec.Items {
			deployment := GetDeployFill(item, controllerTest)
			if err := r.Create(ctx, deployment); err != nil {
				log.Error(err, "create deploy error")
				return err
			}
		}
	}

	//查询svc
	svcList := &corev1.ServiceList{}
	if err := r.List(ctx, svcList, client.InNamespace(controllerTest.Namespace),
		client.MatchingLabels{"app": controllerTest.Spec.Selector.MatchLabels["app"]}); err != nil {
		log.Error(err, "unable to list child services")
		return err
	}
	//没有svc 则创建svc
	if len(svcList.Items) == 0 {
		for _, item := range controllerTest.Spec.Items {
			svc := GetSvcFill(item, controllerTest)
			if err := r.Create(ctx, svc); err != nil {
				log.Error(err, "create svc error")
				return err
			}
		}
	}

	//查询ingress
	ingList := &v1be.IngressList{}
	if err := r.List(ctx, ingList, client.InNamespace(controllerTest.Namespace),
		client.MatchingLabels{"app": controllerTest.Spec.Selector.MatchLabels["app"]}); err != nil {
		log.Error(err, "unable to list child ingress")
		return err
	}
	//没有ing 则创建ing
	if len(ingList.Items) == 0 {
		svc := GetIngFill(controllerTest)
		if err := r.Create(ctx, svc); err != nil {
			log.Error(err, "create ing error")
			return err
		}
	}
	return nil
}

//获取deploy资源填写

func GetDeployFill(item webappv1.Item, controllerTest *webappv1.ControllerTest) *v1.Deployment {
	containers := make([]corev1.Container, 0)
	ports := make([]corev1.ContainerPort, 0)
	for _, port := range item.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          "",
			HostPort:      0,
			ContainerPort: port.TargetPort,
			Protocol:      "",
			HostIP:        "",
		})
	}
	containers = append(containers, corev1.Container{
		Name:                     item.Name,
		Image:                    item.Image,
		Command:                  nil,
		Args:                     nil,
		WorkingDir:               "",
		Ports:                    ports,
		EnvFrom:                  nil,
		Env:                      nil,
		Resources:                corev1.ResourceRequirements{},
		VolumeMounts:             nil,
		VolumeDevices:            nil,
		LivenessProbe:            nil,
		ReadinessProbe:           nil,
		StartupProbe:             nil,
		Lifecycle:                nil,
		TerminationMessagePath:   "",
		TerminationMessagePolicy: "",
		ImagePullPolicy:          "",
		SecurityContext:          nil,
		Stdin:                    false,
		StdinOnce:                false,
		TTY:                      false,
	})
	deployment := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       item.Name,
			GenerateName:               "",
			Namespace:                  controllerTest.Namespace,
			SelfLink:                   "",
			UID:                        "",
			ResourceVersion:            "",
			Generation:                 0,
			CreationTimestamp:          metav1.Time{},
			DeletionTimestamp:          nil,
			DeletionGracePeriodSeconds: nil,
			Labels:                     controllerTest.Spec.Selector.MatchLabels,
			Annotations:                nil,
			OwnerReferences:            nil,
			Finalizers:                 nil,
			ClusterName:                "",
			ManagedFields:              nil,
		},
		Spec: v1.DeploymentSpec{
			Replicas:                item.Replicas,
			Selector:                &metav1.LabelSelector{
				MatchLabels: item.Selector.MatchLabels,
				MatchExpressions: nil,
			},
			Template:                corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:                       "",
					GenerateName:               "",
					Namespace:                  "",
					SelfLink:                   "",
					UID:                        "",
					ResourceVersion:            "",
					Generation:                 0,
					CreationTimestamp:          metav1.Time{},
					DeletionTimestamp:          nil,
					DeletionGracePeriodSeconds: nil,
					Labels:                     item.Selector.MatchLabels,
					Annotations:                nil,
					OwnerReferences:            nil,
					Finalizers:                 nil,
					ClusterName:                "",
					ManagedFields:              nil,
				},
				Spec:       corev1.PodSpec{
					Volumes:                       nil,
					InitContainers:                nil,
					Containers:                    containers,
					EphemeralContainers:           nil,
					RestartPolicy:                 "",
					TerminationGracePeriodSeconds: nil,
					ActiveDeadlineSeconds:         nil,
					DNSPolicy:                     "",
					NodeSelector:                  nil,
					ServiceAccountName:            "",
					DeprecatedServiceAccount:      "",
					AutomountServiceAccountToken:  nil,
					NodeName:                      "",
					HostNetwork:                   false,
					HostPID:                       false,
					HostIPC:                       false,
					ShareProcessNamespace:         nil,
					SecurityContext:               nil,
					ImagePullSecrets:              nil,
					Hostname:                      "",
					Subdomain:                     "",
					Affinity:                      nil,
					SchedulerName:                 "",
					Tolerations:                   nil,
					HostAliases:                   nil,
					PriorityClassName:             "",
					Priority:                      nil,
					DNSConfig:                     nil,
					ReadinessGates:                nil,
					RuntimeClassName:              nil,
					EnableServiceLinks:            nil,
					PreemptionPolicy:              nil,
					Overhead:                      nil,
					TopologySpreadConstraints:     nil,
				},
			},
			Strategy:                v1.DeploymentStrategy{},
			MinReadySeconds:         0,
			RevisionHistoryLimit:    nil,
			Paused:                  false,
			ProgressDeadlineSeconds: nil,
		},
		Status: v1.DeploymentStatus{},
	}
	return deployment
}

//获取svc资源填写

func GetSvcFill(item webappv1.Item, controllerTest *webappv1.ControllerTest) *corev1.Service {
	ports := make([]corev1.ServicePort,0)
	for _, port := range item.Ports {
		ports = append(ports,corev1.ServicePort{
			Name:       "",
			Protocol:   corev1.ProtocolTCP,
			Port:       port.Port,
			TargetPort: intstr.IntOrString{IntVal:port.TargetPort},
			NodePort:   0,
		})
	}
	svc := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       item.Name,
			GenerateName:               "",
			Namespace:                  controllerTest.Namespace,
			SelfLink:                   "",
			UID:                        "",
			ResourceVersion:            "",
			Generation:                 0,
			CreationTimestamp:          metav1.Time{},
			DeletionTimestamp:          nil,
			DeletionGracePeriodSeconds: nil,
			Labels:                     controllerTest.Spec.Selector.MatchLabels,
			Annotations:                nil,
			OwnerReferences:            nil,
			Finalizers:                 nil,
			ClusterName:                "",
			ManagedFields:              nil,
		},
		Spec:       corev1.ServiceSpec{
			Ports:                    ports,
			Selector:                 item.Selector.MatchLabels,
			ClusterIP:                "",
			Type:                     "",
			ExternalIPs:              nil,
			SessionAffinity:          "",
			LoadBalancerIP:           "",
			LoadBalancerSourceRanges: nil,
			ExternalName:             "",
			ExternalTrafficPolicy:    "",
			HealthCheckNodePort:      0,
			PublishNotReadyAddresses: false,
			SessionAffinityConfig:    nil,
			IPFamily:                 nil,
			TopologyKeys:             nil,
		},
		Status:     corev1.ServiceStatus{},
	}
	return svc
}

//获取ing资源填写

func GetIngFill(controllerTest *webappv1.ControllerTest) *v1be.Ingress {
	annotations := make(map[string]string)
	annotations["kubernetes.io/ingress.class"] = "nginx"
	ing := &v1be.Ingress{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       controllerTest.Name,
			GenerateName:               "",
			Namespace:                  controllerTest.Namespace,
			SelfLink:                   "",
			UID:                        "",
			ResourceVersion:            "",
			Generation:                 0,
			CreationTimestamp:          metav1.Time{},
			DeletionTimestamp:          nil,
			DeletionGracePeriodSeconds: nil,
			Labels:                     controllerTest.Spec.Selector.MatchLabels,
			Annotations:                annotations,
			OwnerReferences:            nil,
			Finalizers:                 nil,
			ClusterName:                "",
			ManagedFields:              nil,
		},
		Spec:       v1be.IngressSpec{
			Backend: nil,
			TLS:     nil,
			Rules:   controllerTest.Spec.Rules,
		},
		Status:     v1be.IngressStatus{},
	}
	return ing
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *ControllerTestReconciler) deleteExternalResources(ctx context.Context, controllerTest *webappv1.ControllerTest) error {
	// 删除 controllerTest关联的resources
	// 删除deployment
	deployment := &v1.Deployment{}
	if err := r.DeleteAllOf(ctx, deployment, client.InNamespace(controllerTest.Namespace),
		client.MatchingLabels{"app": controllerTest.Spec.Selector.MatchLabels["app"]}); err != nil {
		r.Log.Error(err,"删除controllerTest关联deployments失败")
		return err
	}
	//删除service
	svcList := &corev1.ServiceList{}
	if err := r.List(ctx, svcList, client.InNamespace(controllerTest.Namespace),
		client.MatchingLabels{"app": controllerTest.Spec.Selector.MatchLabels["app"]}); err != nil {
		r.Log.Error(err, "unable to list child services")
		return err
	}
	//没有svc 则创建svc
	if len(svcList.Items) != 0 {
		for _, item := range svcList.Items {
			service := &corev1.Service{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:                       item.Name,
					Namespace:                  item.Namespace,
				},
				Spec:       corev1.ServiceSpec{},
				Status:     corev1.ServiceStatus{},
			}
			if err := r.Delete(ctx, service); err != nil {
				r.Log.Error(err,"删除controllerTest关联services失败")
				return err
			}
		}
	}
	//删除ingress
	ingress := &v1be.Ingress{}
	if err := r.DeleteAllOf(ctx, ingress, client.InNamespace(controllerTest.Namespace),
		client.MatchingLabels{"app": controllerTest.Spec.Selector.MatchLabels["app"]}); err != nil {
		r.Log.Error(err,"删除controllerTest关联ingresses失败")
		return err
	}
	return nil
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}