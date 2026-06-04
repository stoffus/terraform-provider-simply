// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package acctest

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stoffus/terraform-provider-simply/internal/provider"
)

var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"simply": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func PreCheck(t *testing.T) {}
