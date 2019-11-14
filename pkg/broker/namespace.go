package broker

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SubmarinerBrokerNamespace = "submariner-k8s-broker"
)

func NewBrokerNamespace() *v1.Namespace {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: SubmarinerBrokerNamespace,
		},
	}

	return ns
}
