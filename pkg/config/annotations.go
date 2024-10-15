package config

const PREFIX = "oblik.socialgouv.io/"

func getAnnotationFromMap(name string, annotations map[string]string) string {
	return annotations[PREFIX+name]
}

func getAnnotations(Annotable Annotable) map[string]string {
	annotations := Annotable.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	return annotations
}

func getLabelFromMap(name string, labels map[string]string) string {
	return labels[PREFIX+name]
}
