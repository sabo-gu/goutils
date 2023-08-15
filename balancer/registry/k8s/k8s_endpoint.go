package k8s

import "time"

type K8sEndpointInfo struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		SelfLink          string    `json:"selfLink"`
		UID               string    `json:"uid"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
	} `json:"metadata"`
	Subsets []struct {
		Addresses []struct {
			IP        string `json:"ip"`
			NodeName  string `json:"nodeName"`
			TargetRef struct {
				Kind            string `json:"kind"`
				Namespace       string `json:"namespace"`
				Name            string `json:"name"`
				UID             string `json:"uid"`
				ResourceVersion string `json:"resourceVersion"`
			} `json:"targetRef"`
		} `json:"addresses"`
		Ports []struct {
			Name     string `json:"name"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
		} `json:"ports"`
	} `json:"subsets"`
}
