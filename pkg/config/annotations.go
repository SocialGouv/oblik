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

func getLabels(Annotable Annotable) map[string]string {
	labels := Annotable.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	return labels
}

func getLabelFromMap(name string, labels map[string]string) string {
	return labels[PREFIX+name]
}
