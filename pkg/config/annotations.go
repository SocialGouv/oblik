package config

func getAnnotationFromMap(name string, annotations map[string]string) string {
	return annotations["oblik.socialgouv.io/"+name]
}

func getAnnotations(annotatable Annotatable) map[string]string {
	annotations := annotatable.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	return annotations
}
