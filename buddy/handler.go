package buddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
)

var statusUnprocessableEntity = 422

// AppHandler is the main app
type AppHandler struct {
	BackendBroker backendBroker
	Logger        lager.Logger
}

type errorResponse struct {
	Error       string `json:"error,omitempty"`
	Description string `json:"description"`
}

func (b AppHandler) catalog(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	suffix := "-" + vars["suffix"]
	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/catalog", b.BackendBroker.URL)
	backendReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		b.Logger.Error("backend-catalog-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header

	resp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-catalog-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		b.respond(w, http.StatusUnauthorized, errorResponse{
			Description: "Not authorized",
		})
	}

	jsonData, err := ioutil.ReadAll(resp.Body)
	b.Logger.Info(string(jsonData))
	b.Logger.Info(b.BackendBroker.URL)
	var catalog brokerapi.CatalogResponse
	err = json.Unmarshal(jsonData, &catalog)
	if err != nil {
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	for i, service := range catalog.Services {
		catalog.Services[i].ID = service.ID + suffix
		catalog.Services[i].Name = service.Name + suffix
		for j, plan := range service.Plans {
			catalog.Services[i].Plans[j].ID = plan.ID + suffix
		}
	}
	b.respond(w, http.StatusOK, catalog)
}

func (b AppHandler) provision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	suffix := "-" + vars["suffix"]
	instanceID := vars["instance_id"]

	var details struct {
		ServiceID        string      `json:"service_id"`
		PlanID           string      `json:"plan_id"`
		OrganizationGUID string      `json:"organization_guid"`
		SpaceGUID        string      `json:"space_guid"`
		Parameters       interface{} `json:"parameters,omitempty"`
	}

	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		b.respond(w, statusUnprocessableEntity, errorResponse{
			Description: err.Error(),
		})
		return
	}

	fmt.Printf("provision: decoded details: %#v\n", details)

	details.ServiceID = strings.TrimSuffix(details.ServiceID, suffix)
	details.PlanID = strings.TrimSuffix(details.PlanID, suffix)
	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s", b.BackendBroker.URL, instanceID)
	buffer := &bytes.Buffer{}
	if err := json.NewEncoder(buffer).Encode(details); err != nil {
		b.Logger.Error("backend-provision-encode-details", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}

	fmt.Println("provision: encoded details:", buffer.String())

	backendReq, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		b.Logger.Error("backend-provision-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-provision-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()
	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.WriteHeader(httpResp.StatusCode)
	w.Write(data)
}

func (b AppHandler) deprovision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s?plan_id=%s&service_id=%s", b.BackendBroker.URL, instanceID, req.FormValue("plan_id"), req.FormValue("service_id"))
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("DELETE", url, buffer)
	if err != nil {
		b.Logger.Error("backend-deprovision-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-deprovision-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.WriteHeader(httpResp.StatusCode)
	w.Write(data)
}

func (b AppHandler) lastOperation(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s/last_operation", b.BackendBroker.URL, instanceID)
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("GET", url, buffer)
	if err != nil {
		b.Logger.Error("backend-lastoperations-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-lastoperations-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.WriteHeader(httpResp.StatusCode)
	w.Write(data)
}

func (b AppHandler) update(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s", b.BackendBroker.URL, instanceID)
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("PATCH", url, buffer)
	if err != nil {
		b.Logger.Error("backend-update-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-update-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.WriteHeader(httpResp.StatusCode)
	w.Write(data)
}

func (b AppHandler) bind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindID := vars["binding_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s", b.BackendBroker.URL, instanceID, bindID)
	buffer := &bytes.Buffer{}
	backendReq, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		b.Logger.Error("backend-binding-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-binding-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.WriteHeader(httpResp.StatusCode)
	w.Write(data)
	return
}

func (b AppHandler) unbind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s?plan_id=%s&service_id=%s", b.BackendBroker.URL, instanceID, bindingID, req.FormValue("plan_id"), req.FormValue("service_id"))
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("DELETE", url, buffer)
	if err != nil {
		b.Logger.Error("backend-unbinding-req", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-unbinding-resp", err)
		b.respond(w, http.StatusInternalServerError, errorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.WriteHeader(httpResp.StatusCode)
	w.Write(data)
}

func (b AppHandler) reject(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Please provide a suffix in url"))
}

func (b AppHandler) respond(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		b.Logger.Error("encoding response", err, lager.Data{"status": status, "response": response})
	}
}
