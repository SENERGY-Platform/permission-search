package configuration

func getResourceList(c Config) (result []string) {
	for resource := range c.Resources {
		result = append(result, resource)
	}
	return
}

func getAnnotationResourceIndex(c Config) (result map[string][]string) {
	result = map[string][]string{}
	for resourceName, resource := range c.Resources {
		for annotationResource := range resource.Annotations {
			result[annotationResource] = append(result[annotationResource], resourceName)
		}
	}
	return result
}
