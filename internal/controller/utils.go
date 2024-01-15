package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	hyperdhcpv1beta1 "github.com/cldmnky/hyperdhcp/api/v1beta1"
)

func CreateOrUpdateWithRetries(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	f controllerutil.MutateFn,
) (controllerutil.OperationResult, error) {
	var operationResult controllerutil.OperationResult
	updateErr := wait.ExponentialBackoff(retry.DefaultBackoff, func() (ok bool, err error) {
		operationResult, err = controllerutil.CreateOrUpdate(ctx, c, obj, f)
		if err == nil {
			return true, nil
		}
		if !apierrors.IsConflict(err) {
			return false, err
		}
		return false, nil
	})
	return operationResult, updateErr
}

func makeLabels(name string, server *hyperdhcpv1beta1.Server) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/version":    "0.0.1",
		"app.kubernetes.io/managed-by": "ksdns",
		"app.kubernetes.io/instance":   server.Name,
	}
}
