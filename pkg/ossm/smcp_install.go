// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ossm

import (
	"strings"
	"fmt"
	"testing"
	"time"

        "os"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func installDefaultSMCP25() {
	util.Log.Info("Create SMCP v2.5 in ", meshNamespace)
	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
	util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV25_template, smcp))
	util.KubeApplyContents(meshNamespace, smmr)

	// patch SMCP identity if it's on a ROSA cluster
	if util.Getenv("ROSA", "false") == "true" {
		util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
	}
	util.Log.Info("Waiting for mesh installation to complete")
	util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)
	time.Sleep(time.Duration(20) * time.Second)
}

func installDefaultSMCP24() {
	util.Log.Info("Create SMCP v2.4 in ", meshNamespace)
	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
	util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV24_template, smcp))
	util.KubeApplyContents(meshNamespace, smmr)

	// patch SMCP identity if it's on a ROSA cluster
	if util.Getenv("ROSA", "false") == "true" {
		util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
	}
	util.Log.Info("Waiting for mesh installation to complete")
	util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)
	time.Sleep(time.Duration(20) * time.Second)
}

func changeSvcToLB() {
    util.Log.Info("Changing service type from ClusterIP to LoadBalancer")
    time.Sleep(time.Duration(20) * time.Second)

    // Change service type to LoadBalancer
    util.Shell(`oc patch svc istio-ingressgateway -n istio-system -p '{"spec": {"type": "LoadBalancer"}}'`)

    // Get the HOST NAME
    host_name, err := util.Shell(`oc get route -n istio-system istio-ingressgateway -o jsonpath='{.spec.host}'`)
    util.Log.Info("Retrieved Host Name of route istio-ingressgateway: ", host_name)
    if err != nil {
        util.Log.Error("Failed to get Host Name of route istio-ingressgateway: ", err)
        os.Exit(1)
    }

    // Get the EXTERNAL IP
    external_ip, err := util.Shell(`oc get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`)
    util.Log.Info("Retrieved External IP of svc istio-ingressgateway: ", external_ip)
    if err != nil {
        util.Log.Error("Failed to get External IP of svc istio-ingressgateway: ", err)
        os.Exit(1)
    }

    // Update /etc/hosts
    util.Log.Info("Updating /etc/hosts file with External IP and Host Name entry for DNS resolution")
    existingEntry, _ := util.Shell(fmt.Sprintf(`grep "%s" /etc/hosts`, host_name))

    if existingEntry == "" {
        // If the hostname doesn't exist, append it
        _, err := util.Shell(fmt.Sprintf(`echo "%s %s" | sudo tee -a /etc/hosts`, external_ip, host_name))
        if err != nil {
            util.Log.Error("Failed to append /etc/hosts file with External IP and Host Name entry: ", err)
	    os.Exit(1)
	}
    } else {
        // If it exists, update the line
        _, err := util.Shell(fmt.Sprintf(`sudo sed -i 's/.*%s$/%s %s/' /etc/hosts`, host_name, external_ip, host_name))
        if err != nil {
            util.Log.Error("Failed to update /etc/hosts file with External IP and Host Name entry ", err)
	    os.Exit(1)
	}
    }
}

func TestSMCPInstall(t *testing.T) {
	defer changeSvcToLB()
	defer installDefaultSMCP25()

	t.Run("smcp_test_install_2.5", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.5 in ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV25_template, smcp))
		util.KubeApplyContents(meshNamespace, smmr)

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_uninstall_2.5", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.5 in ", meshNamespace)
		util.KubeDeleteContents(meshNamespace, smmr)
		util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV25_template, smcp))
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_install_2.4", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.4 in ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV24_template, smcp))
		util.KubeApplyContents(meshNamespace, smmr)

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_uninstall_2.4", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.4 in ", meshNamespace)
		util.KubeDeleteContents(meshNamespace, smmr)
		util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV24_template, smcp))
		time.Sleep(time.Duration(40) * time.Second)
	})

	t.Run("smcp_test_install_2.3", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Create SMCP v2.3 in namespace ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template, smcp))
		util.KubeApplyContents(meshNamespace, smmr)

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_uninstall_2.3", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete SMCP v2.3 in ", meshNamespace)
		util.KubeDeleteContents(meshNamespace, smmr)
		util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV23_template, smcp))
		time.Sleep(time.Duration(60) * time.Second)
	})

	t.Run("smcp_test_upgrade_2.3_to_2.4", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Create SMCP v2.3 in namespace ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template, smcp))
		util.KubeApplyContents(meshNamespace, smmr)

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		util.Shell(`oc get -n %s pods`, meshNamespace)

		util.Log.Info("Upgrade SMCP to v2.4")
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV24_template, smcp))

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		time.Sleep(time.Duration(10) * time.Second)
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 360s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})

	t.Run("smcp_test_upgrade_2.4_to_2.5", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Create SMCP v2.4 in namespace ", meshNamespace)
		util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV24_template, smcp))
		util.KubeApplyContents(meshNamespace, smmr)

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 300s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		util.Shell(`oc get -n %s pods`, meshNamespace)

		util.Log.Info("Upgrade SMCP to v2.5")
		util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV25_template, smcp))

		// patch SMCP identity if it's on a ROSA cluster
		if util.Getenv("ROSA", "false") == "true" {
			util.Shell(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}'`, meshNamespace, smcpName)
		}
		util.Log.Info("Waiting for mesh installation to complete")
		time.Sleep(time.Duration(10) * time.Second)
		util.Shell(`oc wait --for condition=Ready -n %s smcp/%s --timeout 360s`, meshNamespace, smcpName)

		util.Log.Info("Verify SMCP status and pods")
		msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
		if !strings.Contains(msg, "ComponentsReady") {
			util.Log.Error("SMCP not Ready")
			t.Error("SMCP not Ready")
		}
		util.Shell(`oc get -n %s pods`, meshNamespace)
	})
}
