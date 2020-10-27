package configuration

func getResourceList(c Config) (result []string) {
	for resource := range c.Resources {
		result = append(result, resource)
	}
	return
}
