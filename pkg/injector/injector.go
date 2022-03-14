/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package injector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	scheme "github.com/quanxiang-cloud/qtcc/pkg/client/clientset/versioned"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Admission controller's control qtcc,
// more reference from https://github.com/dapr/dapr/blob/master/pkg/injector/injector.go

const (
	port = 4001
)

var (
	log = ctrl.Log.WithName("Injector")
)

// Injector is the interface for the quanxiang tenant controller center runtime injection component.
type Injector interface {
	Run(ctx context.Context)
}

type injector struct {
	config       Config
	deserializer runtime.Decoder
	server       *http.Server
	kubeClient   kubernetes.Interface
	client       scheme.Interface
}

func NewInjector(config Config, client scheme.Interface, kubeClient kubernetes.Interface) Injector {
	mux := http.NewServeMux()

	i := &injector{
		config: config,
		deserializer: serializer.NewCodecFactory(
			runtime.NewScheme(),
		).UniversalDeserializer(),
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		kubeClient: kubeClient,
		client:     client,
	}

	mux.HandleFunc("/mutate", i.handleRequest)
	return i
}

func (i *injector) handleRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := context.Background()

	body, ok := i.getBody(w, r)
	if !ok {
		return
	}

	log.Info(string(body), "body", "JSON")

	var patchOps []PatchOperation
	var admissionResponse = &v1.AdmissionResponse{}

	ar := v1.AdmissionReview{}
	_, gvk, err := i.deserializer.Decode(body, nil, &ar)
	if err != nil {
		log.Error(err, "Can't decode body")
	} else {
		patchOps, err = i.getPatchOperations(ctx, &ar)
		if err != nil {
			log.Error(err, "getPatchOperations")
			admissionResponse.Result = &metav1.Status{
				Message: err.Error(),
			}
		}
	}

	if len(patchOps) == 0 {
		admissionResponse = &v1.AdmissionResponse{
			Allowed: true,
		}
	} else {
		var patchBytes []byte
		patchBytes, err = json.Marshal(patchOps)
		if err != nil {
			admissionResponse.Result = &metav1.Status{
				Message: err.Error(),
			}
		} else {
			admissionResponse = &v1.AdmissionResponse{
				Allowed: true,
				Patch:   patchBytes,
				PatchType: func() *v1.PatchType {
					pt := v1.PatchTypeJSONPatch
					return &pt
				}(),
			}
		}
	}

	admissionReview := v1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
			admissionReview.SetGroupVersionKind(*gvk)
		}
	}

	log.Info("ready to write response ...")
	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)

		log.Error(err, "Can't deserialize response: %s", "app")
	}
	w.Header().Set("Content-Type", runtime.ContentTypeJSON)
	if _, err := w.Write(respBytes); err != nil {
		log.Error(err, "write response")
	}
}

func (i *injector) getBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Error(fmt.Errorf("empty body"), "")
		http.Error(w, "empty body", http.StatusBadRequest)
		return nil, false
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != runtime.ContentTypeJSON {
		http.Error(
			w,
			fmt.Sprintf("invalid Content-Type, expect `%s`", runtime.ContentTypeJSON),
			http.StatusUnsupportedMediaType,
		)

		return nil, false
	}

	return body, true
}

func (i *injector) Run(ctx context.Context) {
	doneCh := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			log.Info("injector is shutting down")
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5,
			)
			defer cancel()
			i.server.Shutdown(shutdownCtx) // nolint: errcheck
		case <-doneCh:
		}
	}()

	log.Info("injector is listening", "port", i.server.Addr)
	err := i.server.ListenAndServeTLS(i.config.TLSCertFile, i.config.TLSKeyFile)
	if err != http.ErrServerClosed {
		log.Error(err, "injector error")
	}
	close(doneCh)
}
