package server

import (
	"k8s.io/api/admission/v1beta1"
	"net/http"
)

type WebHookServerInt interface {
	mutating(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
	Start()
	Stop()
}

type Options struct {
	Port              int
	WebHookServerName string
	WebHookNamespace  string
	KubeConfig        string
	//TLSCertFile          string
	//TLSKeyFile           string
	MigrationPlanCfgFile string
}

type webHookServer struct {
	migrationPlan map[workloadType]workload_node
	server        *http.Server
}

type workload_node map[string][]string

type workloadType string

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}
