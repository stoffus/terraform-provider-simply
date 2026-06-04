// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/christopher/terraform-provider-simply/internal/services/dns_record"
	"github.com/christopher/terraform-provider-simply/internal/services/dns_zone"
	"github.com/christopher/terraform-provider-simply/internal/simply"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &SimplyProvider{}

type SimplyProvider struct {
	version string
}

type SimplyProviderModel struct {
	AccountName types.String `tfsdk:"account_name"`
	APIKey      types.String `tfsdk:"api_key"`
	Endpoint    types.String `tfsdk:"endpoint"`
	HTTPTimeout types.Int64  `tfsdk:"http_timeout"`
}

func (p *SimplyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "simply"
	resp.Version = p.version
}

func (p *SimplyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Simply provider manages Simply.com resources.",
		Attributes: map[string]schema.Attribute{
			"account_name": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Simply.com account name. Can also be set with `SIMPLY_ACCOUNT_NAME`.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Simply.com API key. Can also be set with `SIMPLY_API_KEY`.",
			},
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Simply.com API endpoint. Defaults to `https://api.simply.com/2`.",
			},
			"http_timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "HTTP timeout in seconds. Can also be set with `SIMPLY_HTTP_TIMEOUT`.",
			},
		},
	}
}

func (p *SimplyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SimplyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountName := valueOrEnv(data.AccountName, "SIMPLY_ACCOUNT_NAME")
	apiKey := valueOrEnv(data.APIKey, "SIMPLY_API_KEY")
	endpoint := valueOrDefault(data.Endpoint, "https://api.simply.com/2")
	timeoutSeconds := int64(30)

	if envTimeout := os.Getenv("SIMPLY_HTTP_TIMEOUT"); envTimeout != "" {
		parsed, err := strconv.ParseInt(envTimeout, 10, 64)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("http_timeout"), "Invalid SIMPLY_HTTP_TIMEOUT", fmt.Sprintf("Expected an integer number of seconds, got %q.", envTimeout))
			return
		}
		timeoutSeconds = parsed
	}
	if !data.HTTPTimeout.IsNull() && !data.HTTPTimeout.IsUnknown() {
		timeoutSeconds = data.HTTPTimeout.ValueInt64()
	}

	if accountName == "" {
		resp.Diagnostics.AddAttributeError(path.Root("account_name"), "Missing Simply Account Name", "Set account_name in the provider configuration or SIMPLY_ACCOUNT_NAME in the environment.")
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing Simply API Key", "Set api_key in the provider configuration or SIMPLY_API_KEY in the environment.")
	}
	if timeoutSeconds <= 0 {
		resp.Diagnostics.AddAttributeError(path.Root("http_timeout"), "Invalid HTTP Timeout", "HTTP timeout must be greater than zero.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client := simply.NewClient(simply.ClientConfig{
		AccountName: accountName,
		APIKey:      apiKey,
		Endpoint:    endpoint,
		Timeout:     time.Duration(timeoutSeconds) * time.Second,
	})

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *SimplyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		dnsrecord.NewResource,
	}
}

func (p *SimplyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		dnsrecord.NewDataSource,
		dnsrecord.NewListDataSource,
		dnszone.NewDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SimplyProvider{version: version}
	}
}

func valueOrEnv(value types.String, envName string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return os.Getenv(envName)
}

func valueOrDefault(value types.String, fallback string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return fallback
}
