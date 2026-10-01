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

package v2

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/eclipse-che/che-operator/pkg/common/constants"
	k8shelper "github.com/eclipse-che/che-operator/pkg/common/k8s-helper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateScmSecrets(t *testing.T) {
	k8sHelper := k8shelper.GetInstance()

	githubSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "eclipse-che",
			Name:      "github-scm-secret",
		},
		Data: map[string][]byte{
			"id":     []byte("id"),
			"secret": []byte("secret"),
		},
	}
	_, err := k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Create(context.TODO(), githubSecret, metav1.CreateOptions{})
	assert.Nil(t, err)

	gitlabSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "eclipse-che",
			Name:      "gitlab-scm-secret",
			Annotations: map[string]string{
				constants.CheEclipseOrgScmServerEndpoint: "gitlab-endpoint-secret",
			},
		},
		Data: map[string][]byte{
			"id":     []byte("id"),
			"secret": []byte("secret"),
		},
	}
	_, err = k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Create(context.TODO(), gitlabSecret, metav1.CreateOptions{})
	assert.Nil(t, err)

	bitbucketSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "eclipse-che",
			Name:      "bitbucket-scm-secret",
		},
		Data: map[string][]byte{
			"private.key":  []byte("id"),
			"consumer.key": []byte("secret"),
		},
	}
	_, err = k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Create(context.TODO(), bitbucketSecret, metav1.CreateOptions{})
	assert.Nil(t, err)

	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			GitServices: CheClusterGitServices{
				GitHub: []GitHubService{
					{
						SecretName: "github-scm-secret",
						Endpoint:   "github-endpoint",
					},
				},
				GitLab: []GitLabService{
					{
						SecretName: "gitlab-scm-secret",
						Endpoint:   "gitlab-endpoint-checluster",
					},
				},
				BitBucket: []BitBucketService{
					{
						SecretName: "bitbucket-scm-secret",
					},
				},
			},
		},
	}

	cheClusterValidator := CheClusterValidator{}

	_, err = cheClusterValidator.ValidateCreate(context.TODO(), checluster)
	assert.Nil(t, err)

	githubSecret, err = k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Get(context.TODO(), "github-scm-secret", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "github", githubSecret.Annotations[constants.CheEclipseOrgOAuthScmServer])
	assert.Equal(t, "github-endpoint", githubSecret.Annotations[constants.CheEclipseOrgScmServerEndpoint])
	assert.Equal(t, constants.OAuthScmConfiguration, githubSecret.Labels[constants.KubernetesComponentLabelKey])
	assert.Equal(t, constants.CheEclipseOrg, githubSecret.Labels[constants.KubernetesPartOfLabelKey])

	gitlabSecret, err = k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Get(context.TODO(), "gitlab-scm-secret", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "gitlab", gitlabSecret.Annotations[constants.CheEclipseOrgOAuthScmServer])
	assert.Equal(t, "gitlab-endpoint-secret", gitlabSecret.Annotations[constants.CheEclipseOrgScmServerEndpoint])
	assert.Equal(t, constants.OAuthScmConfiguration, gitlabSecret.Labels[constants.KubernetesComponentLabelKey])
	assert.Equal(t, constants.CheEclipseOrg, gitlabSecret.Labels[constants.KubernetesPartOfLabelKey])

	bitbucketSecret, err = k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Get(context.TODO(), "bitbucket-scm-secret", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "bitbucket", bitbucketSecret.Annotations[constants.CheEclipseOrgOAuthScmServer])
	assert.Empty(t, bitbucketSecret.Annotations[constants.CheEclipseOrgScmServerEndpoint])
	assert.Equal(t, constants.OAuthScmConfiguration, bitbucketSecret.Labels[constants.KubernetesComponentLabelKey])
	assert.Equal(t, constants.CheEclipseOrg, bitbucketSecret.Labels[constants.KubernetesPartOfLabelKey])
}

func TestValidateOpenVSXClaimSize(t *testing.T) {
	cheClusterValidator := CheClusterValidator{}

	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			Components: CheClusterComponents{
				OpenVSXRegistry: OpenVSXRegistry{
					Database: &OpenVSXDatabase{
						Storage: &PVC{ClaimSize: "5Gi"},
					},
					Server: &OpenVSXServer{
						Storage: &PVC{ClaimSize: "4Gi"},
					},
				},
			},
		},
	}

	err := cheClusterValidator.validate(checluster)
	assert.NoError(t, err)
}

