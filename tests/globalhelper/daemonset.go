package globalhelper

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/gomega"
)

// CreateAndWaitUntilDaemonSetIsReady creates daemonSet and waits until all pods are up and running.
func CreateAndWaitUntilDaemonSetIsReady(daemonSet *appsv1.DaemonSet, timeout time.Duration) error {
	runningDaemonSet, err := APIClient.DaemonSets(daemonSet.Namespace).Create(
		context.Background(), daemonSet, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create daemonset %q (ns %s): %w", daemonSet.Name, daemonSet.Namespace, err)
	}

	Eventually(func() bool {
		status, err := isDaemonSetReady(runningDaemonSet.Namespace, runningDaemonSet.Name)
		if err != nil {
			glog.Fatal(fmt.Sprintf(
				"daemonset %s is not ready, retry in 5 seconds", runningDaemonSet.Name))

			return false
		}

		return status
	}, timeout, retryInterval*time.Second).Should(Equal(true), "DaemonSet is not ready")

	return nil
}

func isDaemonSetReady(namespace string, name string) (bool, error) {
	daemonSet, err := APIClient.DaemonSets(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		return false, err
	}

	if daemonSet.Status.NumberReady > 0 && daemonSet.Status.NumberUnavailable == 0 {
		return true, nil
	}

	return false, nil
}
