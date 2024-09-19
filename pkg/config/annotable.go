package config

type Annotable interface {
	GetAnnotations() map[string]string
	GetNamespace() string
	GetName() string
}
