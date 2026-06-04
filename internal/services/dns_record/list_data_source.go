// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package dnsrecord

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

var _ datasource.DataSource = &DNSRecordsDataSource{}

func NewListDataSource() datasource.DataSource {
	return &DNSRecordsDataSource{}
}

type DNSRecordsDataSource struct {
	client *simply.Client
}

type dnsRecordsDataSourceModel struct {
	ID      types.String     `tfsdk:"id"`
	Product types.String     `tfsdk:"product"`
	Records []dnsRecordModel `tfsdk:"records"`
}

func (d *DNSRecordsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_records"
}

func (d *DNSRecordsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists DNS records in a Simply.com product DNS zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source ID.",
			},
			"product": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Simply.com product object, usually the domain name.",
				Validators:          productValidators(),
			},
			"records": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DNS records in the zone.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: dnsRecordNestedAttributes(),
				},
			},
		},
	}
}

func (d *DNSRecordsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DNSRecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dnsRecordsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, err := d.client.ListDNSRecords(ctx, data.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read DNS Records", err.Error())
		return
	}

	data.ID = data.Product
	data.Records = make([]dnsRecordModel, 0, len(records))
	for _, record := range records {
		data.Records = append(data.Records, modelFromRecord(data.Product.ValueString(), record))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func dnsRecordNestedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Record ID in the form `product:record_id`.",
		},
		"product": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Simply.com product object.",
		},
		"record_id": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Simply.com DNS record ID.",
		},
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Record name within the DNS zone.",
		},
		"type": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "DNS record type.",
		},
		"data": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "DNS record data.",
		},
		"ttl": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Record TTL in seconds.",
		},
		"priority": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Priority value.",
		},
		"comment": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Record comment.",
		},
	}
}