func TestValidateOpenVSXServerClaimSizeInvalid(t *testing.T) {
	cheClusterValidator := CheClusterValidator{}

	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			Components: CheClusterComponents{
				OpenVSXRegistry: OpenVSXRegistry{
					Server: &OpenVSXServer{
						Storage: &PVC{ClaimSize: "A"},
					},
				},
			},
		},
	}

	err := cheClusterValidator.validate(checluster)
	assert.Error(t, err)
}

func TestValidateOpenVSXSDatabaseClaimSizeInvalid(t *testing.T) {
	cheClusterValidator := CheClusterValidator{}

	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			Components: CheClusterComponents{
				OpenVSXRegistry: OpenVSXRegistry{
					Database: &OpenVSXDatabase{
						Storage: &PVC{ClaimSize: "B"},
					},
				},
			},
		},
	}

	err := cheClusterValidator.validate(checluster)
	assert.Error(t, err)
}

func TestValidateOpenVSXCredentialsSecret(t *testing.T) {
	k8sHelper := k8shelper.GetInstance()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "eclipse-che",
			Name:      "openvsx-credentials",
		},
		Data: map[string][]byte{
			"database-user":           []byte("user"),
			"database-password":       []byte("password"),
			"database-name":           []byte("openvsx"),
			"openvsx-publisher-name":  []byte("pub"),
			"openvsx-publisher-token": []byte("pub-token"),
			"openvsx-admin-name":      []byte("admin"),
			"openvsx-admin-token":     []byte("admin-token"),
		},
	}
	_, err := k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Create(context.TODO(), secret, metav1.CreateOptions{})
	assert.NoError(t, err)

	cheClusterValidator := CheClusterValidator{}
	credentialsSecretName := "openvsx-credentials"
	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			Components: CheClusterComponents{
				OpenVSXRegistry: OpenVSXRegistry{
					CredentialsSecretName: &credentialsSecretName,
				},
			},
		},
	}

	err = cheClusterValidator.validate(checluster)
	assert.NoError(t, err)
}

func TestValidateOpenVSXCredentialsSecretMissingKeys(t *testing.T) {
	k8sHelper := k8shelper.GetInstance()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "eclipse-che",
			Name:      "openvsx-credentials-incomplete",
		},
		Data: map[string][]byte{
			"database-user": []byte("user"),
		},
	}
	_, err := k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Create(context.TODO(), secret, metav1.CreateOptions{})
	assert.NoError(t, err)

	cheClusterValidator := CheClusterValidator{}
	credentialsSecretName := "openvsx-credentials-incomplete"
	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			Components: CheClusterComponents{
				OpenVSXRegistry: OpenVSXRegistry{
					CredentialsSecretName: &credentialsSecretName,
				},
			},
		},
	}

	err = cheClusterValidator.validate(checluster)
	assert.Error(t, err)
}

func TestValidateScmSecretsShouldThrowError(t *testing.T) {
	checluster := &CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: "eclipse-che",
		},
		Spec: CheClusterSpec{
			GitServices: CheClusterGitServices{
				GitHub: []GitHubService{
					{
						SecretName: "github-scm-secret-with-errors",
					},
				},
			},
		},
	}

	cheClusterValidator := CheClusterValidator{}
	_, err := cheClusterValidator.ValidateCreate(context.TODO(), checluster)
	assert.Error(t, err)
	assert.Equal(t, "secret 'github-scm-secret-with-errors' not found", err.Error())

	githubSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "eclipse-che",
			Name:      "github-scm-secret-with-errors",
		},
	}

	k8sHelper := k8shelper.GetInstance()
	_, err = k8sHelper.GetClientSet().CoreV1().Secrets("eclipse-che").Create(context.TODO(), githubSecret, metav1.CreateOptions{})
	assert.Nil(t, err)

	_, err = cheClusterValidator.ValidateCreate(context.TODO(), checluster)
	assert.Error(t, err)
	assert.Equal(t, "mandatory keys [id, secret] not found in secret github-scm-secret-with-errors", err.Error())
}

