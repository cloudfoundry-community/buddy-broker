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

type BuddyHandler struct {
	BackendBroker backendBroker
	Logger        lager.Logger
}

type ErrorResponse struct {
	Error       string `json:"error,omitempty"`
	Description string `json:"description"`
}

func (b BuddyHandler) catalog(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	suffix := "-" + vars["suffix"]
	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/catalog", b.BackendBroker.URL)
	backendReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		b.Logger.Error("backend-catalog-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header

	resp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-catalog-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		b.respond(w, http.StatusUnauthorized, ErrorResponse{
			Description: "Not authorized",
		})
	}

	jsonData, err := ioutil.ReadAll(resp.Body)
	b.Logger.Info(string(jsonData))
	b.Logger.Info(b.BackendBroker.URL)
	var catalog brokerapi.CatalogResponse
	err = json.Unmarshal(jsonData, &catalog)
	if err != nil {
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
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

func (b BuddyHandler) provision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	suffix := "-" + vars["suffix"]
	instanceID := vars["instance_id"]

	var details brokerapi.ProvisionDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		b.respond(w, statusUnprocessableEntity, ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	details.ServiceID = strings.TrimSuffix(details.ServiceID, suffix)
	details.PlanID = strings.TrimSuffix(details.PlanID, suffix)

	var provisioningResponse brokerapi.ProvisioningResponse
	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s", b.BackendBroker.URL, instanceID)
	buffer := &bytes.Buffer{}
	if err := json.NewEncoder(buffer).Encode(details); err != nil {
		b.Logger.Error("backend-provision-encode-details", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		b.Logger.Error("backend-provision-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-provision-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusCreated || httpResp.StatusCode == http.StatusOK {
		var jsonData []byte
		jsonData, err = ioutil.ReadAll(httpResp.Body)

		if err = json.Unmarshal(jsonData, &provisioningResponse); err != nil {
			b.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
			return
		}
		if err == nil {
			b.Logger.Info("provision-success", lager.Data{
				"instance-id": instanceID,
				"plan-id":     details.PlanID,
				"backend-uri": b.BackendBroker.URL,
			})
			b.respond(w, httpResp.StatusCode, provisioningResponse)
			return
		}
	}
}

func (b BuddyHandler) deprovision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s?plan_id=%s&service_id=%s", b.BackendBroker.URL, instanceID, req.FormValue("plan_id"), req.FormValue("service_id"))
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("DELETE", url, buffer)
	if err != nil {
		b.Logger.Error("backend-deprovision-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-deprovision-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.Write(data)
	w.WriteHeader(httpResp.StatusCode)
}

func (b BuddyHandler) lastOperation(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s/last_operation", b.BackendBroker.URL, instanceID)
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("GET", url, buffer)
	if err != nil {
		b.Logger.Error("backend-lastoperations-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-lastoperations-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.Write(data)
	w.WriteHeader(httpResp.StatusCode)
}

func (b BuddyHandler) update(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s", b.BackendBroker.URL, instanceID)
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("PATCH", url, buffer)
	if err != nil {
		b.Logger.Error("backend-update-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-update-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.Write(data)
	w.WriteHeader(httpResp.StatusCode)
}

func (b BuddyHandler) bind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindID := vars["binding_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s", b.BackendBroker.URL, instanceID, bindID)
	buffer := &bytes.Buffer{}
	backendReq, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		b.Logger.Error("backend-binding-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-binding-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.Write(data)
	w.WriteHeader(httpResp.StatusCode)
	return
}

func (b BuddyHandler) unbind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]

	client := &http.Client{}
	url := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s?plan_id=%s&service_id=%s", b.BackendBroker.URL, instanceID, bindingID, req.FormValue("plan_id"), req.FormValue("service_id"))
	buffer := &bytes.Buffer{}

	backendReq, err := http.NewRequest("DELETE", url, buffer)
	if err != nil {
		b.Logger.Error("backend-unbinding-req", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	backendReq.Header = req.Header
	backendReq.Body = req.Body

	httpResp, err := client.Do(backendReq)
	if err != nil {
		b.Logger.Error("backend-unbinding-resp", err)
		b.respond(w, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(httpResp.Body)
	w.Write(data)
	w.WriteHeader(httpResp.StatusCode)
}

func (b BuddyHandler) reject(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Please provide a suffix in url"))
}

func (b BuddyHandler) respond(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		b.Logger.Error("encoding response", err, lager.Data{"status": status, "response": response})
	}
}
