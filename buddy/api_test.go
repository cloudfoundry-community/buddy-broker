package buddy_test

import (
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/cloudfoundry-community/buddy-broker/buddy"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Api", func() {
	var (
		backend   *ghttp.Server
		brokerAPI http.Handler
	)

	BeforeEach(func() {
		backend = ghttp.NewServer()
		os.Setenv("BACKEND_BROKER", backend.URL())
		logger := lager.NewLogger("buddy-api-tests")
		brokerAPI = New(logger)
	})

	Describe("Test not found", func() {
		makeRequest := func() *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest("GET", "/v2/catalog", nil)
			brokerAPI.ServeHTTP(recorder, request)
			return recorder
		}

		It("has a returned 404", func() {
			response := makeRequest()

			Ω(response.Code).Should(Equal(404))
		})
	})

})