func TestValidateForgejoScmSecrets(t *testing.T) {
	k8sHelper := k8shelper.GetInstance()
	namespace := "eclipse-che-forgejo"

	newCheCluster := func(secretNames ...string) *CheCluster {
		forgejo := []ForgejoService{}
		for _, secretName := range secretNames {
			forgejo = append(forgejo, ForgejoService{SecretName: secretName})
		}
		return &CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "eclipse-che",
				Namespace: namespace,
			},
			Spec: CheClusterSpec{
				GitServices: CheClusterGitServices{
					Forgejo: forgejo,
				},
			},
		}
	}

	createSecret := func(name string, annotations map[string]string, data map[string][]byte) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespace,
				Name:        name,
				Annotations: annotations,
			},
			Data: data,
		}
		_, err := k8sHelper.GetClientSet().CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	deleteSecret := func(name string) {
		err := k8sHelper.GetClientSet().CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		assert.NoError(t, err)
	}

	validData := map[string][]byte{
		"id":     []byte("id"),
		"secret": []byte("secret"),
	}

	cheClusterValidator := CheClusterValidator{}

	t.Run("missing keys", func(t *testing.T) {
		createSecret("forgejo-missing-keys",
			map[string]string{constants.CheEclipseOrgScmServerEndpoint: "https://forgejo.example.com"},
			map[string][]byte{"id": []byte("id")})

		defer deleteSecret("forgejo-missing-keys")

		_, err := cheClusterValidator.ValidateUpdate(context.TODO(), nil, newCheCluster("forgejo-missing-keys"))
		assert.Error(t, err)
		assert.Equal(t, "mandatory keys [id, secret] not found in secret forgejo-missing-keys", err.Error())
	})

	t.Run("missing endpoint annotation", func(t *testing.T) {
		createSecret("forgejo-missing-endpoint", nil, validData)

		defer deleteSecret("forgejo-missing-endpoint")

		_, err := cheClusterValidator.ValidateUpdate(context.TODO(), nil, newCheCluster("forgejo-missing-endpoint"))
		assert.Error(t, err)
		assert.Equal(t, "annotation 'che.eclipse.org/scm-server-endpoint' not found in secret forgejo-missing-endpoint", err.Error())
	})

	t.Run("valid secret", func(t *testing.T) {
		createSecret("forgejo-valid",
			map[string]string{constants.CheEclipseOrgScmServerEndpoint: "https://forgejo.example.com"},
			validData)

		warnings, err := cheClusterValidator.ValidateUpdate(context.TODO(), nil, newCheCluster("forgejo-valid"))
		assert.NoError(t, err)
		assert.Empty(t, warnings)

		secret, err := k8sHelper.GetClientSet().CoreV1().Secrets(namespace).Get(context.TODO(), "forgejo-valid", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "forgejo", secret.Annotations[constants.CheEclipseOrgOAuthScmServer])
		assert.Equal(t, "https://forgejo.example.com", secret.Annotations[constants.CheEclipseOrgScmServerEndpoint])
		assert.Equal(t, constants.OAuthScmConfiguration, secret.Labels[constants.KubernetesComponentLabelKey])
		assert.Equal(t, constants.CheEclipseOrg, secret.Labels[constants.KubernetesPartOfLabelKey])
	})

	t.Run("more than two secrets", func(t *testing.T) {
		createSecret("forgejo-valid-2",
			map[string]string{constants.CheEclipseOrgScmServerEndpoint: "https://forgejo-2.example.com"},
			validData)
		createSecret("forgejo-valid-3",
			map[string]string{constants.CheEclipseOrgScmServerEndpoint: "https://forgejo-3.example.com"},
			validData)

		warnings, err := cheClusterValidator.ValidateUpdate(context.TODO(), nil, newCheCluster("forgejo-valid", "forgejo-valid-2"))
		assert.NoError(t, err)
		assert.Empty(t, warnings)

		warnings, err = cheClusterValidator.ValidateUpdate(context.TODO(), nil, newCheCluster("forgejo-valid", "forgejo-valid-2", "forgejo-valid-3"))
		assert.NoError(t, err)
		assert.Equal(t, []string{"3 Forgejo OAuth secrets found, only the first 2 (sorted by 'che.eclipse.org/scm-server-endpoint' annotation) are used"}, []string(warnings))
	})
}

func TestForgejoWarningsWhenSecretsCannotBeListed(t *testing.T) {
	clientSet := k8shelper.GetInstance().GetClientSet().(*fake.Clientset)
	reactionChain := clientSet.ReactionChain
	defer func() { clientSet.ReactionChain = reactionChain }()
	clientSet.PrependReactor("list", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("cannot list secrets")
	})

	warnings, err := (&CheClusterValidator{}).ValidateUpdate(context.TODO(), nil, &CheCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "eclipse-che", Namespace: "eclipse-che-forgejo-list-error"},
	})

	assert.NoError(t, err)
	assert.Empty(t, warnings)
}
