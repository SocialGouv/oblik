package config

type Annotable interface {
	GetAnnotations() map[string]string
	GetLabels() map[string]string
	GetNamespace() string
	GetName() string
}
