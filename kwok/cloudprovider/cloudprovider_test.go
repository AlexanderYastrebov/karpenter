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


package kwok

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	fakecr "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"sigs.k8s.io/karpenter/kwok/apis/ec2v1"
	"sigs.k8s.io/karpenter/kwok/options"
	v1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

func TestNodeLooksLikeEC2Instance(t *testing.T) {
	ctx := options.ToContext(context.Background(), &options.Options{InstanceTypesFilePath: "../examples/aws_instance_types.json"})
	instanceTypes, err := ConstructInstanceTypes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	kubeClient := fakecr.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	c := NewCloudProvider(ctx, kubeClient, instanceTypes)

	nodeClaim := &v1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "default-abcde"},
		Spec: v1.NodeClaimSpec{
			NodeClassRef: &v1.NodeClassReference{Group: ec2v1.Group, Kind: ec2v1.Kind, Name: "default"},
			Requirements: []v1.NodeSelectorRequirementWithMinValues{
				{Key: corev1.LabelInstanceTypeStable, Operator: corev1.NodeSelectorOpIn, Values: []string{"m5.large"}},
				{Key: corev1.LabelTopologyZone, Operator: corev1.NodeSelectorOpIn, Values: []string{"us-west-2b"}},
			},
		},
	}
	node, err := c.toNode(nodeClaim)
	if err != nil {
		t.Fatal(err)
	}

	if want := "aws:///us-west-2b/" + node.Name; node.Spec.ProviderID != want {
		t.Errorf("providerID = %q, want %q", node.Spec.ProviderID, want)
	}
	for key, want := range map[string]string{
		corev1.LabelInstanceTypeStable: "m5.large",
		corev1.LabelTopologyZone:       "us-west-2b",
		corev1.LabelTopologyRegion:     "us-west-2",
	} {
		if got := node.Labels[key]; got != want {
			t.Errorf("label %s = %q, want %q", key, got, want)
		}
	}

	if err := kubeClient.Create(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, node.Spec.ProviderID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.ProviderID != node.Spec.ProviderID {
		t.Errorf("Get providerID = %q, want %q", got.Status.ProviderID, node.Spec.ProviderID)
	}
	list, err := c.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("List returned %d nodeclaims, want 1", len(list))
	}
}

func TestRegionOnlyForAWSZones(t *testing.T) {
	ctx := options.ToContext(context.Background(), &options.Options{})
	instanceTypes, err := ConstructInstanceTypes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c := NewCloudProvider(ctx, fakecr.NewClientBuilder().Build(), instanceTypes)
	nodeClaim := &v1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "default-abcde"},
		Spec: v1.NodeClaimSpec{
			Requirements: []v1.NodeSelectorRequirementWithMinValues{
				{Key: corev1.LabelInstanceTypeStable, Operator: corev1.NodeSelectorOpIn, Values: []string{instanceTypes[0].Name}},
				{Key: corev1.LabelTopologyZone, Operator: corev1.NodeSelectorOpIn, Values: []string{"test-zone-a"}},
			},
		},
	}
	node, err := c.toNode(nodeClaim)
	if err != nil {
		t.Fatal(err)
	}
	if region, ok := node.Labels[corev1.LabelTopologyRegion]; ok {
		t.Errorf("unexpected region label %q for zone test-zone-a", region)
	}
	if !strings.HasPrefix(node.Spec.ProviderID, "aws:///test-zone-a/") {
		t.Errorf("providerID = %q", node.Spec.ProviderID)
	}
}

func TestEC2NodeClassIsReadyWithoutConditions(t *testing.T) {
	nodeClass := &ec2v1.EC2NodeClass{}
	if !nodeClass.StatusConditions().Root().IsTrue() {
		t.Errorf("EC2NodeClass without conditions should be Ready")
	}
}
