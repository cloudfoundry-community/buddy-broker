package buddy

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pivotal-golang/lager"
)

func New(logger lager.Logger) http.Handler {
	router := mux.NewRouter()
	handler := AppHandler{Logger: logger}
	handler.LoadBackendBrokerFromEnv()
	router.HandleFunc("/{suffix}/v2/catalog", handler.catalog).Methods("GET")
	router.HandleFunc("/{suffix}/v2/service_instances/{instance_id}", handler.provision).Methods("PUT")
	router.HandleFunc("/{suffix}/v2/service_instances/{instance_id}", handler.deprovision).Methods("DELETE")
	router.HandleFunc("/{suffix}/v2/service_instances/{instance_id}/last_operation", handler.lastOperation).Methods("GET")
	router.HandleFunc("/{suffix}/v2/service_instances/{instance_id}", handler.update).Methods("PATCH")

	router.HandleFunc("/{suffix}/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.bind).Methods("PUT")
	router.HandleFunc("/{suffix}/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.unbind).Methods("DELETE")
	return router
}
