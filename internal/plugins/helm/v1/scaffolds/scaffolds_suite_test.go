package scaffolds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestScaffolds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scaffolds Suite")
}
