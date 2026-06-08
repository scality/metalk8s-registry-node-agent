/*
Copyright 2025 Scality.

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

//nolint:dupl,goconst // E2E tests have intentional code duplication for test clarity and independence
package e2e

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"github.com/scality/metalk8s-registry-node-agent/test/utils"
)

// namespace where the project is deployed in
const namespace = "metalk8s-registry"

// serviceAccountName created for the project
const serviceAccountName = "metalk8s-registry-node-agent-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "metalk8s-registry-node-agent-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "metalk8s-registry-node-agent-metrics-binding"

// Default chunk size of 45kB for multipart uploads
const chunkSize = 45 * 1024

var nodeNames []string
var controllerPodNameByNode = make(map[string]string)
var tlsConfig *tls.Config
var controllerPodName string

var _ = Describe("Manager", Ordered, func() {
	// Before running the tests, set up the environment by creating the namespace,
	// installing CRDs and deploying the controller and related certificates.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to allow the privileged security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=privileged")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with privileged policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("retrieving the worker nodes with specific label")
		cmd = exec.Command("kubectl", "get", "nodes",
			"-l", "role=worker",
			"-o", "go-template={{ range .items }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}")
		nodeOutput, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get nodes")
		Expect(nodeOutput).NotTo(BeEmpty(), "No nodes with worker role found")
		nodeNames = utils.GetNonEmptyLines(nodeOutput)

		By("deploying the E2E certs")
		cmd = exec.Command("kubectl", "apply", "-n", namespace, "-f", "./test/e2e/e2e-certs.yaml")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply certs")
		time.Sleep(3 * time.Second) // Wait for the certs to be created

		By("retrieving the tls certificate")
		var certB64Output, keyB64Output string
		var certBytes, keyBytes []byte
		Eventually(func(g Gomega) {
			cmd = exec.Command("kubectl", "get", "secret", "tls-cert", "-n", namespace, "-o", "jsonpath=\"{.data.tls\\.crt}\"")
			certB64Output, err = utils.Run(cmd)
			certB64Output = strings.ReplaceAll(certB64Output, "\"", "")
			g.Expect(err).NotTo(HaveOccurred(), "Failed to get tls cert")
		}, 1*time.Minute, 2*time.Second).Should(Succeed())

		certBytes, err = base64.StdEncoding.DecodeString(certB64Output)
		Expect(err).NotTo(HaveOccurred(), "Failed to decode cert")

		Eventually(func(g Gomega) {
			cmd = exec.Command("kubectl", "get", "secret", "tls-cert", "-n", namespace, "-o", "jsonpath=\"{.data.tls\\.key}\"")
			keyB64Output, err = utils.Run(cmd)
			keyB64Output = strings.ReplaceAll(keyB64Output, "\"", "")
			g.Expect(err).NotTo(HaveOccurred(), "Failed to get tls key")
		}, 1*time.Minute, 2*time.Second).Should(Succeed())

		keyBytes, err = base64.StdEncoding.DecodeString(keyB64Output)
		Expect(err).NotTo(HaveOccurred(), "Failed to decode key")

		By("generating the tls config")
		clientCert, err := tls.X509KeyPair(certBytes, keyBytes)
		Expect(err).NotTo(HaveOccurred(), "Failed to get tls certificate and key")
		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{clientCert},
			InsecureSkipVerify: true,
		}

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy-e2e", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("scaling the controller-manager pods to all worker nodes")
		cmd = exec.Command("kubectl", "scale", "statefulset",
			"-l", "control-plane=controller-manager",
			"-n", namespace,
			"--replicas", fmt.Sprintf("%d", len(nodeNames)))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to scale the controller-manager pods")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			for _, nodeName := range nodeNames {
				controllerPodName := controllerPodNameByNode[nodeName]

				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				controllerLogs, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
				}
			}

			By("Fetching Kubernetes events")
			cmd := exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			for _, nodeName := range nodeNames {
				controllerPodName := controllerPodNameByNode[nodeName]
				cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
				podDescription, err := utils.Run(cmd)
				if err == nil {
					fmt.Println("Pod description:\n", podDescription)
				} else {
					fmt.Println("Failed to describe controller pod")
				}
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pods are running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(len(nodeNames)), "expected %d controller pods running", len(nodeNames))

				// Retrieve the controller pods
				// and fill the controllerPodNameByNode map
				for _, podName := range podNames {
					g.Expect(podName).To(ContainSubstring("controller-manager"))

					// Validate the pod's status
					cmd = exec.Command("kubectl", "get",
						"pods", podName, "-o", "jsonpath={.status.phase}",
						"-n", namespace,
					)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")

					// Retrieve the node name for the pod
					cmd = exec.Command("kubectl", "get",
						"pods", podName, "-o", "jsonpath={.spec.nodeName}",
						"-n", namespace,
					)
					output, err = utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					controllerPodNameByNode[output] = podName
					// We consider randomly a pod for the rest of the tests
					controllerPodName = podName
				}
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			// We consider only the first pod for this test
			controllerPodName := controllerPodNameByNode[nodeNames[0]]

			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=metalk8s-registry-node-agent-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccount": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
		})

		It("should provisioned cert-manager", func() {
			By("validating that cert-manager has the certificate Secret")
			verifyCertManager := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(verifyCertManager).Should(Succeed())
		})

		It("should have CA injection for validating webhooks", func() {
			By("checking CA injection for validating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"validatingwebhookconfigurations.admissionregistration.k8s.io",
					"metalk8s-registry-node-agent-validating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				vwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput := getMetricsOutput()
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
		//    strings.ToLower(<Kind>),
		// ))
	})

	Context("When applying some NodeSolutionArchives and uploading a solution archive", func() {
		It("should successfully reconcile the resources", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s",
				version: "1.25.3",
				label:   "e2e",
				size:    20000,
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames)
			Expect(err).NotTo(HaveOccurred(), "Failed to create and apply NodeSolutionArchives")

			By("verifying the NodeSolutionArchives status")
			for _, nodeName := range nodeNames {
				Expect(solutionArchive.isAvailableOnNode(nodeName)).To(BeTrue(),
					"NodeSolutionArchive should be available after upload")
				Expect(solutionArchive.isServedOnNode(nodeName)).To(BeTrue(),
					"NodeSolutionArchive should be served after upload")
			}

			By("cleaning up test resources")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())
		})
	})

	Context("When deleting a NodeSolutionArchive", func() {
		It("should delete the solution archive from /archives and unmount from /solutions", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s-deletion",
				version: "1.0.0",
				label:   "deletion",
				size:    200,
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames[0:1])
			Expect(err).NotTo(HaveOccurred(), "Failed to create and apply NodeSolutionArchive")

			nodeName := nodeNames[0] // Test is done on one node only

			By("verifying the NodeSolutionArchive status")
			Expect(solutionArchive.isAvailableOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be available after upload")
			Expect(solutionArchive.isServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be served after upload")

			By("deleting the NodeSolutionArchive resource")
			cmd := exec.Command("kubectl", "delete", "nodesolutionarchive",
				fmt.Sprintf("na-%s-%s-%s", solutionArchive.name, solutionArchive.version, nodeName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete NodeSolutionArchive")

			By("waiting for the controller to process the deletion")
			Expect(solutionArchive.isNotPresentOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should have been deleted")

			By("verifying the solution archive is unmounted from /solutions directory")
			Expect(solutionArchive.isNotMountedOnNode(nodeName)).To(BeTrue(),
				"Solution archive should be unmounted after deletion")

			By("verifying the solution archive file is deleted from /archives directory")
			Expect(solutionArchive.fileNotExistsOnNode(nodeName)).To(BeTrue(),
				"Solution archive file should not exist in /archives directory")
		})
	})

	Context("When a solution archive is manually unmounting", func() {
		It("should automatically remount this solution archive", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s-remount",
				version: "3.0.0",
				label:   "e2e-remount",
				size:    200,
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames[0:1])
			Expect(err).NotTo(HaveOccurred(), "Failed to create and apply NodeSolutionArchive")

			nodeName := nodeNames[0] // Test is done on one node only

			By("verifying the NodeSolutionArchives status")
			Expect(solutionArchive.isAvailableOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be available after upload")
			Expect(solutionArchive.isServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be served after upload")
			mountPoint := solutionArchive.mountPointOnNode(nodeName)
			Expect(mountPoint).NotTo(BeEmpty(), "Mount point should be found")

			By("manually unmounting the solution archive")
			podName := controllerPodNameByNode[nodeName]
			// Use lazy unmount (-l) to handle the case where the mount is still in use
			cmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--",
				"umount", "-l", mountPoint)
			output, err := utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "Unmount command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "Unmount should succeed")

			By("waiting for the controller to detect the unmount and automatically remount")
			// The controller reconciles continuously and should detect the unmount
			// and remount the solution archive automatically. In practice, the controller is so fast
			// that the solution archive may already be remounted by the time we check.
			// We verify that the solution archive remains mounted (i.e., the controller keeps it mounted).
			time.Sleep(500 * time.Millisecond)

			By("verifying the NodeSolutionArchive status, remains Available and Served")
			Expect(solutionArchive.isAvailableOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should remain available after remount")
			Expect(solutionArchive.isServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be served after remount")

			By("cleaning up test resources")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())
		})
	})

	Context("When creating fake solution archive objects", func() {
		It("should automatically delete unexpected solution archive files", func() {
			By("initializing the target pod name")
			nodeName := nodeNames[0]
			targetPodName := controllerPodNameByNode[nodeName]

			By("creating a fake solution archive file")
			solutionArchiveFilePath := fmt.Sprintf("/archives/%s", "metalk8s-fake-solution-archive.iso")
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
				"sh", "-c", "echo 'fake' > "+solutionArchiveFilePath)
			output, err := utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "Create command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "Failed to create fake solution archive file in /archives directory")

			By("waiting for the controller to detect the fake solution archive file and delete it")
			time.Sleep(1 * time.Second)

			By("verifying the fake solution archive file is deleted from /archives directory")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
					"test", "-f", solutionArchiveFilePath)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "solution archive file should not exist in /archives directory")
			}, time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should automatically delete unexpected solution archive directories", func() {
			By("initializing the target pod name")
			nodeName := nodeNames[0]
			targetPodName := controllerPodNameByNode[nodeName]

			By("creating a fake solution archive directory")
			solutionArchiveDirPath := fmt.Sprintf("/archives/%s", "metalk8s-fake-solution-archive")
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
				"sh", "-c", "mkdir -p "+solutionArchiveDirPath)
			output, err := utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "Create command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "Failed to create fake solution archive directory in /archives directory")

			By("waiting for the controller to detect the fake solution archive directory and delete it")
			time.Sleep(1 * time.Second)

			By("verifying the fake solution archive directory is deleted from /archives directory")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
					"test", "-d", solutionArchiveDirPath)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "solution archive directory should not exist in /archives directory")
			}, time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Context("When creating fake solution objects", func() {
		It("should automatically delete unexpected solution files", func() {
			By("initializing the target pod name")
			nodeName := nodeNames[0]
			targetPodName := controllerPodNameByNode[nodeName]

			By("creating a fake solution file")
			solutionFilePath := fmt.Sprintf("/solutions/%s", "metalk8s-fake-solution.txt")
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
				"sh", "-c", "echo 'fake' > "+solutionFilePath)
			output, err := utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "Create command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "Failed to create fake solution file in /solutions directory")

			By("waiting for the controller to detect the fake solution archive file and delete it")
			time.Sleep(1 * time.Second)

			By("verifying the fake solution file is deleted from /solutions directory")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
					"test", "-f", solutionFilePath)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "solution file should not exist in /solutions directory")
			}, time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should automatically delete unexpected solution directories", func() {
			By("initializing the target pod name")
			nodeName := nodeNames[0]
			targetPodName := controllerPodNameByNode[nodeName]

			By("creating a 1-level deep fake solution directory")
			solutionDirPath1 := fmt.Sprintf("/solutions/%s", "metalk8s-fake-solution-archive1")
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
				"sh", "-c", "mkdir -p "+solutionDirPath1)
			output, err := utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "Create command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "Failed to create 1-level deep fake directory in /solutions directory")

			By("creating a 2-level deep fake solution directories")
			solutionDirPath2 := fmt.Sprintf("/solutions/%s", "metalk8s-fake-solution-archive2")
			solutionSubDirPath2 := fmt.Sprintf("%s/%s", solutionDirPath2, "3.2.1")
			cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
				"sh", "-c", "mkdir -p "+solutionSubDirPath2)
			output, err = utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "Create command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "Failed to create 2-level deep fake directory in /solutions directory")

			By("waiting for the controller to detect the fake solution archive directory and delete it")
			time.Sleep(1 * time.Second)

			By("verifying the 1-level deep fake solution directory is deleted from /solutions directory")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
					"test", "-d", solutionDirPath1)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "1-level deep fake directory should be deleted")
			}, time.Minute, 5*time.Second).Should(Succeed())

			By("verifying the 2-level deep fake solution directory are deleted from /solutions directory")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
					"test", "-d", solutionSubDirPath2)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "2-level deep fake directory should be deleted")
			}, time.Minute, 5*time.Second).Should(Succeed())

			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
					"test", "-d", solutionDirPath2)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "2-level deep fake directory should be deleted")
			}, time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Context("When badly modifying a solution archive", func() {
		It("should automatically unmount the solution and delete the solution archive file", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s-update",
				version: "3.2.1",
				label:   "e2e-update",
				size:    200,
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames[0:1])
			Expect(err).NotTo(HaveOccurred(), "Failed to create and apply NodeSolutionArchives")

			nodeName := nodeNames[0] // Only the first node is used for the test

			By("verifying the NodeSolutionArchives status")
			Expect(solutionArchive.isAvailableOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be available after upload")
			Expect(solutionArchive.isServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be served after upload")

			By("manually updating the solution archive")
			podName := controllerPodNameByNode[nodeName]
			solutionArchiveFilePath := fmt.Sprintf("/archives/%s-%s.iso", solutionArchive.name, solutionArchive.version)
			cmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--",
				"sh", "-c", "echo 'fake' > "+solutionArchiveFilePath)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "update should succeed")

			By("waiting for the controller to detect the update and delete the solution archive")
			time.Sleep(500 * time.Millisecond)

			By("verifying the NodeSolutionArchive status")
			Expect(solutionArchive.isUnavailableOnNode(nodeName)).To(BeTrue(),
				"solution archive file should be unavailable after update")
			Expect(solutionArchive.isNotServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should become Unserved after update")

			By("cleaning up test resources")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())
		})
	})

	Context("When deleting a solution archive file", func() {
		It("should automatically unmount the solution and delete the solution archive file", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s-deletion",
				version: "2.0.0",
				label:   "e2e-deletion",
				size:    200,
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames[0:1])
			Expect(err).NotTo(HaveOccurred(), "Failed to create and apply NodeSolutionArchives")

			By("verifying the NodeSolutionArchive status")
			nodeName := nodeNames[0]
			Expect(solutionArchive.isAvailableOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be available after upload")
			Expect(solutionArchive.isServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should be served after upload")

			By("manually deleting the solution archive file")
			podName := controllerPodNameByNode[nodeName]
			solutionArchiveFilePath := fmt.Sprintf("/archives/%s-%s.iso", solutionArchive.name, solutionArchive.version)
			cmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--",
				"sh", "-c", "rm -f "+solutionArchiveFilePath)
			output, err := utils.Run(cmd)
			fmt.Fprintf(GinkgoWriter, "deleting command output: %s\n", output) // nolint: errcheck // No need to check.
			Expect(err).NotTo(HaveOccurred(), "deletion should succeed")

			By("waiting for the controller to detect the deletion")
			time.Sleep(500 * time.Millisecond)

			By("verifying the NodeSolutionArchive status")
			Expect(solutionArchive.isUnavailableOnNode(nodeName)).To(BeTrue(),
				"solution archive file should be unavailable after update")
			Expect(solutionArchive.isNotServedOnNode(nodeName)).To(BeTrue(),
				"NodeSolutionArchive should become Unserved after update")

			By("cleaning up test resources")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())
		})
	})

	Context("When applying a NodeSolutionArchive and uploading a solution archive with a wrong checksum", func() {
		It("should delete the uploaded file", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s-fake",
				version: "1.12.9",
				label:   "e2e-fake",
				size:    200,
				hash:    "wrongChecksum",
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames)
			Expect(err).To(HaveOccurred(), "Should failed to upload solution archive to controller pod after 3 attempts")
			fmt.Println("Error: ", err)

			By("verifying the NodeSolutionArchive status")
			for _, nodeName := range nodeNames {
				Expect(solutionArchive.isUnavailableOnNode(nodeName)).To(BeTrue(),
					"NodeSolutionArchive should be Unavailable after upload")
				Expect(solutionArchive.isNotServedOnNode(nodeName)).To(BeTrue(),
					"NodeSolutionArchive should be not served after upload")
			}

			By("cleaning up test resources")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())
		})
	})

	Context("When applying multiple NodeSolutionArchive versions and uploading different solution archives", func() {
		It("should successfully reconcile the resources, even when deleting one solution archive", func() {
			solutionArchives := []NodeSolutionArchiveForTest{
				{
					name: "metalk8s-version-a",
					// version is a random number with enough digits to avoid conflicts with other tests
					version: fmt.Sprintf("%d.%d.%d", utils.RandomInt(0, 10), utils.RandomInt(0, 10), utils.RandomInt(0, 10)),
					label:   "e2e-multiple-versions",
					size:    200,
				},
				{
					name: "metalk8s-version-a",
					// version is a random number with enough digits to avoid conflicts with other tests
					version: fmt.Sprintf("%d.%d.%d", utils.RandomInt(0, 10), utils.RandomInt(0, 10), utils.RandomInt(0, 10)),
					label:   "e2e-multiple-versions",
					size:    200,
				},
				{
					name: "metalk8s-version-b",
					// version is a random number with enough digits to avoid conflicts with other tests
					version: fmt.Sprintf("%d.%d.%d", utils.RandomInt(0, 10), utils.RandomInt(0, 10), utils.RandomInt(0, 10)),
					label:   "e2e-multiple-versions",
					size:    200,
				},
			}

			By("creating NodeSolutionArchive custom resources")
			var wg sync.WaitGroup
			var mu sync.Mutex
			var applyErrs []error
			for _, sa := range solutionArchives {
				wg.Add(1)
				go func(sa NodeSolutionArchiveForTest) {
					defer wg.Done()
					if err := sa.createAndApplyOnNodes(nodeNames); err != nil {
						mu.Lock()
						applyErrs = append(applyErrs, err)
						mu.Unlock()
					}
				}(sa)
			}
			wg.Wait()
			Expect(applyErrs).To(BeEmpty(), "Failed to create and apply NodeSolutionArchives")

			By("verifying the NodeSolutionArchives status")
			for _, sa := range solutionArchives {
				for _, nodeName := range nodeNames {
					Expect(sa.isAvailableOnNode(nodeName)).To(BeTrue(), "NodeSolutionArchive should be available after upload")
					Expect(sa.isServedOnNode(nodeName)).To(BeTrue(), "NodeSolutionArchive should be served after upload")
				}
			}

			By("deleting a NodeSolutionArchive from 'metalk8s-version-a' family, others should remain available")
			for _, nodeName := range nodeNames {
				nodeSolutionArchiveName := fmt.Sprintf("na-%s-%s-%s",
					solutionArchives[0].name, solutionArchives[0].version, nodeName)
				cmd := exec.Command("kubectl", "delete", "nodesolutionarchives.metalk8s.scality.com",
					nodeSolutionArchiveName)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())
			}

			// The controller automatically reconciles the NodeSolutionArchive when deleted
			// Give it a moment to complete the cleanup
			By("waiting for the controller to clean up the test resources")
			time.Sleep(2 * time.Second)

			By("verifying the NodeSolutionArchive status")
			for _, sa := range solutionArchives[1:] {
				for _, nodeName := range nodeNames {
					By("verifying the NodeSolutionArchive status updates to Available")
					Expect(sa.isAvailableOnNode(nodeName)).To(BeTrue(), "NodeSolutionArchive should remain available")
					Expect(sa.isServedOnNode(nodeName)).To(BeTrue(), "NodeSolutionArchive should remain served")
				}
			}

			By("cleaning up test resources")
			// As the labels are the same for all solution archives, we can select the first one to delete
			Expect(solutionArchives[0].deleteOnAllNodes()).To(Succeed())
		})
	})

	Context("When deleting a NodeSolutionArchive while uploading the solution archive", func() {
		It("should fail the upload", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:       "metalk8s-deletion-while-uploading",
				version:    "2.0.0",
				label:      "deletion-while-uploading",
				size:       200,
				syncUpload: make(chan struct{}),
			}

			By("creating and uploading a solution archive file")
			uploadResult := make(chan error, 1)
			go func(sa NodeSolutionArchiveForTest) {
				err := sa.createAndApplyOnNodes(nodeNames[0:1])
				uploadResult <- err
			}(solutionArchive)

			// Wait for upload to have begun
			<-solutionArchive.syncUpload

			By("deleting the NodeSolutionArchive")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())

			By("waiting for the controller to end uploading the solution archive")
			uploadSuccess := <-uploadResult
			Expect(uploadSuccess).To(HaveOccurred(), "Upload should have failed")
		})
	})

	Context("When uploading an unmountable ISO file", func() {
		It("should fail the mount process and delete the solution archive", func() {
			solutionArchive := NodeSolutionArchiveForTest{
				name:    "metalk8s-unmountable",
				version: "1.2.6",
				label:   "e2e-unmountable",
				size:    200,
				fakeISO: true,
			}

			By("creating and uploading a solution archive file")
			err := solutionArchive.createAndApplyOnNodes(nodeNames)
			Expect(err).NotTo(HaveOccurred(), "Failed to create and apply NodeSolutionArchives")

			By("waiting for the controller to deal with the solution archive")
			time.Sleep(2 * time.Second)

			By("verifying the NodeSolutionArchive status")
			for _, nodeName := range nodeNames {
				Expect(solutionArchive.isUnavailableOnNode(nodeName)).To(BeTrue(),
					"NodeSolutionArchive should be unavailable after upload")
				Expect(solutionArchive.isNotServedOnNode(nodeName)).To(BeTrue(),
					"NodeSolutionArchive should not be served after upload")
			}

			By("cleaning up test resources")
			Expect(solutionArchive.deleteOnAllNodes()).To(Succeed())
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

type NodeSolutionArchiveForTest struct {
	name         string
	version      string
	label        string
	filePath     string
	manifestPath string
	size         int64
	hash         string
	fakeISO      bool
	syncUpload   chan struct{}
	synced       bool
}

// uploadSolutionArchiveToPod uploads a solution archive file dividing it into multipart chunks.
// It uses port-forward to expose the HTTP API locally and then makes HTTP requests.
func (nsa *NodeSolutionArchiveForTest) uploadToPod(podName string) bool {
	By("setting up port-forward to the controller pod")

	// Start port-forward in the background
	// choose a random port for port-forward between 5003 and 5100
	// the range is enough to avoid conflicts with other tests
	// 0 is not a good option, as the value is used in the url
	localForwardPort := utils.RandomInt(5003, 5100)
	portForwardCmd := exec.Command("kubectl", "port-forward",
		"-n", namespace,
		fmt.Sprintf("pod/%s", podName),
		fmt.Sprintf("%d:%d", localForwardPort, 5001),
	)

	// Start the port-forward process
	err := portForwardCmd.Start()
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "Failed to start port-forward: %v\n", err) // nolint: errcheck // No need to check.
		return false
	}

	// Ensure we clean up the port-forward process
	defer func() {
		if portForwardCmd.Process != nil {
			_ = portForwardCmd.Process.Kill()
			_ = portForwardCmd.Wait()
		}
	}()

	// Wait a bit for port-forward to establish
	time.Sleep(2 * time.Second)

	By("reading the solution archive file")
	fileData, err := os.ReadFile(nsa.filePath)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "Failed to read solution archive file: %v\n", err) // nolint: errcheck // No need to check.
		return false
	}

	fileSize := len(fileData)

	// Create HTTP client for all chunk uploads
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	url := fmt.Sprintf("https://localhost:%d/api/v1/uploads/%s/%s", localForwardPort, nsa.name, nsa.version)

	// Calculate number of chunks
	numChunks := (fileSize + chunkSize - 1) / chunkSize
	By(fmt.Sprintf("uploading the solution archive via HTTP API in %d chunk(s)", numChunks))

	// Upload file in chunks
	for chunkIndex := range numChunks {
		start := chunkIndex * chunkSize
		end := min(start+chunkSize, fileSize)

		chunkData := fileData[start:end]
		contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end-1, fileSize)

		By(fmt.Sprintf("uploading chunk %d/%d (bytes %d-%d)", chunkIndex+1, numChunks, start, end-1))

		// Create the HTTP request for this chunk
		req, err := http.NewRequestWithContext(context.Background(),
			http.MethodPut, url, bytes.NewReader(chunkData))
		if err != nil {
			// nolint: errcheck // No need to check.
			fmt.Fprintf(GinkgoWriter, "Failed to create HTTP request for chunk %d: %v\n",
				chunkIndex+1, err)
			return false
		}

		// Set headers
		req.Header.Set("Content-Range", contentRange)
		req.Header.Set("Content-Type", "application/octet-stream")

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			// nolint: errcheck // No need to check.
			fmt.Fprintf(GinkgoWriter, "Failed to upload chunk %d: %v\n",
				chunkIndex+1, err)
			return false
		}

		// Read response body
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // nolint: errcheck // No error check needed.

		// nolint: errcheck // No need to check.
		fmt.Fprintf(GinkgoWriter, "Chunk %d upload response status: %d\n", chunkIndex+1,
			resp.StatusCode)
		// nolint: errcheck // No need to check.
		fmt.Fprintf(GinkgoWriter, "Chunk %d upload response body: %s\n", chunkIndex+1,
			string(respBody))

		if nsa.syncUpload != nil && !nsa.synced {
			close(nsa.syncUpload)
			nsa.synced = true
		}

		// Check if the response indicates success (HTTP 200)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(GinkgoWriter, // nolint: errcheck // No need to check.
				"Chunk %d upload did not return HTTP 200, got %d\n", chunkIndex+1, resp.StatusCode)
			return false
		}

		time.Sleep(500 * time.Millisecond)
	}

	return true
}

// createAndApplyOnNodes creates a temporary ISO file, applies the related CR to the nodes and uploads the file.
func (nsa *NodeSolutionArchiveForTest) createAndApplyOnNodes(nodeNames []string) (err error) {
	randomName := utils.RandomString(12)
	nsa.filePath = filepath.Join("/tmp", fmt.Sprintf("%s.iso", randomName))
	nsa.manifestPath = filepath.Join("/tmp", fmt.Sprintf("%s.yaml", randomName))
	hash := ""
	if nsa.fakeISO {
		hash, err = utils.MakeWrongISOFile(nsa.filePath, nsa.size)
		Expect(err).NotTo(HaveOccurred(), "Failed to create wrong ISO file")
	} else {
		hash, err = utils.MakeISOFile(nsa.filePath, nsa.size)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ISO file")
	}
	if nsa.hash == "" {
		nsa.hash = hash
	}
	defer os.Remove(nsa.filePath) // nolint: errcheck // No error check on defer.

	By("creating NodeSolutionArchive custom resources")
	for _, nodeName := range nodeNames {
		// Create NodeSolutionArchive for the worker nodes
		err = nsa.applyOnNode(nodeName)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply NodeSolutionArchive on node %s", nodeName)
	}

	By("waiting for the controller to initialize the upload session")
	// The controller automatically initializes a session when it reconciles the NodeSolutionArchive
	// Give it a moment to complete
	for _, nodeName := range nodeNames {
		Expect(nsa.isInitializedOnNode(nodeName)).To(BeTrue(),
			"NodeSolutionArchive should be initialized after creation")
	}

	By("uploading the solution archive to the first node")
	primaryNode := nodeNames[0]
	targetPod := controllerPodNameByNode[primaryNode]
	var uploadSuccess bool
	for attempt := 1; attempt <= 3; attempt++ {
		_, _ = fmt.Fprintf(GinkgoWriter, "Upload attempt %d/3 (node %s, pod %s)\n",
			attempt, primaryNode, targetPod)
		uploadSuccess = nsa.uploadToPod(targetPod)
		if uploadSuccess {
			break
		}
		if attempt < 3 {
			fmt.Fprintf(GinkgoWriter, "Upload failed, retrying in 2 seconds...\n") // nolint: errcheck // No need to check.
			time.Sleep(2 * time.Second)
		}
	}
	if !uploadSuccess {
		return fmt.Errorf("failed to upload solution archive to pod after 3 attempts")
	}

	return nil
}

// applyOnNode applies a CR to the node.
func (nsa *NodeSolutionArchiveForTest) applyOnNode(nodeName string) error {
	nodeSolutionArchiveName := fmt.Sprintf("na-%s-%s-%s", nsa.name, nsa.version, nodeName)
	nodeSolutionArchiveYAML := fmt.Sprintf(`
apiVersion: metalk8s.scality.com/v1alpha1
kind: NodeSolutionArchive
metadata:
  name: %s
  labels:
    test-name: %s
spec:
  name: %s
  version: "%s"
  nodeName: %s
  validation:
    checksum:
      type: sha256
      value: "%s"
`, nodeSolutionArchiveName, nsa.label, nsa.name, nsa.version, nodeName, nsa.hash)

	err := os.WriteFile(nsa.manifestPath, []byte(nodeSolutionArchiveYAML), 0644)
	Expect(err).NotTo(HaveOccurred(), "Failed to create temporary NodeSolutionArchive YAML")
	defer os.Remove(nsa.manifestPath) // nolint: errcheck // No error check on defer.

	cmd := exec.Command("kubectl", "apply", "-f", nsa.manifestPath)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply NodeSolutionArchive on node %s", nodeName)

	return nil
}

// deleteOnAllNodes deletes all instances of the CR identified by label
// nolint: unparam // error return is required for assertion.
func (nsa *NodeSolutionArchiveForTest) deleteOnAllNodes() error {
	cmd := exec.Command("kubectl", "delete", "nodesolutionarchives.metalk8s.scality.com",
		"-l", fmt.Sprintf("test-name=%s", nsa.label))
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete NodeSolutionArchives")

	// The controller automatically reconciles the NodeSolutionArchive when deleted
	By("verifying the NodeSolutionArchives are deleted")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "nodesolutionarchives.metalk8s.scality.com",
			"-l", fmt.Sprintf("test-name=%s", nsa.label))
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("No resources found"),
			"Expected 'No resources found' error when querying deleted NodeSolutionArchives")
	}, 2*time.Minute, 5*time.Second).Should(Succeed())

	return nil
}

// isNotPresentOnNode checks if the ISO file is not present on the node.
func (nsa *NodeSolutionArchiveForTest) isNotPresentOnNode(nodeName string) bool {
	timeout := 1 * time.Minute
	interval := 500 * time.Millisecond
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "nodesolutionarchive", fmt.Sprintf("na-%s-%s-%s",
			nsa.name, nsa.version, nodeName))
		output, err := utils.Run(cmd)
		g.Expect(err).To(HaveOccurred())
		g.Expect(output).To(ContainSubstring(
			fmt.Sprintf("nodesolutionarchives.metalk8s.scality.com \"na-%s-%s-%s\" not found",
				nsa.name, nsa.version, nodeName)),
			"NodeSolutionArchive should be deleted")
	}, timeout, interval).Should(Succeed())

	return true
}

// isInitializedOnNode means: initialized status is true
func (nsa *NodeSolutionArchiveForTest) isInitializedOnNode(nodeName string) bool {
	return nsa.initializedStatusOnNode(nodeName, Equal("true"))
}

// isAvailableOnNode means: available status, has a download URL and file exists in /archives directory")
func (nsa *NodeSolutionArchiveForTest) isAvailableOnNode(nodeName string) bool {
	available := nsa.availableStatusOnNode(nodeName, Equal("true"))
	downloadURL := nsa.hasDownloadURLOnNode(nodeName, Not(BeEmpty()))
	fileExists := nsa.fileExistsOnNode(nodeName)

	return available && downloadURL && fileExists
}

// isUnavailableOnNode means: unavailable status, has no download URL and file does not exist in /archives directory")
func (nsa *NodeSolutionArchiveForTest) isUnavailableOnNode(nodeName string) bool {
	available := nsa.availableStatusOnNode(nodeName, Equal("false"))
	downloadURL := nsa.hasDownloadURLOnNode(nodeName, BeEmpty())
	fileExists := nsa.fileNotExistsOnNode(nodeName)

	return available && downloadURL && fileExists
}

// initializedStatusOnNode checks if the NodeSolutionArchive has initialized status for the given node.
func (nsa *NodeSolutionArchiveForTest) initializedStatusOnNode(nodeName string, expected types.GomegaMatcher) bool {
	timeout := 1 * time.Minute
	interval := 50 * time.Millisecond
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "nodesolutionarchive", fmt.Sprintf("na-%s-%s-%s",
			nsa.name, nsa.version, nodeName),
			"-o", "jsonpath={.status.initialized}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(expected)
	}, timeout, interval).Should(Succeed())

	return true
}

// availableStatusOnNode checks if the NodeSolutionArchive has available status for the given node.
func (nsa *NodeSolutionArchiveForTest) availableStatusOnNode(nodeName string, expected types.GomegaMatcher) bool {
	timeout := 1 * time.Minute
	interval := 500 * time.Millisecond
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "nodesolutionarchive", fmt.Sprintf("na-%s-%s-%s",
			nsa.name, nsa.version, nodeName),
			"-o", "jsonpath={.status.available}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(expected)
	}, timeout, interval).Should(Succeed())

	return true
}

// hasDownloadURLOnNode checks if the NodeSolutionArchive has a download URL for the given node.
func (nsa *NodeSolutionArchiveForTest) hasDownloadURLOnNode(nodeName string, expected types.GomegaMatcher) bool {
	cmd := exec.Command("kubectl", "get", "nodesolutionarchive", fmt.Sprintf("na-%s-%s-%s",
		nsa.name, nsa.version, nodeName),
		"-o", "jsonpath={.status.url}")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	Expect(output).To(expected)

	return true
}

// fileExistsOnNode checks if the ISO file exists on the given node.
func (nsa *NodeSolutionArchiveForTest) fileExistsOnNode(nodeName string) bool {
	solutionArchiveFilePath := fmt.Sprintf("/archives/%s-%s.iso", nsa.name, nsa.version)
	targetPodName := controllerPodNameByNode[nodeName]
	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
		"test", "-f", solutionArchiveFilePath)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	return true
}

// fileNotExistsOnNode checks if the ISO file does not exist on the given node.
func (nsa *NodeSolutionArchiveForTest) fileNotExistsOnNode(nodeName string) bool {
	solutionArchiveFilePath := fmt.Sprintf("/archives/%s-%s.iso", nsa.name, nsa.version)
	targetPodName := controllerPodNameByNode[nodeName]
	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
		"test", "-f", solutionArchiveFilePath)
	_, err := utils.Run(cmd)
	Expect(err).To(HaveOccurred())

	return true
}

// isServedOnNode means: served status and is mounted on /solutions directory.
func (nsa *NodeSolutionArchiveForTest) isServedOnNode(nodeName string) bool {
	served := nsa.servedStatusOnNode(nodeName, Equal("true"))
	mounted := nsa.mountPointOnNode(nodeName) != ""

	return served && mounted
}

// isNotServedOnNode means: not served status and is not mounted on /solutions directory.
func (nsa *NodeSolutionArchiveForTest) isNotServedOnNode(nodeName string) bool {
	served := nsa.servedStatusOnNode(nodeName, Equal("false"))
	mounted := nsa.isNotMountedOnNode(nodeName)

	return served && mounted
}

// servedStatusOnNode checks if the NodeSolutionArchive has expected served status for the given node.
func (nsa *NodeSolutionArchiveForTest) servedStatusOnNode(nodeName string, expected types.GomegaMatcher) bool {
	timeout := 1 * time.Minute
	interval := 500 * time.Millisecond
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "nodesolutionarchive", fmt.Sprintf("na-%s-%s-%s",
			nsa.name, nsa.version, nodeName),
			"-o", "jsonpath={.status.served}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(expected)
	}, timeout, interval).Should(Succeed())

	return true
}

// mountPointOnNode extracts the mount point of the NodeSolutionArchive for the given node.
func (nsa *NodeSolutionArchiveForTest) mountPointOnNode(nodeName string) (mountPoint string) {
	targetPodName := controllerPodNameByNode[nodeName]

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
		"sh", "-c", "mount | grep "+nsa.name+"/"+nsa.version+" || true")
	mountOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	Expect(mountOutput).To(ContainSubstring(nsa.name + "/" + nsa.version))

	// Extract mount point from output
	mountFields := strings.Fields(mountOutput)
	for i, field := range mountFields {
		if field == "on" && i+1 < len(mountFields) {
			mountPoint = mountFields[i+1]
			break
		}
	}
	Expect(mountPoint).NotTo(BeEmpty())
	Expect(strings.Contains(mountPoint, "/solutions")).To(BeTrue())

	return mountPoint
}

// isNotMountedOnNode checks if the NodeSolutionArchive is not mounted on the given node.
func (nsa *NodeSolutionArchiveForTest) isNotMountedOnNode(nodeName string) bool {
	targetPodName := controllerPodNameByNode[nodeName]

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPodName, "--",
		"sh", "-c", "mount | grep "+nsa.name+"/"+nsa.version+" || true")
	mountOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	Expect(mountOutput).NotTo(ContainSubstring(nsa.name + "/" + nsa.version))

	return true
}
