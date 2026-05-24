package discovery

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func corev1NS_(name string) client.Object {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}
