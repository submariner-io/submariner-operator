package broker

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateConfigMap(config *rest.Config, name string, namespace string) (*v1.ConfigMap, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}
	cfm , err := clientset.CoreV1().ConfigMaps(namespace).Create(
		NewConfigMap(name, namespace))
	return cfm, err
}

func NewConfigMap(name string, namespace string) *v1.ConfigMap {
	cf := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Namespace: namespace,
		},
		Data:       nil, // fill as and when needed (eg: while joining the cluster)
	}
	return cf
}

//func GetConfigMap(config *rest.Config, name string, namespace string) (*v1.ConfigMap, error) {
//	clientset, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		return nil, fmt.Errorf("error creating the core kubernetes clientset: %s", err)
//	}
//	cfm , err := clientset.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
//	return cfm, err
//}