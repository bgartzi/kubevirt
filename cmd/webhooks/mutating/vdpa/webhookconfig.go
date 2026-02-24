/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kubevirt.io/client-go/log"
)

func createOrUpdateWebhookConfiguration(caCertPEM []byte, svcName, svcNamespace string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("not running in cluster: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %v", err)
	}

	failPolicy := admissionregistrationv1.Fail
	sideEffect := admissionregistrationv1.SideEffectClassNone
	path := webhookPath

	mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				"app": webhookName,
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
			Name:                    webhookRegName,
			AdmissionReviewVersions: []string{"v1"},
			SideEffects:             &sideEffect,
			FailurePolicy:           &failPolicy,
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				CABundle: caCertPEM,
				Service: &admissionregistrationv1.ServiceReference{
					Name:      svcName,
					Namespace: svcNamespace,
					Path:      &path,
				},
			},
			Rules: []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{"kubevirt.io"},
					APIVersions: []string{"v1"},
					Resources:   []string{"virtualmachines"},
				},
			}},
		}},
	}

	client := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations()
	foundWebhookConfig, err := client.Get(context.Background(), webhookName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := client.Create(context.Background(), mutatingWebhookConfig, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create webhook config: %v", err)
		}
		log.Log.V(2).Infof("Created MutatingWebhookConfiguration: %s", webhookName)
		return nil

	} else if err != nil {
		return fmt.Errorf("failed to get webhook config: %v", err)
	}

	mutatingWebhookConfig.ObjectMeta.ResourceVersion = foundWebhookConfig.ObjectMeta.ResourceVersion
	if _, err := client.Update(context.Background(), mutatingWebhookConfig, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update webhook config: %v", err)
	}
	log.Log.V(2).Infof("Updated MutatingWebhookConfiguration: %s", webhookName)
	return nil
}
