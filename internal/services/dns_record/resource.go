// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package dnsrecord

import (
	"context"
	"fmt"

	"github.com/christopher/terraform-provider-simply/internal/simply"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DNSRecordResource{}
var _ resource.ResourceWithImportState = &DNSRecordResource{}

func NewResource() resource.Resource {
	return &DNSRecordResource{}
}

type DNSRecordResource struct {
	client *simply.Client
}

func (r *DNSRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *DNSRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS record in a Simply.com product DNS zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Terraform resource ID in the form `product:record_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Simply.com product object, usually the domain name.",
				Validators:          productValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"record_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Simply.com DNS record ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Record name within the DNS zone.",
				Validators:          recordNameValidators(),
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "DNS record type.",
				Validators:          recordTypeValidators(),
			},
			"data": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "DNS record data.",
				Validators:          recordDataValidators(),
			},
			"ttl": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
				MarkdownDescription: "Record TTL in seconds.",
				Validators:          ttlValidators(),
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Priority value for record types such as MX and SRV.",
				Validators:          priorityValidators(),
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional record comment.",
			},
		},
	}
}

func (r *DNSRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*simply.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *simply.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *DNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recordIDValue, err := r.client.CreateDNSRecord(ctx, plan.Product.ValueString(), payloadFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create DNS Record", err.Error())
		return
	}

	record, found, err := r.client.GetDNSRecord(ctx, plan.Product.ValueString(), recordIDValue)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read DNS Record", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to Read DNS Record", "The record was created but could not be found in the DNS zone.")
		return
	}

	state := modelFromRecord(plan.Product.ValueString(), record)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, found, err := r.client.GetDNSRecord(ctx, state.Product.ValueString(), state.RecordID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read DNS Record", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	nextState := modelFromRecord(state.Product.ValueString(), record)
	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *DNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateDNSRecord(ctx, plan.Product.ValueString(), plan.RecordID.ValueInt64(), payloadFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update DNS Record", err.Error())
		return
	}

	record, found, err := r.client.GetDNSRecord(ctx, plan.Product.ValueString(), plan.RecordID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read DNS Record", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to Read DNS Record", "The record was updated but could not be found in the DNS zone.")
		return
	}

	state := modelFromRecord(plan.Product.ValueString(), record)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDNSRecord(ctx, state.Product.ValueString(), state.RecordID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete DNS Record", err.Error())
	}
}

func (r *DNSRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	product, recordIDValue, err := parseRecordID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("product"), product)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("record_id"), types.Int64Value(recordIDValue))...)
}
