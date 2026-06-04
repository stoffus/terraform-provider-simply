// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package products

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

var _ datasource.DataSource = &DataSource{}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

type DataSource struct {
	client *simply.Client
}

type dataSourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Products []productModel `tfsdk:"products"`
}

type productModel struct {
	Object      types.String   `tfsdk:"object"`
	ObjectURI   types.String   `tfsdk:"object_uri"`
	Name        types.String   `tfsdk:"name"`
	Cancelled   types.Bool     `tfsdk:"cancelled"`
	Domain      domainModel    `tfsdk:"domain"`
	Service     serviceModel   `tfsdk:"service"`
	Nameservers []types.String `tfsdk:"nameservers"`
}

type domainModel struct {
	Name      types.String `tfsdk:"name"`
	NameIDN   types.String `tfsdk:"name_idn"`
	Managed   types.Bool   `tfsdk:"managed"`
	RenewDate types.String `tfsdk:"renew_date"`
}

type serviceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	CreatedDate types.String `tfsdk:"created_date"`
	ExpireDate  types.String `tfsdk:"expire_date"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_products"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Simply.com products available to the configured account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source ID.",
			},
			"products": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Products available to the configured account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"object": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Product identifier to use in other Simply API calls.",
						},
						"object_uri": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Product object URI.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Product name or handle.",
						},
						"cancelled": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the product has been cancelled.",
						},
						"domain": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Domain information for the product.",
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "ASCII domain name.",
								},
								"name_idn": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Internationalized domain name.",
								},
								"managed": schema.BoolAttribute{
									Computed:            true,
									MarkdownDescription: "Whether the domain is registered and managed by Simply.com.",
								},
								"renew_date": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Next renewal date for the domain, if available.",
								},
							},
						},
						"service": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Service information for the product.",
							Attributes: map[string]schema.Attribute{
								"id": schema.Int64Attribute{
									Computed:            true,
									MarkdownDescription: "Service ID.",
								},
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Service name.",
								},
								"created_date": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Product creation date.",
								},
								"expire_date": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Product expiration date, if available.",
								},
							},
						},
						"nameservers": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Simply.com nameservers currently assigned to the domain in Simply.com's system. This does not query the registry.",
						},
					},
				},
			},
		},
	}
}

func (d *DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	products, err := d.client.ListProducts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Products", err.Error())
		return
	}

	data := dataSourceModel{
		ID:       types.StringValue("products"),
		Products: make([]productModel, 0, len(products)),
	}
	for _, product := range products {
		data.Products = append(data.Products, modelFromProduct(product))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func modelFromProduct(product simply.Product) productModel {
	return productModel{
		Object:    types.StringValue(product.Object),
		ObjectURI: types.StringValue(product.ObjectURI),
		Name:      types.StringValue(product.Name),
		Cancelled: types.BoolValue(product.Cancelled),
		Domain: domainModel{
			Name:      types.StringValue(product.Domain.Name),
			NameIDN:   types.StringValue(product.Domain.NameIDN),
			Managed:   types.BoolValue(product.Domain.Managed),
			RenewDate: optionalString(product.Domain.RenewDate),
		},
		Service: serviceModel{
			ID:          types.Int64Value(product.Product.ID),
			Name:        types.StringValue(product.Product.Name),
			CreatedDate: types.StringValue(product.Product.CreatedDate),
			ExpireDate:  optionalString(product.Product.ExpireDate),
		},
		Nameservers: stringValues(product.Servers.Nameservers),
	}
}

func optionalString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func stringValues(values []string) []types.String {
	result := make([]types.String, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}
	return result
}
