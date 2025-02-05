// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shootsecrets

import (
	"context"

	gardencorev1alpha1helper "github.com/gardener/gardener/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/gardener/pkg/utils/infodata"
	"github.com/gardener/gardener/pkg/utils/secrets"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretConfigGeneratorFunc is a func used to generate secret configurations
type SecretConfigGeneratorFunc func(map[string]*secrets.Certificate) ([]secrets.ConfigInterface, error)

// SecretsManager holds the configurations of all required shoot secrets that have to be preserved in the ShootState.
// It uses these configurations to generate new secret infodata and save it into the ShootState
// or create kubernetes secret objects from infodata available in the ShootState and deploy them.
type SecretsManager struct {
	secretConfigGenerator SecretConfigGeneratorFunc

	certificateAuthorities map[string]*secrets.Certificate

	existingSecrets map[string]*corev1.Secret

	GardenerResourceDataList gardencorev1alpha1helper.GardenerResourceDataList
	StaticToken              *secrets.StaticToken
	DeployedSecrets          map[string]*corev1.Secret
}

// NewSecretsManager takes in a list of GardenerResourceData items, a static token secret config, a map of certificate authority configs,
// a function which can generate secret configurations and returns a new SecretsManager struct
func NewSecretsManager(
	gardenerResourceDataList gardencorev1alpha1helper.GardenerResourceDataList,
	secretConfigGenerator SecretConfigGeneratorFunc,
) *SecretsManager {
	return &SecretsManager{
		GardenerResourceDataList: gardenerResourceDataList,
		secretConfigGenerator:    secretConfigGenerator,
		certificateAuthorities:   make(map[string]*secrets.Certificate),
		existingSecrets:          map[string]*corev1.Secret{},
		DeployedSecrets:          map[string]*corev1.Secret{},
	}
}

// WithExistingSecrets adds the provided map of existing secrets to the SecretsManager
func (s *SecretsManager) WithExistingSecrets(existingSecrets map[string]*corev1.Secret) *SecretsManager {
	s.existingSecrets = existingSecrets
	return s
}

// WithCertificateAuthorities adds the provided map of CA secrets to the SecretsManager
func (s *SecretsManager) WithCertificateAuthorities(cas map[string]*secrets.Certificate) *SecretsManager {
	s.certificateAuthorities = cas
	return s
}

// Generate generates InfoData for all shoot secrets managed by gardener and adds it to the SecretManager's
// GardenerResourceData
func (s *SecretsManager) Generate() error {
	secretConfigs, err := s.secretConfigGenerator(s.certificateAuthorities)
	if err != nil {
		return err
	}

	for _, config := range secretConfigs {
		if err := s.generateInfoDataAndUpdateResourceList(config); err != nil {
			return err
		}
	}

	return nil
}

// Deploy gets InfoData for all shoot secrets managed by gardener from the SecretManager's GardenerResourceDataList
// and uses it to generate kubernetes secrets and deploy them in the provided namespace.
func (s *SecretsManager) Deploy(ctx context.Context, k8sClient client.Client, namespace string) error {
	if s.secretConfigGenerator == nil {
		return nil
	}

	secretConfigs, err := s.secretConfigGenerator(s.certificateAuthorities)
	if err != nil {
		return err
	}

	deployedSecrets, err := secrets.GenerateClusterSecretsWithFunc(ctx, k8sClient, s.existingSecrets, secretConfigs, namespace, func(c secrets.ConfigInterface) (secrets.DataInterface, error) {
		return s.getInfoDataAndGenerateSecret(c)
	})
	if err != nil {
		return err
	}

	for name, secret := range deployedSecrets {
		s.DeployedSecrets[name] = secret
	}

	return nil
}

func (s *SecretsManager) generateInfoDataAndUpdateResourceList(secretConfig secrets.ConfigInterface) error {
	if s.GardenerResourceDataList.Get(secretConfig.GetName()) != nil {
		return nil
	}
	data, err := secretConfig.GenerateInfoData()
	if err != nil {
		return err
	}
	return infodata.UpsertInfoData(&s.GardenerResourceDataList, secretConfig.GetName(), data)
}

func (s *SecretsManager) getInfoDataAndGenerateSecret(secretConfig secrets.ConfigInterface) (secrets.DataInterface, error) {
	secretInfoData, err := infodata.GetInfoData(s.GardenerResourceDataList, secretConfig.GetName())
	if err != nil {
		return nil, err
	}
	if secretInfoData == nil {
		return secretConfig.Generate()
	}

	return secretConfig.GenerateFromInfoData(secretInfoData)
}

func (s *SecretsManager) deploySecret(ctx context.Context, k8sClient client.Client, namespace string, secretInterface secrets.DataInterface, secretName string) (*corev1.Secret, error) {
	if secret, ok := s.existingSecrets[secretName]; ok {
		return secret, nil
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretInterface.SecretData(),
	}

	if err := k8sClient.Create(ctx, secret); err != nil {
		return nil, err
	}
	return secret, nil
}
