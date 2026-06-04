// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package dnsrecord

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

var _ datasource.DataSource = &DNSRecordDataSource{}

func NewDataSource() datasource.DataSource {
	return &DNSRecordDataSource{}
}

type DNSRecordDataSource struct {
	client *simply.Client
}

type dnsRecordDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Product  types.String `tfsdk:"product"`
	RecordID types.Int64  `tfsdk:"record_id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Data     types.String `tfsdk:"data"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
	Comment  types.String `tfsdk:"comment"`
}

func (d *DNSRecordDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (d *DNSRecordDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a DNS record in a Simply.com product DNS zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source ID in the form `product:record_id`.",
			},
			"product": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Simply.com product object, usually the domain name.",
				Validators:          productValidators(),
			},
			"record_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Simply.com DNS record ID.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Record name within the DNS zone.",
				Validators:          recordNameValidators(),
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DNS record type.",
				Validators:          recordTypeValidators(),
			},
			"data": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "DNS record data.",
				Validators:          recordDataValidators(),
			},
			"ttl": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Record TTL in seconds.",
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Priority value.",
				Validators:          priorityValidators(),
			},
			"comment": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Record comment.",
			},
		},
	}
}

func (d *DNSRecordDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DNSRecordDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dnsRecordDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var matches []simply.DNSRecord
	records, err := d.client.ListDNSRecords(ctx, data.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read DNS Records", err.Error())
		return
	}
	for _, record := range records {
		if recordMatchesDataSource(data, record) {
			matches = append(matches, record)
		}
	}
	if len(matches) == 0 {
		resp.Diagnostics.AddError("DNS Record Not Found", "No DNS record matched the configured lookup attributes.")
		return
	}
	if len(matches) > 1 {
		resp.Diagnostics.AddError("Multiple DNS Records Found", "More than one DNS record matched the configured lookup attributes. Add record_id or more filters.")
		return
	}

	model := modelFromRecord(data.Product.ValueString(), matches[0])
	data.ID = model.ID
	data.RecordID = model.RecordID
	data.Name = model.Name
	data.Type = model.Type
	data.Data = model.Data
	data.TTL = model.TTL
	data.Priority = model.Priority
	data.Comment = model.Comment
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func recordMatchesDataSource(data dnsRecordDataSourceModel, record simply.DNSRecord) bool {
	if !data.RecordID.IsNull() && !data.RecordID.IsUnknown() && data.RecordID.ValueInt64() != record.RecordID {
		return false
	}
	if !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != record.Name {
		return false
	}
	if !data.Type.IsNull() && !data.Type.IsUnknown() && strings.ToUpper(data.Type.ValueString()) != strings.ToUpper(record.Type) {
		return false
	}
	if !data.Data.IsNull() && !data.Data.IsUnknown() && data.Data.ValueString() != record.Data {
		return false
	}
	if !data.Priority.IsNull() && !data.Priority.IsUnknown() {
		if record.Priority == nil || data.Priority.ValueInt64() != *record.Priority {
			return false
		}
	}
	return true
}
