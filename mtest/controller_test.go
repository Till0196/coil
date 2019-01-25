package mtest

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cybozu-go/coil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("coil-controller", func() {
	BeforeEach(func() {
		initializeCoil()
		coilctl("pool create default 10.1.0.0/24 2")

		// Dump current nodes
		kubectl("get nodes/10.0.0.101 -o json >/tmp/node1.json")
		kubectl("get nodes/10.0.0.102 -o json >/tmp/node2.json")
	})
	AfterEach(func() {
		cleanCoil()

		kubectl("apply -f /tmp/node1.json")
		kubectl("apply -f /tmp/node2.json")
	})

	It("deletes node's block by watching kubernetes api", func() {
		By("creating pods")
		overrides := fmt.Sprintf(`{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" } } }`, node1)
		_, stderr, err := kubectl("run --generator=run-pod/v1 nginx --image=nginx --overrides='" + overrides + "' --restart=Never")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			stdout, stderr, err := kubectl("get pods --selector=run=nginx -o json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			podList := new(corev1.PodList)
			err = json.Unmarshal(stdout, podList)
			Expect(err).NotTo(HaveOccurred())

			if len(podList.Items) > 0 && podList.Items[0].Status.Phase != corev1.PodRunning {
				return errors.New("Pod is not created")
			}
			return nil
		}).Should(Succeed())

		By("deleting node resources")
		_, stderr, err = kubectl("delete -f /tmp/node1.json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("checking block is released")
		Eventually(func() error {
			stdout := coilctl("pool show default 10.1.0.0/16 --json")

			ba := new(coil.BlockAssignment)
			err = json.Unmarshal(stdout, ba)
			Expect(err).NotTo(HaveOccurred())

			if _, ok := ba.Nodes[node1]; ok {
				return errors.New("node1 is still in block assignments")
			}
			return nil
		})
	})

	It("deletes node's block on initialization of the controller", func() {
		By("creating pods")
		overrides := fmt.Sprintf(`{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" } } }`, node1)
		_, stderr, err := kubectl("run --generator=run-pod/v1 nginx --image=nginx --overrides='" + overrides + "' --restart=Never")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			stdout, stderr, err := kubectl("get pods --selector=run=nginx -o json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			podList := new(corev1.PodList)
			err = json.Unmarshal(stdout, podList)
			Expect(err).NotTo(HaveOccurred())

			if len(podList.Items) > 0 && podList.Items[0].Status.Phase != corev1.PodRunning {
				return errors.New("Pod is not created")
			}
			return nil
		}).Should(Succeed())

		By("deleting coil-controllers Deployment")
		_, stderr, err = kubectl("delete deployment/coil-controllers --namespace=kube-system")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			stdout, stderr, err := kubectl("get deployment --selector=k8s-app=coil-controllers --namespace=kube-system -o json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			deployments := new(appsv1.DeploymentList)
			err = json.Unmarshal(stdout, deployments)
			Expect(err).NotTo(HaveOccurred())

			if len(deployments.Items) > 0 {
				return errors.New("deployments still exist")
			}
			return nil
		}).Should(Succeed())

		By("deleting Node resources")
		_, stderr, err = kubectl("delete -f /tmp/node1.json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			stdout, stderr, err := kubectl("get nodes --selector=kubernetes.io/hostname=10.0.0.101 -o json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			nodes := new(corev1.NodeList)
			err = json.Unmarshal(stdout, nodes)
			Expect(err).NotTo(HaveOccurred())

			if len(nodes.Items) > 0 {
				return errors.New("nodes still exist")
			}
			return nil
		}).Should(Succeed())

		_, stderr, err = kubectl("apply -f /data/deploy.yml")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("checking block is released")
		Eventually(func() error {
			stdout := coilctl("pool show default 10.1.0.0/16 --json")

			ba := new(coil.BlockAssignment)
			err = json.Unmarshal(stdout, ba)
			Expect(err).NotTo(HaveOccurred())

			if _, ok := ba.Nodes[node1]; ok {
				return errors.New("node1 is still in block assignments")
			}
			return nil
		})
	})
})
