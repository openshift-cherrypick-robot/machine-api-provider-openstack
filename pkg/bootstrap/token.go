/*
Copyright 2018 The Kubernetes Authors.

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
package bootstrap

import (
	"context"
	"fmt"
	"shiftstack/machine-api-provider-openstack/pkg/cloud/openstack/options"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tokenapi "k8s.io/cluster-bootstrap/token/api"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GeneratesTokenSecret returns a Secret conform to kubeadms bootstrap tokens
// Inspired by https://github.com/kubernetes/kubernetes/blob/03a145de8ad282764828f43821433001974718e9/cmd/kubeadm/app/apis/kubeadm/bootstraptokenhelpers.go#L34
// and the underlying type BootstrapToken.
// We might change the implementation if a type BootstrapToken hits client-go.
func generateTokenSecret(token string, expiration time.Time) (*v1.Secret, error) {
	substrs := tokenutil.BootstrapTokenRegexp.FindStringSubmatch(token)
	if len(substrs) != 3 {
		return nil, fmt.Errorf("the bootstrap token %q was not in the form %q", token, tokenapi.BootstrapTokenPattern)
	}
	tokenID := substrs[1]
	tokenSecret := substrs[2]

	expirationStr := expiration.Format(time.RFC3339)

	data := map[string][]byte{
		tokenapi.BootstrapTokenIDKey:               []byte(tokenID),
		tokenapi.BootstrapTokenSecretKey:           []byte(tokenSecret),
		tokenapi.BootstrapTokenExpirationKey:       []byte(expirationStr),
		tokenapi.BootstrapTokenUsageAuthentication: []byte("true"),
		tokenapi.BootstrapTokenUsageSigningKey:     []byte("true"),
		tokenapi.BootstrapTokenExtraGroupsKey:      []byte("system:bootstrappers:kubeadm:default-node-token"),
		tokenapi.BootstrapTokenDescriptionKey:      []byte("bootstrap token generated by cluster-api-provider-openstack"),
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tokenutil.BootstrapTokenSecretName(tokenID),
			Namespace: metav1.NamespaceSystem,
		},
		Type: v1.SecretType(tokenapi.SecretTypeBootstrapToken),
		Data: data,
	}, nil
}

func CreateBootstrapToken(client client.Client) (string, error) {
	token, err := tokenutil.GenerateBootstrapToken()
	if err != nil {
		return "", err
	}

	expiration := time.Now().UTC().Add(options.TokenTTL)
	tokenSecret, err := generateTokenSecret(token, expiration)
	if err != nil {
		panic(fmt.Sprintf("unable to create token. there might be a bug somwhere: %v", err))
	}

	err = client.Create(context.TODO(), tokenSecret)
	if err != nil {
		return "", err
	}

	return tokenutil.TokenFromIDAndSecret(
		string(tokenSecret.Data[tokenapi.BootstrapTokenIDKey]),
		string(tokenSecret.Data[tokenapi.BootstrapTokenSecretKey]),
	), nil
}
