/*
Copyright The Kubernetes NMState Authors.


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

package handler

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("NNCP cleanup", func() {

	BeforeEach(func() {
		By("Create a policy")
		setDesiredStateWithPolicy(bridge1, linuxBrUp(bridge1))

		By("Wait for policy to be ready")
		policy.WaitForAvailablePolicy(bridge1)
	})

	AfterEach(func() {
		deletePolicy(bridge1)
		updateDesiredStateAndWait(linuxBrAbsent(bridge1))
		resetDesiredStateForNodes()
	})

	Context("when a policy is deleted", func() {
		BeforeEach(func() {
			By("Delete the policy")
			deletePolicy(bridge1)
		})

		AfterEach(func() {
			deletePolicy(bridge1)
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})

		It("should also delete nodes enactments", func() {
			for _, node := range nodes {
				Eventually(func() bool {
					key := nmstate.EnactmentKey(node, bridge1)
					enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
					err := testenv.Client.Get(context.TODO(), key, &enactment)
					return errors.IsNotFound(err)
				}, 10*time.Second, 1*time.Second).Should(BeTrue(), "Enactment has not being deleted")
			}
		})
	})

	Context("when a policy is deleted while one of the nodes is down", func() {
		var restartedNode string

		BeforeEach(func() {
			restartedNode = nodes[0]
			restartNodeWithoutWaiting(restartedNode)

			By("Delete the policy")
			deletePolicy(bridge1)
		})

		It("should also delete nodes enactments", func() {
			for _, node := range nodes {
				if node == restartedNode {
					continue
				}
				verifyEnactmentRemoved(node, 10*time.Second)
			}

			waitForNodeToStart(restartedNode)
			verifyEnactmentRemoved(restartedNode, 4*time.Minute)
		})
	})
})

func verifyEnactmentRemoved(node string, timeout time.Duration) {
	Eventually(func() bool {
		key := nmstate.EnactmentKey(node, bridge1)
		enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		err := testenv.Client.Get(context.TODO(), key, &enactment)
		return errors.IsNotFound(err)
	}, timeout, 1*time.Second).Should(BeTrue(), "Enactment has not being deleted")
}
