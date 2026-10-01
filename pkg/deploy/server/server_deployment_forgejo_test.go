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
	"context"
	"errors"
	"testing"

	"github.com/eclipse-che/che-operator/pkg/common/chetypes"
	"github.com/eclipse-che/che-operator/pkg/common/test"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var errListSecrets = errors.New("cannot list secrets")

// failSecretListsFrom makes every secret List call fail, starting from the given call number (1-based),
// and returns a pointer to the number of secret List calls made so far.
func failSecretListsFrom(cheCtx *chetypes.CheContext, failFrom int) *int {
	calls := 0
	cheCtx.ClusterAPI.Client = interceptor.NewClient(
		cheCtx.ClusterAPI.Client.(client.WithWatch),
		interceptor.Funcs{
			List: func(c context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*corev1.SecretList); ok {
					calls++
					if failFrom > 0 && calls >= failFrom {
						return errListSecrets
					}
				}
				return cl.List(c, list, opts...)
			},
		})
	return &calls
}

func TestMountForgejoOAuthConfigListError(t *testing.T) {
	cheCtx := test.NewCtxBuilder().Build()
	failSecretListsFrom(cheCtx, 1)

	err := MountForgejoOAuthConfig(cheCtx, &appsv1.Deployment{})

	assert.ErrorIs(t, err, errListSecrets)
}

func TestDeploymentSpecForgejoOAuthConfigError(t *testing.T) {
	// count the secret List calls made by the mounts performed before the Forgejo one
	cheCtx := test.NewCtxBuilder().Build()
	calls := failSecretListsFrom(cheCtx, 0)
	deployment, err := NewCheServerReconciler().getDeploymentSpec(test.NewCtxBuilder().Build())
	assert.NoError(t, err)
	assert.NoError(t, MountBitBucketOAuthConfig(cheCtx, deployment))
	assert.NoError(t, MountGitHubOAuthConfig(cheCtx, deployment))
	assert.NoError(t, MountGitLabOAuthConfig(cheCtx, deployment))
	callsBeforeForgejo := *calls

	// fail the Forgejo one
	cheCtx = test.NewCtxBuilder().Build()
	failSecretListsFrom(cheCtx, callsBeforeForgejo+1)

	deployment, err = NewCheServerReconciler().getDeploymentSpec(cheCtx)

	assert.ErrorIs(t, err, errListSecrets)
	assert.Nil(t, deployment)
}
