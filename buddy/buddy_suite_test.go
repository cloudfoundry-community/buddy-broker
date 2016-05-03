package buddy_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBuddy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buddy Suite")
}
