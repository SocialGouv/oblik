package config

type Annotatable interface {
	GetAnnotations() map[string]string
	GetNamespace() string
	GetName() string
}
