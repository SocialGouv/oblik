package config

func getAnnotationFromMap(name string, annotations map[string]string) string {
	return annotations["oblik.socialgouv.io/"+name]
}

func getAnnotations(Annotable Annotable) map[string]string {
	annotations := Annotable.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	return annotations
}
