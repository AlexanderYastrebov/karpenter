/*
Copyright The Kubernetes Authors.

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

// Package ec2v1 contains a minimal stand-in for the AWS provider's EC2NodeClass (karpenter.k8s.aws/v1).
// It lets the KWOK provider serve NodePools that reference an EC2NodeClass, so the cluster looks like
// EKS + Karpenter to tools that only understand AWS NodeClasses. Only metadata and status are decoded;
// the spec is ignored. The CRD itself comes from the AWS karpenter-crd chart.
package ec2v1

import (
	"github.com/awslabs/operatorpkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	Group = "karpenter.k8s.aws"
	Kind  = "EC2NodeClass"
)

func init() {
	gv := schema.GroupVersion{Group: Group, Version: "v1"}
	metav1.AddToGroupVersion(scheme.Scheme, gv)
	scheme.Scheme.AddKnownTypes(gv,
		&EC2NodeClass{},
		&EC2NodeClassList{},
	)
}

type EC2NodeClassStatus struct {
	Conditions []status.Condition `json:"conditions,omitempty"`
}

type EC2NodeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status EC2NodeClassStatus `json:"status,omitempty"`
}

type EC2NodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EC2NodeClass `json:"items"`
}

func (in *EC2NodeClass) StatusConditions(opts ...status.ForOption) status.ConditionSet {
	return status.NewReadyConditions().For(in, opts...)
}

// GetConditions reports the NodeClass as Ready when no conditions are set, since there is no
// AWS controller to resolve AMIs, subnets and security groups for it.
func (in *EC2NodeClass) GetConditions() []status.Condition {
	if len(in.Status.Conditions) == 0 {
		return []status.Condition{{
			Type:               status.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             status.ConditionReady,
			LastTransitionTime: in.CreationTimestamp,
		}}
	}
	return in.Status.Conditions
}

func (in *EC2NodeClass) SetConditions(conditions []status.Condition) {
	in.Status.Conditions = conditions
}

func (in *EC2NodeClass) DeepCopyInto(out *EC2NodeClass) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]status.Condition, len(in.Status.Conditions))
		for i := range in.Status.Conditions {
			in.Status.Conditions[i].DeepCopyInto(&out.Status.Conditions[i])
		}
	}
}

func (in *EC2NodeClass) DeepCopy() *EC2NodeClass {
	if in == nil {
		return nil
	}
	out := new(EC2NodeClass)
	in.DeepCopyInto(out)
	return out
}

func (in *EC2NodeClass) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

func (in *EC2NodeClassList) DeepCopyInto(out *EC2NodeClassList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]EC2NodeClass, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *EC2NodeClassList) DeepCopy() *EC2NodeClassList {
	if in == nil {
		return nil
	}
	out := new(EC2NodeClassList)
	in.DeepCopyInto(out)
	return out
}

func (in *EC2NodeClassList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
