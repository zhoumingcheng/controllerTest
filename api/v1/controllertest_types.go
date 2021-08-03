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

package v1

import (
	v1be "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ControllerTestSpec defines the desired state of ControllerTest
type ControllerTestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ControllerTest. Edit ControllerTest_types.go to remove/update
	Foo string `json:"foo,omitempty"`
	//标签选择 属于该资源下的deploy，svc，ing会打上该标签
	Selector metav1.LabelSelector `json:"selector"`
	//快速部署 会默认设置一些参数
	Items []Item `json:"items"` //deploy与svc快速部署模板
	Rules []v1be.IngressRule `json:"rules"` //ingress 配置rules
}

// deploy与svc快速部署模板

type Item struct {
	Name string `json:"name"` //deployment名称，serviceName与ingressName也相同 pod名称与nginx名称也根据该名称配置
	Replicas *int32 `json:"replicas"` //对应deploy的replicas字段
	Selector metav1.LabelSelector `json:"selector"` //用于svc与deploy的关联
	Image string `json:"image"` //对应deploy的image字段  容器镜像
	Ports []Port `json:"ports"`//对应deploy 容器ports 字段下的ContainerPort 容器对应的端口
}

// 对应service暴露在cluster ip上的端口

type Port struct {
	TargetPort int32 `json:"targetPort"` //对应deploy 容器ports 字段下的ContainerPort 容器对应的端口 和 targetPort
	Port int32 `json:"port"` //对应service暴露在cluster ip上的端口
}

// ControllerTestStatus defines the observed state of ControllerTest
type ControllerTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// ControllerTest is the Schema for the controllertests API
type ControllerTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControllerTestSpec   `json:"spec,omitempty"`
	Status ControllerTestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ControllerTestList contains a list of ControllerTest
type ControllerTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControllerTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ControllerTest{}, &ControllerTestList{})
}
