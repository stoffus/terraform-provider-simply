// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package dnsrecord

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

type dnsRecordModel struct {
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

func recordID(product string, recordID int64) string {
	return fmt.Sprintf("%s:%d", product, recordID)
}

func parseRecordID(id string) (string, int64, error) {
	product, recordIDText, ok := strings.Cut(id, ":")
	if !ok || product == "" || recordIDText == "" {
		return "", 0, fmt.Errorf("expected import ID in the form product:record_id")
	}
	recordID, err := strconv.ParseInt(recordIDText, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid record_id %q: %w", recordIDText, err)
	}
	return product, recordID, nil
}

func payloadFromModel(model dnsRecordModel) simply.DNSRecordPayload {
	payload := simply.DNSRecordPayload{
		Name: model.Name.ValueString(),
		Type: strings.ToUpper(model.Type.ValueString()),
		Data: model.Data.ValueString(),
		TTL:  model.TTL.ValueInt64(),
	}
	if !model.Priority.IsNull() && !model.Priority.IsUnknown() {
		value := model.Priority.ValueInt64()
		payload.Priority = &value
	}
	if !model.Comment.IsNull() && !model.Comment.IsUnknown() {
		value := model.Comment.ValueString()
		payload.Comment = &value
	}
	return payload
}

func modelFromRecord(product string, record simply.DNSRecord) dnsRecordModel {
	model := dnsRecordModel{
		ID:       types.StringValue(recordID(product, record.RecordID)),
		Product:  types.StringValue(product),
		RecordID: types.Int64Value(record.RecordID),
		Name:     types.StringValue(record.Name),
		Type:     types.StringValue(record.Type),
		Data:     types.StringValue(record.Data),
		TTL:      types.Int64Value(record.TTL),
		Priority: types.Int64Null(),
		Comment:  types.StringNull(),
	}
	if record.Priority != nil {
		model.Priority = types.Int64Value(*record.Priority)
	}
	if record.Comment != nil {
		model.Comment = types.StringValue(*record.Comment)
	}
	return model
}
