/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

// Package webhook embeds the webhook YAML files
package webhook

import _ "embed"

var (
	//go:embed certificate.yaml
	Certificate []byte

	//go:embed deployment.yaml
	Deployment []byte

	//go:embed service.yaml
	Service []byte

	//go:embed self_signed_issuer.yaml
	SelfSignedIssuer []byte

	//go:embed validating_webhook_config.yaml
	ValidatingWebhookConfig []byte
)
