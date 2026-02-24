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
	"crypto/tls"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

const certDuration = 365 * 24 * time.Hour // 1 year

func generateCertificates(svcName, svcNamespace string) (caCertPEM []byte, tlsCert tls.Certificate, err error) {
	caKeyPair, err := triple.NewCA("vdpa-webhook.kubevirt.io", certDuration)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("failed to create CA: %v", err)
	}
	caCertPEM = cert.EncodeCertPEM(caKeyPair.Cert)

	serverKeyPair, err := triple.NewServerKeyPair(
		caKeyPair,
		fmt.Sprintf("%s.%s.pod.cluster.local", svcName, svcNamespace),
		svcName,
		svcNamespace,
		"cluster.local",
		nil,
		nil,
		certDuration,
	)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("create server cert: %v", err)
	}

	serverCertPEM := cert.EncodeCertPEM(serverKeyPair.Cert)
	serverKeyPEM := cert.EncodePrivateKeyPEM(serverKeyPair.Key)

	tlsCert, err = tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("load TLS key pair: %v", err)
	}

	return caCertPEM, tlsCert, nil
}
