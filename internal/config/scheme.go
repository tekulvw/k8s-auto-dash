package config

import (
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/anomalyco/k8s-auto-dash/api/v1alpha1"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = v1.AddToScheme(s)
	return s
}
