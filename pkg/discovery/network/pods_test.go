package network

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/fake"
)

const (
	testComponent1 = "test-component1"
	testComponent2 = "test-component2"
	testFirstPod   = "first-pod"
	testSecondPod  = "second-pod"
	testThirdPod   = "third-pod"
)

var _ = Describe("findPod", func() {

	var clientSet *fake.Clientset

	BeforeEach(func() {
		clientSet = fake.NewSimpleClientset(
			fakePodWithName(testFirstPod, testComponent1, nil, nil),
			fakePodWithName(testSecondPod, testComponent2, nil, nil),
			fakePodWithName(testThirdPod, testComponent2, nil, nil),
		)
	})

	When("There are no pods to be found", func() {
		It("Should return nil", func() {
			pod, err := findPod(clientSet, "component=not-to-be-found")
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).To(BeNil())
		})
	})

	When("A pod is found", func() {
		It("Should return the pod", func() {
			pod, err := findPod(clientSet, componentLabel(testComponent1))
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal(testFirstPod))
		})
	})

	When("Multiple pods are found", func() {
		It("Should return the first pod", func() {
			pod, err := findPod(clientSet, componentLabel(testComponent2))
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal(testSecondPod))
		})
	})
})

const (
	testParameter1 = "--parameter1"
	testValue1     = "value1"
	testComponent3 = "test-component3"
)

var _ = Describe("findPodCommandParameter", func() {

	var clientSet *fake.Clientset

	BeforeEach(func() {
		clientSet = fake.NewSimpleClientset(
			fakePodWithName(testFirstPod, testComponent1, nil, nil),
			fakePodWithName(testSecondPod, testComponent2,
				[]string{"component1", testParameter1 + "=" + testValue1}, nil),
			fakePodWithName(testThirdPod, testComponent3,
				[]string{"sh", "-c", "component1 " + testParameter1 + "=" + testValue1}, nil),
		)
	})

	When("There are no pods to be found", func() {
		It("Should return empty string", func() {
			param, err := findPodCommandParameter(clientSet, componentLabel("not-to-be-found"), testParameter1)
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(BeEmpty())
		})
	})

	When("A pod is found, but does not contain the parameter", func() {
		It("Should return an empty string", func() {
			param, err := findPodCommandParameter(clientSet, componentLabel(testComponent1), "unknown-parameter")
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(BeEmpty())
		})
	})

	When("A pod is found, and contains the parameter", func() {
		It("Should return the parameter value", func() {
			param, err := findPodCommandParameter(clientSet, componentLabel(testComponent2), testParameter1)
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(Equal(testValue1))

		})
	})

	When("A pod is found, and the parameter is wrapped in a sh call", func() {
		It("Should return the parameter value", func() {
			param, err := findPodCommandParameter(clientSet, componentLabel(testComponent3), testParameter1)
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(Equal(testValue1))
		})
	})
})

func componentLabel(component string) string {
	return "component=" + component
}
