package deploy

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SubmarinerNamespace = "submariner"
)

func NewSubmarinerNamespace() *v1.Namespace {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: SubmarinerNamespace,
		},
	}

	return ns
}
