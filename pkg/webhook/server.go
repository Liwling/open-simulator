package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/alibaba/open-simulator/pkg/migrate/migrator/utils"
	simontype "github.com/alibaba/open-simulator/pkg/type"
	bottomutils "github.com/alibaba/open-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubernetes/pkg/apis/core/v1"
	"math/big"
	"net/http"
	"sigs.k8s.io/yaml"
	"strings"
	"sync"
	"time"
)

var (
	once sync.Once
	ws   *webHookServer
	err  error
)

const (
	admissionWebhookAnnotationStatusKey = "admission-webhook-simon/status"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

func NewWebhookServer(options Options) (WebHookServerInt, error) {
	once.Do(func() {
		ws, err = newWebHookServer(options)
	})
	return ws, err
}

func newWebHookServer(options Options) (*webHookServer, error) {
	webhookServiceName, webhookNamespace := options.WebHookServerName, options.WebHookNamespace
	dnsNames := []string{
		webhookServiceName,
		webhookServiceName + "." + webhookNamespace,
		webhookServiceName + "." + webhookNamespace + ".svc",
	}
	commonName := webhookServiceName + "." + webhookNamespace + ".svc"

	org := "simon.migration"
	caPEM, certPEM, certKeyPEM, err := generateCert([]string{org}, dnsNames, commonName)
	if err != nil {
		log.Errorf("Failed to generate ca and certificate key pair: %v", err)
	}

	kubeClient, err := bottomutils.CreateKubeClient(options.KubeConfig)
	if err != nil {
		log.Errorf("Failed to create kube client: %v", err)
	}

	// create or update the mutatingwebhookconfiguration
	err = createOrUpdateMutatingWebhookConfiguration(kubeClient, caPEM, webhookServiceName, webhookNamespace)
	if err != nil {
		log.Errorf("Failed to create or update the mutating webhook configuration: %v", err)
	}

	// load tls cert/key file
	tlsCertKey, err := tls.X509KeyPair(certPEM.Bytes(), certKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	migrationPlan, err := loadConfig(options.MigrationPlanCfgFile)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", ws.serve)

	ws := &webHookServer{
		server: &http.Server{
			Addr:      fmt.Sprintf(":%v", options.Port),
			Handler:   mux,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{tlsCertKey}},
		},
		migrationPlan: migrationPlan,
	}

	return ws, nil
}

// generateCert generate a self-signed CA for given organization
// and sign certificate with the CA for given common name and dns names
// it resurns the CA, certificate and private key in PEM format
func generateCert(orgs, dnsNames []string, commonName string) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error) {
	// init CA config
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(2022),
		Subject:               pkix.Name{Organization: orgs},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // expired in 1 year
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// generate private key for CA
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, err
	}

	// create the CA certificate
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// CA certificate with PEM encoded
	caPEM := new(bytes.Buffer)
	_ = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// new certificate config
	newCert := &x509.Certificate{
		DNSNames:     dnsNames,
		SerialNumber: big.NewInt(1024),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: orgs,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0), // expired in 1 year
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// generate new private key
	newPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, err
	}

	// sign the new certificate
	newCertBytes, err := x509.CreateCertificate(rand.Reader, newCert, ca, &newPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// new certificate with PEM encoded
	newCertPEM := new(bytes.Buffer)
	_ = pem.Encode(newCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: newCertBytes,
	})

	// new private key with PEM encoded
	newPrivateKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(newPrivateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(newPrivateKey),
	})

	return caPEM, newCertPEM, newPrivateKeyPEM, nil
}

func (ws *webHookServer) Start() {
	if err := ws.server.ListenAndServeTLS("", ""); err != nil {
		log.Errorf("Failed to listen and serve webhook server: %v", err)
	}
}

func (ws *webHookServer) Stop() {
	log.Infof("Got OS shutdown signal, shutting down wenhook server gracefully...")
	if err := ws.server.Shutdown(context.Background()); err != nil {
		log.Errorf("Failed to shut down server: %v", err)
	}
}

// main mutation process
func (whsvr *webHookServer) mutating(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		availableAnnotations map[string]string
	)

	log.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// TODO: 拦截所需修改的pod，并透出调度信息
	var expectedNodeNameValue string
	if !mutationRequired(whsvr.migrationPlan, pod, &expectedNodeNameValue) {
		log.Infof("Skipping validation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	// TODO：patch调度信息
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "mutated"}
	patchBytes, err := createPatch(expectedNodeNameValue, availableAnnotations, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *webHookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		log.Infof("The path of request: %s", r.URL.Path)
		if r.URL.Path == "/mutate" {
			admissionResponse = whsvr.mutating(&ar)
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		log.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	log.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func mutationRequired(migrationPlan map[workloadType]workload_node, pod corev1.Pod, nodeNameValue *string) bool {
	required := false
	for k, v1 := range migrationPlan {
		if pod.OwnerReferences[0].Kind == string(k) {
			str := strings.Split(pod.OwnerReferences[0].Name, simontype.SeparateSymbol)
			if v2, exist := v1[str[0]]; exist {
				required = true
				*nodeNameValue = v2[0]
				v1[str[0]] = utils.DeleteElemInStringSlice(v2[0], v2)
			}
			break
		}
	}
	log.Infof("Mutation policy for %v/%v: required:%v", pod.Namespace, pod.Name, required)
	return required
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

func createPatch(expectedNodeSelectorValue string, availableAnnotations map[string]string, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, updateNodeName(expectedNodeSelectorValue, "/spec/nodeName"))
	patch = append(patch, updateAnnotation(availableAnnotations, annotations)...)
	// patch = append(patch, updateLabels(availableLabels, labels)...)

	return json.Marshal(patch)
}

func updateNodeName(value string, basePath string) patchOperation {
	return patchOperation{
		Op:    "replace",
		Path:  basePath,
		Value: value,
	}
}

func loadConfig(configFile string) (map[workloadType]workload_node, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	log.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var migrationPlan map[workloadType]workload_node
	if err := yaml.Unmarshal(data, &migrationPlan); err != nil {
		return nil, err
	}

	return migrationPlan, nil
}
