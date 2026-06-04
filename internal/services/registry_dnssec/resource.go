// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package registrydnssec

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client *simply.Client
}

type model struct {
	ID      types.String `tfsdk:"id"`
	Product types.String `tfsdk:"product"`
	Keys    []keyModel   `tfsdk:"keys"`
}

type keyModel struct {
	Type types.String `tfsdk:"type"`
	Data types.String `tfsdk:"data"`
}

func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_dnssec_keys"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages all DNSSEC keys registered for a Simply.com domain product.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Terraform resource ID, equal to the product object.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Simply.com product object, usually the domain name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keys": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "DNSSEC keys to publish at the registry. The Simply.com API removes all keys at once, so this resource manages the full set for the product.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "DNSSEC record type. Must be `ds` or `dnskey`.",
							Validators: []validator.String{
								stringvalidator.OneOf("ds", "dnskey"),
							},
						},
						"data": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Zone-file formatted record data. For DS: `keytag algorithm digestType digest`. For DNSKEY: `flags protocol algorithm pubkey`.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.addKeys(ctx, plan.Product.ValueString(), plan.Keys); err != nil {
		resp.Diagnostics.AddError("Unable to Add Registry DNSSEC Keys", err.Error())
		return
	}

	state, err := r.read(ctx, plan.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Registry DNSSEC Keys", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, err := r.read(ctx, state.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Registry DNSSEC Keys", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	product := plan.Product.ValueString()
	if err := r.client.DeleteRegistryDNSSECKeys(ctx, product); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Registry DNSSEC Keys", err.Error())
		return
	}
	if err := r.addKeys(ctx, product, plan.Keys); err != nil {
		resp.Diagnostics.AddError("Unable to Add Registry DNSSEC Keys", err.Error())
		return
	}

	state, err := r.read(ctx, product)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Registry DNSSEC Keys", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRegistryDNSSECKeys(ctx, state.Product.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Registry DNSSEC Keys", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("product"), req.ID)...)
}

func (r *Resource) addKeys(ctx context.Context, product string, keys []keyModel) error {
	for _, key := range keys {
		payload := simply.AddDNSSECKeyPayload{
			Type: key.Type.ValueString(),
			Data: key.Data.ValueString(),
		}
		if err := r.client.AddRegistryDNSSECKey(ctx, product, payload); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resource) read(ctx context.Context, product string) (model, error) {
	keys, err := r.client.ListRegistryDNSSECKeys(ctx, product)
	if err != nil {
		return model{}, err
	}

	return model{
		ID:      types.StringValue(product),
		Product: types.StringValue(product),
		Keys:    keyModelsFromAPI(keys),
	}, nil
}

func keyModelsFromAPI(keys []simply.DNSSECKey) []keyModel {
	result := make([]keyModel, 0, len(keys))
	for _, key := range keys {
		result = append(result, keyModel{
			Type: types.StringValue(key.Type),
			Data: types.StringValue(key.Data),
		})
	}
	return result
}
