// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package registrynameservers

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
	ID          types.String   `tfsdk:"id"`
	Product     types.String   `tfsdk:"product"`
	Nameservers []types.String `tfsdk:"nameservers"`
}

func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_nameservers"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages registry nameservers for a Simply.com domain product. Deleting this resource removes it from Terraform state only because Simply.com does not expose an API operation to remove registry nameservers.",
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
			"nameservers": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Authoritative nameserver hostnames to set at the registry. Simply.com requires 2 to 13 nameservers.",
				Validators: []validator.List{
					listvalidator.SizeBetween(2, 13),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
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

	if err := r.client.SetRegistryNameservers(ctx, plan.Product.ValueString(), stringsFromValues(plan.Nameservers)); err != nil {
		resp.Diagnostics.AddError("Unable to Set Registry Nameservers", err.Error())
		return
	}

	state, err := r.read(ctx, plan.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Registry Nameservers", err.Error())
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
		resp.Diagnostics.AddError("Unable to Read Registry Nameservers", err.Error())
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

	if err := r.client.SetRegistryNameservers(ctx, plan.Product.ValueString(), stringsFromValues(plan.Nameservers)); err != nil {
		resp.Diagnostics.AddError("Unable to Set Registry Nameservers", err.Error())
		return
	}

	state, err := r.read(ctx, plan.Product.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Registry Nameservers", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Registry Nameservers Removed From State Only",
		"Simply.com does not expose an API operation to remove registry nameservers. Terraform will forget this resource without changing the registry nameserver configuration.",
	)
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("product"), req.ID)...)
}

func (r *Resource) read(ctx context.Context, product string) (model, error) {
	nameservers, err := r.client.GetRegistryNameservers(ctx, product)
	if err != nil {
		return model{}, err
	}
	return model{
		ID:          types.StringValue(product),
		Product:     types.StringValue(product),
		Nameservers: valuesFromStrings(nameservers),
	}, nil
}

func stringsFromValues(values []types.String) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !value.IsNull() && !value.IsUnknown() {
			result = append(result, value.ValueString())
		}
	}
	return result
}

func valuesFromStrings(values []string) []types.String {
	result := make([]types.String, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}
	return result
}
