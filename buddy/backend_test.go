package buddy_test

import (
	"os"

	. "github.com/cloudfoundry-community/buddy-broker/buddy"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backend", func() {
	Describe("Test load environment", func() {
		It("returns hostname", func() {
			url := "https://localhost"
			logger := lager.NewLogger("buddy-backend-tests")
			os.Setenv("BACKEND_BROKER", url)
			handler := &AppHandler{Logger: logger}
			handler.LoadBackendBrokerFromEnv()
			Expect(handler.BackendBroker.URL).Should(Equal(url))

		})
	})
})
