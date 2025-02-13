package helper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/test-network-function/cnfcert-tests-verification/tests/accesscontrol/parameters"
	"github.com/test-network-function/cnfcert-tests-verification/tests/globalhelper"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/client"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/container"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/deployment"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/installplan"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/namespaces"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/resourcequota"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/service"
	"github.com/test-network-function/cnfcert-tests-verification/tests/utils/subscription"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteNamespaces(nsToDelete []string, clientSet *client.ClientSet, timeout time.Duration) error {
	failedNs := make(map[string]error)

	for _, namespace := range nsToDelete {
		err := namespaces.DeleteAndWait(
			clientSet,
			namespace,
			timeout,
		)
		if err != nil {
			failedNs[namespace] = err
		}
	}

	if len(failedNs) > 0 {
		return fmt.Errorf("failed to remove the following namespaces: %v", failedNs)
	}

	return nil
}

func DefineDeployment(replica int32, containers int, name string) (*appsv1.Deployment, error) {
	if containers < 1 {
		return nil, errors.New("invalid number of containers")
	}

	deploymentStruct := deployment.DefineDeployment(name, parameters.TestAccessControlNameSpace,
		globalhelper.Configuration.General.TestImage, parameters.TestDeploymentLabels)

	globalhelper.AppendContainersToDeployment(deploymentStruct, containers-1, globalhelper.Configuration.General.TestImage)
	deployment.RedefineWithReplicaNumber(deploymentStruct, replica)

	return deploymentStruct, nil
}

func DefineDeploymentWithClusterRoleBindingWithServiceAccount(replica int32, containers int, name string) (*appsv1.Deployment, error) {
	err := globalhelper.CreateClusterRoleBinding(parameters.TestAccessControlNameSpace, "my-service-account")
	if err != nil {
		return nil, err
	}

	deploymentStruct := deployment.DefineDeployment(name, parameters.TestAccessControlNameSpace,
		globalhelper.Configuration.General.TestImage, parameters.TestDeploymentLabels)

	globalhelper.AppendContainersToDeployment(deploymentStruct, containers-1, globalhelper.Configuration.General.TestImage)
	deployment.RedefineWithReplicaNumber(deploymentStruct, replica)
	deployment.AppendServiceAccount(deploymentStruct, "my-service-account")

	return deploymentStruct, nil
}

func DefineDeploymentWithNamespace(replica int32, containers int, name string, namespace string) (*appsv1.Deployment, error) {
	if containers < 1 {
		return nil, errors.New("invalid number of containers")
	}

	deploymentStruct := deployment.DefineDeployment(name, namespace,
		globalhelper.Configuration.General.TestImage, parameters.TestDeploymentLabels)

	globalhelper.AppendContainersToDeployment(deploymentStruct, containers-1, globalhelper.Configuration.General.TestImage)
	deployment.RedefineWithReplicaNumber(deploymentStruct, replica)

	return deploymentStruct, nil
}

func DefineDeploymentWithContainerPorts(name string, replicaNumber int32, ports []corev1.ContainerPort) (*appsv1.Deployment, error) {
	if len(ports) < 1 {
		return nil, errors.New("invalid number of containers")
	}

	deploymentStruct := deployment.DefineDeployment(name, parameters.TestAccessControlNameSpace,
		globalhelper.Configuration.General.TestImage, parameters.TestDeploymentLabels)

	globalhelper.AppendContainersToDeployment(deploymentStruct, len(ports)-1, globalhelper.Configuration.General.TestImage)
	deployment.RedefineWithReplicaNumber(deploymentStruct, replicaNumber)

	portSpecs := container.CreateContainerSpecsFromContainerPorts(ports,
		globalhelper.Configuration.General.TestImage, "test")

	deployment.RedefineWithContainerSpecs(deploymentStruct, portSpecs)

	return deploymentStruct, nil
}

func SetServiceAccountAutomountServiceAccountToken(namespace, saname, value string) error {
	var boolVal bool

	serviceacct, err := globalhelper.APIClient.ServiceAccounts(namespace).
		Get(context.TODO(), saname, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("error getting service account: %w", err)
	}

	switch value {
	case "true":
		boolVal = true
		serviceacct.AutomountServiceAccountToken = &boolVal

	case "false":
		boolVal = false
		serviceacct.AutomountServiceAccountToken = &boolVal

	case "nil":
		serviceacct.AutomountServiceAccountToken = nil

	default:
		return fmt.Errorf("invalid value for token value")
	}

	_, err = globalhelper.APIClient.ServiceAccounts(parameters.TestAccessControlNameSpace).
		Update(context.TODO(), serviceacct, metav1.UpdateOptions{})

	return err
}

func DefineAndCreateResourceQuota(namespace string, clientSet *client.ClientSet) error {
	quota := resourcequota.DefineResourceQuota("quota1", parameters.CPURequest, parameters.MemoryRequest,
		parameters.CPULimit, parameters.MemoryLimit)

	return namespaces.ApplyResourceQuota(namespace, clientSet, quota)
}

func DefineAndCreateInstallPlan(name, namespace string, clientSet *client.ClientSet) error {
	plan := installplan.DefineInstallPlan(name, namespace)

	return globalhelper.APIClient.Create(context.TODO(), plan)
}

func DefineAndCreateSubscription(name, namespace string, clientSet *client.ClientSet) error {
	subscription := subscription.DefineSubscription(name, namespace)

	return globalhelper.APIClient.Create(context.TODO(), subscription)
}

// DefineAndCreateServiceOnCluster defines service resource and creates it on cluster.
func DefineAndCreateServiceOnCluster(name string, port int32, targetPort int32, withNodePort bool,
	ipFams []corev1.IPFamily, ipFamPolicy string) error {
	var testService *corev1.Service

	if ipFamPolicy == "" {
		testService = service.DefineService(
			name,
			parameters.TestAccessControlNameSpace,
			port,
			targetPort,
			corev1.ProtocolTCP,
			parameters.TestDeploymentLabels,
			ipFams,
			nil)
	} else {
		ipPolicy := corev1.IPFamilyPolicy(ipFamPolicy)

		testService = service.DefineService(
			name,
			parameters.TestAccessControlNameSpace,
			port,
			targetPort,
			corev1.ProtocolTCP,
			parameters.TestDeploymentLabels,
			ipFams,
			&ipPolicy)
	}

	if withNodePort {
		var err error

		testService, err = service.RedefineWithNodePort(testService)
		if err != nil {
			return err
		}
	}

	_, err := globalhelper.APIClient.Services(parameters.TestAccessControlNameSpace).Create(
		context.Background(),
		testService, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service on cluster: %w", err)
	}

	return nil
}
