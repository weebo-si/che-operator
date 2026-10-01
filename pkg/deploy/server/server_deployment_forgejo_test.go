//
// Copyright (c) 2019-2025 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"runtime"
	"strings"
	"testing"

	"github.com/eclipse-che/che-operator/pkg/common/chetypes"
	"github.com/eclipse-che/che-operator/pkg/common/test"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var errListSecrets = errors.New("cannot list secrets")

// failSecretListsIn makes every secret List call fail when it is made from the given function.
func failSecretListsIn(ctx *chetypes.DeployContext, funcName string) {
	ctx.ClusterAPI.Client = interceptor.NewClient(
		ctx.ClusterAPI.Client.(client.WithWatch),
		interceptor.Funcs{
			List: func(c context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*corev1.SecretList); ok && isCalledFrom(funcName) {
					return errListSecrets
				}
				return cl.List(c, list, opts...)
			},
		})
}

// isCalledFrom reports whether the given function is in the current call stack.
func isCalledFrom(funcName string) bool {
	pc := make([]uintptr, 64)
	frames := runtime.CallersFrames(pc[:runtime.Callers(2, pc)])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, "."+funcName) {
			return true
		}
		if !more {
			return false
		}
	}
}

func TestMountForgejoOAuthConfigListError(t *testing.T) {
	ctx := test.NewCtxBuilder().Build()
	failSecretListsIn(ctx, "MountForgejoOAuthConfig")

	err := MountForgejoOAuthConfig(ctx, &appsv1.Deployment{})

	assert.ErrorIs(t, err, errListSecrets)
}

func TestDeploymentSpecForgejoOAuthConfigError(t *testing.T) {
	ctx := test.NewCtxBuilder().Build()
	failSecretListsIn(ctx, "MountForgejoOAuthConfig")

	deployment, err := NewCheServerReconciler().getDeploymentSpec(ctx)

	assert.ErrorIs(t, err, errListSecrets)
	assert.Nil(t, deployment)
}

func TestMountForgejoOAuthConfigLogsWhenTooManySecrets(t *testing.T) {
	newForgejoSecret := func(name string, endpoint string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "eclipse-che",
				Labels: map[string]string{
					"app.kubernetes.io/part-of":   "che.eclipse.org",
					"app.kubernetes.io/component": "oauth-scm-configuration",
				},
				Annotations: map[string]string{
					"che.eclipse.org/oauth-scm-server":    "forgejo",
					"che.eclipse.org/scm-server-endpoint": endpoint,
				},
			},
		}
	}

	captureLogs := func(t *testing.T) *bytes.Buffer {
		buf := &bytes.Buffer{}
		originalLog := log
		log = logr.FromSlogHandler(slog.NewTextHandler(buf, nil))
		t.Cleanup(func() { log = originalLog })
		return buf
	}

	newDeployment := func() *appsv1.Deployment {
		return &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{}}}}}}
	}

	warning := "che-server only reads the first 2"

	t.Run("two secrets", func(t *testing.T) {
		logs := captureLogs(t)
		ctx := test.NewCtxBuilder().
			WithObjects(newForgejoSecret("forgejo-oauth-config-a", "https://forgejo-a.example.com")).
			WithObjects(newForgejoSecret("forgejo-oauth-config-b", "https://forgejo-b.example.com")).
			WithObjects(newForgejoSecret("forgejo-oauth-config-no-endpoint", "")).
			Build()

		assert.NoError(t, MountForgejoOAuthConfig(ctx, newDeployment()))
		assert.NotContains(t, logs.String(), warning)
	})

	t.Run("three secrets", func(t *testing.T) {
		logs := captureLogs(t)
		ctx := test.NewCtxBuilder().
			WithObjects(newForgejoSecret("forgejo-oauth-config-a", "https://forgejo-a.example.com")).
			WithObjects(newForgejoSecret("forgejo-oauth-config-b", "https://forgejo-b.example.com")).
			WithObjects(newForgejoSecret("forgejo-oauth-config-c", "https://forgejo-c.example.com")).
			Build()

		assert.NoError(t, MountForgejoOAuthConfig(ctx, newDeployment()))
		assert.Contains(t, logs.String(), "3 Forgejo OAuth secrets found, "+warning)
	})
}
