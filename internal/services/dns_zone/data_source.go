// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package dnszone

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

var _ datasource.DataSource = &DNSZoneDataSource{}

func NewDataSource() datasource.DataSource {
	return &DNSZoneDataSource{}
}

type DNSZoneDataSource struct {
	client *simply.Client
}

type dnsZoneDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Product types.String `tfsdk:"product"`
	Name    types.String `tfsdk:"name"`
}

func (d *DNSZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (d *DNSZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up Simply.com DNS zone metadata.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source ID.",
			},
			"product": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Simply.com product object, usually the domain name.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "DNS zone name.",
			},
		},
	}
}

func (d *DNSZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*simply.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *simply.Client, got: %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *DNSZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dnsZoneDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := d.client.GetDNSZone(ctx, data.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read DNS Zone", err.Error())
		return
	}

	data.ID = data.Product
	data.Name = types.StringValue(zone.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
