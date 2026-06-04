// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package acctest

import (
	"testing"

	"github.com/christopher/terraform-provider-simply/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"simply": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func PreCheck(t *testing.T) {}
