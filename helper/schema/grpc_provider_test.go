// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/internal/plugin/convert"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestGRPCProviderServerConfigureProvider(t *testing.T) {
	t.Parallel()

	type FakeMetaStruct struct {
		Attr string
	}

	testCases := map[string]struct {
		server                   *GRPCProviderServer
		req                      *tfprotov5.ConfigureProviderRequest
		expected                 *tfprotov5.ConfigureProviderResponse
		expectedProviderDeferred *Deferred
		expectedMeta             any
	}{
		"no-Configure-or-Schema": {
			server: NewGRPCProviderServer(&Provider{
				// ConfigureFunc, ConfigureContextFunc, ConfigureProvider, and Schema intentionally
				// omitted.
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.EmptyObject,
						cty.EmptyObjectVal,
						// cty.Object(map[string]cty.Type{
						// 	"id": cty.String,
						// }),
						// cty.NullVal(cty.Object(map[string]cty.Type{
						// 	"id": cty.String,
						// })),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"Schema-no-Configure": {
			server: NewGRPCProviderServer(&Provider{
				// ConfigureFunc, ConfigureContextFunc, and ConfigureProvider intentionally omitted.
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-error": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureFunc: func(d *ResourceData) (any, error) {
					return nil, fmt.Errorf("test error")
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "test error",
						Detail:   "",
					},
				},
			},
		},
		"ConfigureContextFunc-error": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					return nil, diag.Errorf("test error")
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "test error",
						Detail:   "",
					},
				},
			},
		},
		"ConfigureProvider-error": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					resp.Diagnostics = diag.Errorf("test error")
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "test error",
						Detail:   "",
					},
				},
			},
		},
		"ConfigureContextFunc-warning": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					return nil, diag.Diagnostics{
						{
							Severity: diag.Warning,
							Summary:  "test warning summary",
							Detail:   "test warning detail",
						},
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityWarning,
						Summary:  "test warning summary",
						Detail:   "test warning detail",
					},
				},
			},
		},
		"ConfigureProvider-warning": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					resp.Diagnostics = diag.Diagnostics{
						{
							Severity: diag.Warning,
							Summary:  "test warning summary",
							Detail:   "test warning detail",
						},
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityWarning,
						Summary:  "test warning summary",
						Detail:   "test warning detail",
					},
				},
			},
		},
		"ConfigureFunc-Get-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, fmt.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-Get-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-Get-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.Get("test").(string)
					expected := ""

					if got != expected {
						resp.Diagnostics = diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-Get-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, fmt.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-Get-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-Get-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.Get("test").(string)
					expected := ""

					if got != expected {
						resp.Diagnostics = diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-Get-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, fmt.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-Get-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-Get-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.Get("test").(string)
					expected := ""

					if got != expected {
						resp.Diagnostics = diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-Get-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, fmt.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-Get-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.Get("test").(string)
					expected := ""

					if got != expected {
						return nil, diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-Get-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.Get("test").(string)
					expected := ""

					if got != expected {
						resp.Diagnostics = diag.Errorf("unexpected Get difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-Get-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureFunc: func(d *ResourceData) (any, error) {
					got := d.Get("test").(string)
					expected := "test-value"

					if got != expected {
						return nil, fmt.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-Get-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					got := d.Get("test").(string)
					expected := "test-value"

					if got != expected {
						return nil, diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-Get-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.Get("test").(string)
					expected := "test-value"

					if got != expected {
						resp.Diagnostics = diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-Get-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureFunc: func(d *ResourceData) (any, error) {
					got := d.Get("test").(string)
					expected := "test-value"

					if got != expected {
						return nil, fmt.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"other": {
						Optional: true,
						Type:     TypeString,
					},
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-Get-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					got := d.Get("test").(string)
					expected := "test-value"

					if got != expected {
						return nil, diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"other": {
						Optional: true,
						Type:     TypeString,
					},
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-Get-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.Get("test").(string)
					expected := "test-value"

					if got != expected {
						resp.Diagnostics = diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}
				},
				Schema: map[string]*Schema{
					"other": {
						Optional: true,
						Type:     TypeString,
					},
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOk-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOk-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOk-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOk-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOk-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOk-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOk-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOk-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOk-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOk-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOk-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOk-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOk("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOk-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOk("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOk-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOk("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOk-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOk("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOk-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOk("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOk-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOk("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOk-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOk("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOk difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOkExists-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOkExists-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOkExists-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOkExists("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOkExists-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOkExists-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOkExists-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOkExists("test")
					expected := ""
					expectedOk := false

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOkExists-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := true

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOkExists-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := true

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOkExists-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOkExists("test")
					expected := ""
					expectedOk := true

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOkExists-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := true

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOkExists-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOkExists("test")
					expected := ""
					expectedOk := true

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOkExists-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOkExists("test")
					expected := ""
					expectedOk := true

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOkExists-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOkExists("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOkExists-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOkExists("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOkExists-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOkExists("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetOkExists-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got, ok := d.GetOkExists("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, fmt.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetOkExists-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got, ok := d.GetOkExists("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
					}

					if got.(string) != expected {
						return nil, diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetOkExists-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got, ok := req.ResourceData.GetOkExists("test")
					expected := "test-value"
					expectedOk := true

					if ok != expectedOk {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %t, got: %t", expectedOk, ok)
						return
					}

					if got.(string) != expected {
						resp.Diagnostics = diag.Errorf("unexpected GetOkExists difference: expected: %s, got: %s", expected, got)
						return
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetRawConfig-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.NullVal(cty.String),
					})

					if got.Equals(expected).False() {
						return nil, fmt.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetRawConfig-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.NullVal(cty.String),
					})

					if got.Equals(expected).False() {
						return nil, diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetRawConfig-null": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.NullVal(cty.String),
					})

					if got.Equals(expected).False() {
						resp.Diagnostics = diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetRawConfig-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.NullVal(cty.String),
					})

					if got.Equals(expected).False() {
						return nil, fmt.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetRawConfig-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.NullVal(cty.String),
					})

					if got.Equals(expected).False() {
						return nil, diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetRawConfig-null-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.NullVal(cty.String),
					})

					if got.Equals(expected).False() {
						resp.Diagnostics = diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetRawConfig-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.StringVal(""),
					})

					if got.Equals(expected).False() {
						return nil, fmt.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetRawConfig-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.StringVal(""),
					})

					if got.Equals(expected).False() {
						return nil, diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetRawConfig-zero-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.StringVal(""),
					})

					if got.Equals(expected).False() {
						resp.Diagnostics = diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetRawConfig-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.StringVal(""),
					})

					if got.Equals(expected).False() {
						return nil, fmt.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetRawConfig-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (interface{}, diag.Diagnostics) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.StringVal(""),
					})

					if got.Equals(expected).False() {
						return nil, diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetRawConfig-zero-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				Schema: map[string]*Schema{
					"other": {
						Type:     TypeString,
						Optional: true,
					},
					"test": {
						Type:     TypeString,
						Optional: true,
					},
				},
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.StringVal(""),
					})

					if got.Equals(expected).False() {
						resp.Diagnostics = diag.Errorf("unexpected GetRawConfig difference: expected: %s, got: %s", expected, got)
					}
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal(""),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetRawConfig-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureFunc: func(d *ResourceData) (any, error) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.StringVal("test-value"),
					})

					if got.Equals(expected).False() {
						return nil, fmt.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetRawConfig-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.StringVal("test-value"),
					})

					if got.Equals(expected).False() {
						return nil, diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetRawConfig-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"test": cty.StringVal("test-value"),
					})

					if got.Equals(expected).False() {
						resp.Diagnostics = diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-GetRawConfig-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureFunc: func(d *ResourceData) (any, error) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.StringVal("test-value"),
					})

					if got.Equals(expected).False() {
						return nil, fmt.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"other": {
						Optional: true,
						Type:     TypeString,
					},
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureContextFunc-GetRawConfig-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					got := d.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.StringVal("test-value"),
					})

					if got.Equals(expected).False() {
						return nil, diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}

					return nil, nil
				},
				Schema: map[string]*Schema{
					"other": {
						Optional: true,
						Type:     TypeString,
					},
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureProvider-GetRawConfig-value-other-value": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					got := req.ResourceData.GetRawConfig()
					expected := cty.ObjectVal(map[string]cty.Value{
						"other": cty.StringVal("other-value"),
						"test":  cty.StringVal("test-value"),
					})

					if got.Equals(expected).False() {
						resp.Diagnostics = diag.Errorf("unexpected difference: expected: %s, got: %s", expected, got)
					}
				},
				Schema: map[string]*Schema{
					"other": {
						Optional: true,
						Type:     TypeString,
					},
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"other": cty.String,
							"test":  cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"other": cty.StringVal("other-value"),
							"test":  cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
		},
		"ConfigureFunc-Meta": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureFunc: func(d *ResourceData) (any, error) {
					return &FakeMetaStruct{
						Attr: "hello world!",
					}, nil
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
			expectedMeta: &FakeMetaStruct{
				Attr: "hello world!",
			},
		},
		"ConfigureContextFunc-Meta": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureContextFunc: func(ctx context.Context, d *ResourceData) (any, diag.Diagnostics) {
					return &FakeMetaStruct{
						Attr: "hello world!",
					}, nil
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
			expectedMeta: &FakeMetaStruct{
				Attr: "hello world!",
			},
		},
		"ConfigureProvider-Meta": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					resp.Meta = &FakeMetaStruct{
						Attr: "hello world!",
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
			expectedMeta: &FakeMetaStruct{
				Attr: "hello world!",
			},
		},
		"ConfigureProvider-Deferred-Allowed": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					resp.Deferred = &Deferred{
						Reason: DeferredReasonProviderConfigUnknown,
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				ClientCapabilities: &tfprotov5.ConfigureProviderClientCapabilities{
					DeferralAllowed: true,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{},
			expectedProviderDeferred: &Deferred{
				Reason: DeferredReasonProviderConfigUnknown,
			},
		},
		"ConfigureProvider-Deferred-ClientCapabilities-Unset-Diagnostic": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					resp.Deferred = &Deferred{
						Reason: DeferredReasonProviderConfigUnknown,
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				// No ClientCapabilities set, deferred response will cause a diagnostic to be returned
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Invalid Deferred Provider Response",
						Detail: "Provider configured a deferred response for all resources and data sources but the Terraform request " +
							"did not indicate support for deferred actions. This is an issue with the provider and should be reported to the provider developers.",
					},
				},
			},
		},
		"ConfigureProvider-Deferred-Not-Allowed-Diagnostic": {
			server: NewGRPCProviderServer(&Provider{
				ConfigureProvider: func(ctx context.Context, req ConfigureProviderRequest, resp *ConfigureProviderResponse) {
					resp.Deferred = &Deferred{
						Reason: DeferredReasonProviderConfigUnknown,
					}
				},
				Schema: map[string]*Schema{
					"test": {
						Optional: true,
						Type:     TypeString,
					},
				},
			}),
			req: &tfprotov5.ConfigureProviderRequest{
				ClientCapabilities: &tfprotov5.ConfigureProviderClientCapabilities{
					// Deferred response will cause a diagnostic to be returned
					DeferralAllowed: false,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"test": cty.StringVal("test-value"),
						}),
					),
				},
			},
			expected: &tfprotov5.ConfigureProviderResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Invalid Deferred Provider Response",
						Detail: "Provider configured a deferred response for all resources and data sources but the Terraform request " +
							"did not indicate support for deferred actions. This is an issue with the provider and should be reported to the provider developers.",
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp, err := testCase.server.ConfigureProvider(context.Background(), testCase.req)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, testCase.expected); diff != "" {
				t.Fatalf("unexpected difference: %s", diff)
			}

			if diff := cmp.Diff(testCase.server.provider.Meta(), testCase.expectedMeta); diff != "" {
				t.Fatalf("unexpected difference: %s", diff)
			}

			if len(resp.Diagnostics) == 0 {
				if diff := cmp.Diff(testCase.server.provider.providerDeferred, testCase.expectedProviderDeferred); diff != "" {
					t.Fatalf("unexpected difference: %s", diff)
				}
			}
		})
	}
}

func TestGRPCProviderServerGetResourceIdentitySchemas(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Provider *Provider
		Expected *tfprotov5.GetResourceIdentitySchemasResponse
	}{
		"resources": {
			Provider: &Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource1": {
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"test": {
										Type:              TypeString,
										RequiredForImport: true,
										OptionalForImport: false,
										Description:       "test resource",
									},
								}
							},
						},
					},
					"test_resource2": {
						Identity: &ResourceIdentity{
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"test2": {
										Type:              TypeString,
										RequiredForImport: false,
										OptionalForImport: true,
										Description:       "test resource 2",
									},
									"test2-2": {
										Type:              TypeList,
										RequiredForImport: false,
										OptionalForImport: true,
										Description:       "test resource 2-2",
									},
									"test2-3": {
										Type:              TypeInt,
										RequiredForImport: false,
										OptionalForImport: true,
										Description:       "test resource 2-3",
									},
								}
							},
						},
					},
				},
			},
			Expected: &tfprotov5.GetResourceIdentitySchemasResponse{
				IdentitySchemas: map[string]*tfprotov5.ResourceIdentitySchema{
					"test_resource1": {
						Version: 1,
						IdentityAttributes: []*tfprotov5.ResourceIdentitySchemaAttribute{
							{
								Name:              "test",
								Type:              tftypes.String,
								RequiredForImport: true,
								OptionalForImport: false,
								Description:       "test resource",
							},
						},
					},
					"test_resource2": {
						IdentityAttributes: []*tfprotov5.ResourceIdentitySchemaAttribute{
							{
								Name:              "test2",
								Type:              tftypes.String,
								RequiredForImport: false,
								OptionalForImport: true,
								Description:       "test resource 2",
							},
							{
								Name:              "test2-2",
								Type:              tftypes.List{ElementType: tftypes.String},
								RequiredForImport: false,
								OptionalForImport: true,
								Description:       "test resource 2-2",
							},
							{
								Name:              "test2-3",
								Type:              tftypes.Number,
								RequiredForImport: false,
								OptionalForImport: true,
								Description:       "test resource 2-3",
							},
						},
					},
				},
			},
		},
		"primitive attributes": {
			Provider: &Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						Identity: &ResourceIdentity{
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"bool_attr":       {Type: TypeBool, Description: "Boolean attribute"},
									"float_attr":      {Type: TypeFloat, Description: "Float attribute"},
									"int_attr":        {Type: TypeInt, Description: "Int attribute"},
									"list_bool_attr":  {Type: TypeList, Elem: TypeBool, Description: "List Bool attribute"},
									"list_float_attr": {Type: TypeList, Elem: TypeFloat, Description: "List Float attribute"},
									"list_int_attr":   {Type: TypeList, Elem: TypeInt, Description: "List Int attribute"},
									"list_str_attr":   {Type: TypeList, Elem: TypeString, Description: "List String attribute"},
									"string_attr":     {Type: TypeString, Description: "String attribute"},
								}
							},
						},
					},
				},
			},
			Expected: &tfprotov5.GetResourceIdentitySchemasResponse{
				IdentitySchemas: map[string]*tfprotov5.ResourceIdentitySchema{
					"test_resource": {
						IdentityAttributes: []*tfprotov5.ResourceIdentitySchemaAttribute{
							{Name: "bool_attr", Type: tftypes.Bool, Description: "Boolean attribute"},
							{Name: "float_attr", Type: tftypes.Number, Description: "Float attribute"},
							{Name: "int_attr", Type: tftypes.Number, Description: "Int attribute"},
							{Name: "list_bool_attr", Type: tftypes.List{ElementType: tftypes.Bool}, Description: "List Bool attribute"},
							{Name: "list_float_attr", Type: tftypes.List{ElementType: tftypes.Number}, Description: "List Float attribute"},
							{Name: "list_int_attr", Type: tftypes.List{ElementType: tftypes.Number}, Description: "List Int attribute"},
							{Name: "list_str_attr", Type: tftypes.List{ElementType: tftypes.String}, Description: "List String attribute"},
							{Name: "string_attr", Type: tftypes.String, Description: "String attribute"},
						},
					},
				},
			},
		},
		"no identity schema": {
			Provider: &Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource1": {
						Identity: &ResourceIdentity{
							Version: 1,
						},
					},
				},
			},
			Expected: &tfprotov5.GetResourceIdentitySchemasResponse{
				IdentitySchemas: map[string]*tfprotov5.ResourceIdentitySchema{},
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test_resource1': resource does not have an identity schema",
					},
				},
			},
		},
		"empty identity schema": {
			Provider: &Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource1": {
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{}
							},
						},
					},
				},
			},
			Expected: &tfprotov5.GetResourceIdentitySchemasResponse{
				IdentitySchemas: map[string]*tfprotov5.ResourceIdentitySchema{},
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test_resource1': identity schema must have at least one attribute",
					},
				},
			},
		},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := NewGRPCProviderServer(testCase.Provider)

			testReq := &tfprotov5.GetResourceIdentitySchemasRequest{}

			resp, err := server.GetResourceIdentitySchemas(context.Background(), testReq)

			if err != nil {
				t.Fatalf("unexpected gRPC error: %s", err)
			}

			// Prevent false positives with random map access in testing
			for _, schema := range resp.IdentitySchemas {
				sort.Slice(schema.IdentityAttributes, func(i int, j int) bool {
					return schema.IdentityAttributes[i].Name < schema.IdentityAttributes[j].Name
				})
			}

			if diff := cmp.Diff(resp, testCase.Expected); diff != "" {
				t.Errorf("unexpected response difference: %s", diff)
			}
		})
	}
}

// Based on TestUpgradeState_jsonState
func TestUpgradeResourceIdentity_jsonState(t *testing.T) {
	r := &Resource{
		SchemaVersion: 1,
		Identity: &ResourceIdentity{
			Version: 1,
			SchemaFunc: func() map[string]*Schema {
				return map[string]*Schema{
					"id": {
						Type:              TypeString,
						RequiredForImport: true,
						OptionalForImport: false,
						Description:       "id of thing",
					},
				}
			},
			IdentityUpgraders: []IdentityUpgrader{
				{
					Version: 0,
					Type: tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"identity": tftypes.String,
						},
					},
					// upgrades former identity using "identity" as the attribute name to the new and shiny one just using "id"
					Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
						id, ok := rawState["identity"].(string)
						if !ok {
							return nil, fmt.Errorf("identity not found in %#v", rawState)
						}
						rawState["id"] = id
						delete(rawState, "identity")
						return rawState, nil
					},
				},
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	req := &tfprotov5.UpgradeResourceIdentityRequest{
		TypeName: "test",
		Version:  0,
		RawIdentity: &tfprotov5.RawState{
			JSON: []byte(`{"identity":"Peter"}`),
		},
	}

	resp, err := server.UpgradeResourceIdentity(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	idschema, err := r.CoreIdentitySchema()

	if err != nil {
		t.Fatal(err)
	}

	val, err := msgpack.Unmarshal(resp.UpgradedIdentity.IdentityData.MsgPack, idschema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("Peter"),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

// Based on TestUpgradeState_removedAttr
func TestUpgradeResourceIdentity_removedAttr(t *testing.T) {
	r := &Resource{
		SchemaVersion: 1,
		Identity: &ResourceIdentity{
			Version: 1,
			SchemaFunc: func() map[string]*Schema {
				return map[string]*Schema{
					"id": {
						Type:              TypeString,
						RequiredForImport: true,
						OptionalForImport: false,
						Description:       "id of thing",
					},
				}
			},
			IdentityUpgraders: []IdentityUpgrader{
				{
					Version: 0,
					Type: tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"identity": tftypes.String,
							"removed":  tftypes.String,
						},
					},
					Upgrade: func(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
						id, ok := rawState["identity"].(string)
						if !ok {
							return nil, fmt.Errorf("identity not found in %#v", rawState)
						}
						rawState["id"] = id
						delete(rawState, "identity")
						delete(rawState, "removed")
						return rawState, nil
					},
				},
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	req := &tfprotov5.UpgradeResourceIdentityRequest{
		TypeName: "test",
		Version:  0,
		RawIdentity: &tfprotov5.RawState{
			JSON: []byte(`{"identity":"Peter", "removed":"to_be_removed"}`),
		},
	}

	resp, err := server.UpgradeResourceIdentity(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	idschema, err := r.CoreIdentitySchema()
	if err != nil {
		t.Fatal(err)
	}

	val, err := msgpack.Unmarshal(resp.UpgradedIdentity.IdentityData.MsgPack, idschema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("Peter"),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

// Based on TestUpgradeState_jsonStateBigInt
// This test currently does not return the integer and does not recognize it as an attribute
func TestUpgradeResourceIdentity_jsonStateBigInt(t *testing.T) {
	r := &Resource{
		UseJSONNumber: true,
		SchemaVersion: 1,
		Identity: &ResourceIdentity{
			Version: 1,
			SchemaFunc: func() map[string]*Schema {
				return map[string]*Schema{
					"int": {
						Type:              TypeInt,
						RequiredForImport: true,
						OptionalForImport: false,
						Description:       "",
					},
				}
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	req := &tfprotov5.UpgradeResourceIdentityRequest{
		TypeName: "test",
		Version:  0,
		RawIdentity: &tfprotov5.RawState{
			JSON: []byte(`{"int":7227701560655103598}`),
		},
	}

	resp, err := server.UpgradeResourceIdentity(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	idschema, err := r.CoreIdentitySchema()
	if err != nil {
		t.Fatal(err)
	}

	val, err := msgpack.Unmarshal(resp.UpgradedIdentity.IdentityData.MsgPack, idschema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"int": cty.NumberIntVal(7227701560655103598),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

func TestGRPCProviderServerGetMetadata(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Provider *Provider
		Expected *tfprotov5.GetMetadataResponse
	}{
		"datasources": {
			Provider: &Provider{
				DataSourcesMap: map[string]*Resource{
					"test_datasource1": nil, // implementation not necessary
					"test_datasource2": nil, // implementation not necessary
				},
			},
			Expected: &tfprotov5.GetMetadataResponse{
				DataSources: []tfprotov5.DataSourceMetadata{
					{
						TypeName: "test_datasource1",
					},
					{
						TypeName: "test_datasource2",
					},
				},
				Functions:          []tfprotov5.FunctionMetadata{},
				EphemeralResources: []tfprotov5.EphemeralResourceMetadata{},
				Resources:          []tfprotov5.ResourceMetadata{},
				ServerCapabilities: &tfprotov5.ServerCapabilities{
					GetProviderSchemaOptional: true,
				},
			},
		},
		"datasources and resources": {
			Provider: &Provider{
				DataSourcesMap: map[string]*Resource{
					"test_datasource1": nil, // implementation not necessary
					"test_datasource2": nil, // implementation not necessary
				},
				ResourcesMap: map[string]*Resource{
					"test_resource1": nil, // implementation not necessary
					"test_resource2": nil, // implementation not necessary
				},
			},
			Expected: &tfprotov5.GetMetadataResponse{
				DataSources: []tfprotov5.DataSourceMetadata{
					{
						TypeName: "test_datasource1",
					},
					{
						TypeName: "test_datasource2",
					},
				},
				Functions:          []tfprotov5.FunctionMetadata{},
				EphemeralResources: []tfprotov5.EphemeralResourceMetadata{},
				Resources: []tfprotov5.ResourceMetadata{
					{
						TypeName: "test_resource1",
					},
					{
						TypeName: "test_resource2",
					},
				},
				ServerCapabilities: &tfprotov5.ServerCapabilities{
					GetProviderSchemaOptional: true,
				},
			},
		},
		"resources": {
			Provider: &Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource1": nil, // implementation not necessary
					"test_resource2": nil, // implementation not necessary
				},
			},
			Expected: &tfprotov5.GetMetadataResponse{
				DataSources:        []tfprotov5.DataSourceMetadata{},
				Functions:          []tfprotov5.FunctionMetadata{},
				EphemeralResources: []tfprotov5.EphemeralResourceMetadata{},
				Resources: []tfprotov5.ResourceMetadata{
					{
						TypeName: "test_resource1",
					},
					{
						TypeName: "test_resource2",
					},
				},
				ServerCapabilities: &tfprotov5.ServerCapabilities{
					GetProviderSchemaOptional: true,
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := NewGRPCProviderServer(testCase.Provider)

			testReq := &tfprotov5.GetMetadataRequest{}

			resp, err := server.GetMetadata(context.Background(), testReq)

			if err != nil {
				t.Fatalf("unexpected gRPC error: %s", err)
			}

			// Prevent false positives with random map access in testing
			sort.Slice(resp.DataSources, func(i int, j int) bool {
				return resp.DataSources[i].TypeName < resp.DataSources[j].TypeName
			})

			sort.Slice(resp.Resources, func(i int, j int) bool {
				return resp.Resources[i].TypeName < resp.Resources[j].TypeName
			})

			if diff := cmp.Diff(resp, testCase.Expected); diff != "" {
				t.Errorf("unexpected response difference: %s", diff)
			}
		})
	}
}

func TestGRPCProviderServerMoveResourceState(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		request  *tfprotov5.MoveResourceStateRequest
		expected *tfprotov5.MoveResourceStateResponse
	}{
		"nil": {
			server:   NewGRPCProviderServer(&Provider{}),
			request:  nil,
			expected: nil,
		},
		"request-TargetTypeName-missing": {
			server: NewGRPCProviderServer(&Provider{}),
			request: &tfprotov5.MoveResourceStateRequest{
				TargetTypeName: "test_resource",
			},
			expected: &tfprotov5.MoveResourceStateResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Unknown Resource Type",
						Detail:   "The \"test_resource\" resource type is not supported by this provider.",
					},
				},
			},
		},
		"request-TargetTypeName": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {},
				},
			}),
			request: &tfprotov5.MoveResourceStateRequest{
				TargetTypeName: "test_resource",
			},
			expected: &tfprotov5.MoveResourceStateResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Move Resource State Not Supported",
						Detail:   "The \"test_resource\" resource type does not support moving resource state across resource types.",
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp, err := testCase.server.MoveResourceState(context.Background(), testCase.request)

			if testCase.request != nil && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(resp, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestGRPCProviderServerValidateResourceTypeConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		request  *tfprotov5.ValidateResourceTypeConfigRequest
		expected *tfprotov5.ValidateResourceTypeConfigResponse
	}{
		"Provider with empty resource returns no errors": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{},
		},
		"Client without WriteOnlyAttributesAllowed capabilities: null WriteOnly attribute returns no errors": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeInt,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{},
		},
		"Server without WriteOnlyAttributesAllowed capabilities: WriteOnly Attribute with Value returns an error": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeInt,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NumberIntVal(2),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Write-only Attribute Not Allowed",
						Detail: "The resource contains a non-null value for write-only attribute \"foo\" " +
							"Write-only attributes are only supported in Terraform 1.11 and later.",
						Attribute: tftypes.NewAttributePath().WithAttributeName("foo"),
					},
				},
			},
		},
		"Server without WriteOnlyAttributesAllowed capabilities: multiple WriteOnly Attributes with Value returns multiple errors": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeInt,
								Optional:  true,
								WriteOnly: true,
							},
							"bar": {
								Type:      TypeInt,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
							"bar": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NumberIntVal(2),
							"bar": cty.NumberIntVal(2),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Write-only Attribute Not Allowed",
						Detail: "The resource contains a non-null value for write-only attribute \"bar\" " +
							"Write-only attributes are only supported in Terraform 1.11 and later.",
						Attribute: tftypes.NewAttributePath().WithAttributeName("bar"),
					},
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Write-only Attribute Not Allowed",
						Detail: "The resource contains a non-null value for write-only attribute \"foo\" " +
							"Write-only attributes are only supported in Terraform 1.11 and later.",
						Attribute: tftypes.NewAttributePath().WithAttributeName("foo"),
					},
				},
			},
		},
		"Server without WriteOnlyAttributesAllowed capabilities: multiple nested WriteOnly Attributes with Value returns multiple errors": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeInt,
								Optional:  true,
								WriteOnly: true,
							},
							"bar": {
								Type:     TypeInt,
								Optional: true,
							},
							"config_block_attr": {
								Type:      TypeList,
								Optional:  true,
								WriteOnly: true,
								Elem: &Resource{
									Schema: map[string]*Schema{
										"nested_attr": {
											Type:     TypeString,
											Optional: true,
										},
										"writeonly_nested_attr": {
											Type:      TypeString,
											WriteOnly: true,
											Optional:  true,
										},
									},
								},
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
							"bar": cty.Number,
							"config_block_attr": cty.List(cty.Object(map[string]cty.Type{
								"nested_attr":           cty.String,
								"writeonly_nested_attr": cty.String,
							})),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NumberIntVal(2),
							"bar": cty.NumberIntVal(2),
							"config_block_attr": cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									"nested_attr":           cty.StringVal("value"),
									"writeonly_nested_attr": cty.StringVal("value"),
								}),
							}),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Write-only Attribute Not Allowed",
						Detail: "The resource contains a non-null value for write-only attribute \"foo\" " +
							"Write-only attributes are only supported in Terraform 1.11 and later.",
						Attribute: tftypes.NewAttributePath().WithAttributeName("foo"),
					},
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Write-only Attribute Not Allowed",
						Detail: "The resource contains a non-null value for write-only attribute \"writeonly_nested_attr\" " +
							"Write-only attributes are only supported in Terraform 1.11 and later.",
						Attribute: tftypes.NewAttributePath().
							WithAttributeName("config_block_attr").
							WithElementKeyInt(0).
							WithAttributeName("writeonly_nested_attr"),
					},
				},
			},
		},
		"Server with ValidateRawResourceConfigFunc: WriteOnlyAttributesAllowed true returns diags": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						ValidateRawResourceConfigFuncs: []ValidateRawResourceConfigFunc{
							func(ctx context.Context, req ValidateResourceConfigFuncRequest, resp *ValidateResourceConfigFuncResponse) {
								if req.WriteOnlyAttributesAllowed {
									resp.Diagnostics = diag.Diagnostics{
										{
											Severity: diag.Error,
											Summary:  "ValidateRawResourceConfigFunc Error",
										},
									}
								}
							},
							func(ctx context.Context, req ValidateResourceConfigFuncRequest, resp *ValidateResourceConfigFuncResponse) {
								if req.WriteOnlyAttributesAllowed {
									resp.Diagnostics = diag.Diagnostics{
										{
											Severity: diag.Error,
											Summary:  "ValidateRawResourceConfigFunc Error",
										},
									}
								}
							},
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeInt,
								Optional:  true,
								WriteOnly: true,
							},
							"bar": {
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				ClientCapabilities: &tfprotov5.ValidateResourceTypeConfigClientCapabilities{
					WriteOnlyAttributesAllowed: true,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
							"bar": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NumberIntVal(2),
							"bar": cty.NumberIntVal(2),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "ValidateRawResourceConfigFunc Error",
					},
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "ValidateRawResourceConfigFunc Error",
					},
				},
			},
		},
		"Server with ValidateRawResourceConfigFunc: WriteOnlyAttributesAllowed false returns diags": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						ValidateRawResourceConfigFuncs: []ValidateRawResourceConfigFunc{
							func(ctx context.Context, req ValidateResourceConfigFuncRequest, resp *ValidateResourceConfigFuncResponse) {
								if !req.WriteOnlyAttributesAllowed {
									resp.Diagnostics = diag.Diagnostics{
										{
											Severity: diag.Error,
											Summary:  "ValidateRawResourceConfigFunc Error",
										},
									}
								}
							},
							func(ctx context.Context, req ValidateResourceConfigFuncRequest, resp *ValidateResourceConfigFuncResponse) {
								if !req.WriteOnlyAttributesAllowed {
									resp.Diagnostics = diag.Diagnostics{
										{
											Severity: diag.Error,
											Summary:  "ValidateRawResourceConfigFunc Error",
										},
									}
								}
							},
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
							"bar": {
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
							"bar": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NumberIntVal(2),
							"bar": cty.NumberIntVal(2),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "ValidateRawResourceConfigFunc Error",
					},
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "ValidateRawResourceConfigFunc Error",
					},
				},
			},
		},
		"Server with ValidateRawResourceConfigFunc: equal config value returns diags": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test_resource": {
						ValidateRawResourceConfigFuncs: []ValidateRawResourceConfigFunc{
							func(ctx context.Context, req ValidateResourceConfigFuncRequest, resp *ValidateResourceConfigFuncResponse) {
								equals := req.RawConfig.Equals(cty.ObjectVal(map[string]cty.Value{
									"id":  cty.NullVal(cty.String),
									"foo": cty.NumberIntVal(2),
									"bar": cty.NumberIntVal(2),
								}))
								if equals.True() {
									resp.Diagnostics = diag.Diagnostics{
										{
											Severity: diag.Error,
											Summary:  "ValidateRawResourceConfigFunc Error",
										},
									}
								}
							},
							func(ctx context.Context, req ValidateResourceConfigFuncRequest, resp *ValidateResourceConfigFuncResponse) {
								equals := req.RawConfig.Equals(cty.ObjectVal(map[string]cty.Value{
									"id":  cty.NullVal(cty.String),
									"foo": cty.NumberIntVal(2),
									"bar": cty.NumberIntVal(2),
								}))
								if equals.True() {
									resp.Diagnostics = diag.Diagnostics{
										{
											Severity: diag.Error,
											Summary:  "ValidateRawResourceConfigFunc Error",
										},
									}
								}
							},
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
							"bar": {
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			}),
			request: &tfprotov5.ValidateResourceTypeConfigRequest{
				TypeName: "test_resource",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
							"bar": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NumberIntVal(2),
							"bar": cty.NumberIntVal(2),
						}),
					),
				},
			},
			expected: &tfprotov5.ValidateResourceTypeConfigResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "ValidateRawResourceConfigFunc Error",
					},
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "ValidateRawResourceConfigFunc Error",
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp, err := testCase.server.ValidateResourceTypeConfig(context.Background(), testCase.request)

			if testCase.request != nil && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(resp, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestUpgradeState_jsonState(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"two": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.StateUpgraders = []StateUpgrader{
		{
			Version: 0,
			Type: cty.Object(map[string]cty.Type{
				"id":   cty.String,
				"zero": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["zero"].(float64)
				if !ok {
					return nil, fmt.Errorf("zero not found in %#v", m)
				}
				m["one"] = float64(1)
				delete(m, "zero")
				return m, nil
			},
		},
		{
			Version: 1,
			Type: cty.Object(map[string]cty.Type{
				"id":  cty.String,
				"one": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["one"].(float64)
				if !ok {
					return nil, fmt.Errorf("one not found in %#v", m)
				}
				m["two"] = float64(2)
				delete(m, "one")
				return m, nil
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	req := &tfprotov5.UpgradeResourceStateRequest{
		TypeName: "test",
		Version:  0,
		RawState: &tfprotov5.RawState{
			JSON: []byte(`{"id":"bar","zero":0}`),
		},
	}

	resp, err := server.UpgradeResourceState(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	val, err := msgpack.Unmarshal(resp.UpgradedState.MsgPack, r.CoreConfigSchema().ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"two": cty.NumberIntVal(2),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

func TestUpgradeState_jsonStateBigInt(t *testing.T) {
	r := &Resource{
		UseJSONNumber: true,
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"int": {
				Type:     TypeInt,
				Required: true,
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	req := &tfprotov5.UpgradeResourceStateRequest{
		TypeName: "test",
		Version:  0,
		RawState: &tfprotov5.RawState{
			JSON: []byte(`{"id":"bar","int":7227701560655103598}`),
		},
	}

	resp, err := server.UpgradeResourceState(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	val, err := msgpack.Unmarshal(resp.UpgradedState.MsgPack, r.CoreConfigSchema().ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"int": cty.NumberIntVal(7227701560655103598),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

func TestUpgradeState_removedAttr(t *testing.T) {
	r1 := &Resource{
		Schema: map[string]*Schema{
			"two": {
				Type:     TypeString,
				Optional: true,
			},
		},
	}

	r2 := &Resource{
		Schema: map[string]*Schema{
			"multi": {
				Type:     TypeSet,
				Optional: true,
				Elem: &Resource{
					Schema: map[string]*Schema{
						"set": {
							Type:     TypeSet,
							Optional: true,
							Elem: &Resource{
								Schema: map[string]*Schema{
									"required": {
										Type:     TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	r3 := &Resource{
		Schema: map[string]*Schema{
			"config_mode_attr": {
				Type:       TypeList,
				ConfigMode: SchemaConfigModeAttr,
				Optional:   true,
				Elem: &Resource{
					Schema: map[string]*Schema{
						"foo": {
							Type:     TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}

	p := &Provider{
		ResourcesMap: map[string]*Resource{
			"r1": r1,
			"r2": r2,
			"r3": r3,
		},
	}

	server := NewGRPCProviderServer(p)

	for _, tc := range []struct {
		name     string
		raw      string
		expected cty.Value
	}{
		{
			name: "r1",
			raw:  `{"id":"bar","removed":"removed","two":"2"}`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("bar"),
				"two": cty.StringVal("2"),
			}),
		},
		{
			name: "r2",
			raw:  `{"id":"bar","multi":[{"set":[{"required":"ok","removed":"removed"}]}]}`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"set": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"required": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
		},
		{
			name: "r3",
			raw:  `{"id":"bar","config_mode_attr":[{"foo":"ok","removed":"removed"}]}`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
				"config_mode_attr": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("ok"),
					}),
				}),
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &tfprotov5.UpgradeResourceStateRequest{
				TypeName: tc.name,
				Version:  0,
				RawState: &tfprotov5.RawState{
					JSON: []byte(tc.raw),
				},
			}
			resp, err := server.UpgradeResourceState(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}

			if len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					t.Errorf("%#v", d)
				}
				t.Fatal("error")
			}
			val, err := msgpack.Unmarshal(resp.UpgradedState.MsgPack, p.ResourcesMap[tc.name].CoreConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}
			if !tc.expected.RawEquals(val) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.expected, val)
			}
		})
	}

}

func TestUpgradeState_flatmapState(t *testing.T) {
	r := &Resource{
		SchemaVersion: 4,
		Schema: map[string]*Schema{
			"four": {
				Type:     TypeInt,
				Required: true,
			},
			"block": {
				Type:     TypeList,
				Optional: true,
				Elem: &Resource{
					Schema: map[string]*Schema{
						"attr": {
							Type:     TypeString,
							Optional: true,
						},
					},
				},
			},
		},
		// this MigrateState will take the state to version 2
		MigrateState: func(v int, is *terraform.InstanceState, _ interface{}) (*terraform.InstanceState, error) {
			switch v {
			case 0:
				_, ok := is.Attributes["zero"]
				if !ok {
					return nil, fmt.Errorf("zero not found in %#v", is.Attributes)
				}
				is.Attributes["one"] = "1"
				delete(is.Attributes, "zero")
				fallthrough
			case 1:
				_, ok := is.Attributes["one"]
				if !ok {
					return nil, fmt.Errorf("one not found in %#v", is.Attributes)
				}
				is.Attributes["two"] = "2"
				delete(is.Attributes, "one")
			default:
				return nil, fmt.Errorf("invalid schema version %d", v)
			}
			return is, nil
		},
	}

	r.StateUpgraders = []StateUpgrader{
		{
			Version: 2,
			Type: cty.Object(map[string]cty.Type{
				"id":  cty.String,
				"two": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["two"].(float64)
				if !ok {
					return nil, fmt.Errorf("two not found in %#v", m)
				}
				m["three"] = float64(3)
				delete(m, "two")
				return m, nil
			},
		},
		{
			Version: 3,
			Type: cty.Object(map[string]cty.Type{
				"id":    cty.String,
				"three": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["three"].(float64)
				if !ok {
					return nil, fmt.Errorf("three not found in %#v", m)
				}
				m["four"] = float64(4)
				delete(m, "three")
				return m, nil
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	testReqs := []*tfprotov5.UpgradeResourceStateRequest{
		{
			TypeName: "test",
			Version:  0,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":   "bar",
					"zero": "0",
				},
			},
		},
		{
			TypeName: "test",
			Version:  1,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":  "bar",
					"one": "1",
				},
			},
		},
		// two and  up could be stored in flatmap or json states
		{
			TypeName: "test",
			Version:  2,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":  "bar",
					"two": "2",
				},
			},
		},
		{
			TypeName: "test",
			Version:  2,
			RawState: &tfprotov5.RawState{
				JSON: []byte(`{"id":"bar","two":2}`),
			},
		},
		{
			TypeName: "test",
			Version:  3,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":    "bar",
					"three": "3",
				},
			},
		},
		{
			TypeName: "test",
			Version:  3,
			RawState: &tfprotov5.RawState{
				JSON: []byte(`{"id":"bar","three":3}`),
			},
		},
		{
			TypeName: "test",
			Version:  4,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":   "bar",
					"four": "4",
				},
			},
		},
		{
			TypeName: "test",
			Version:  4,
			RawState: &tfprotov5.RawState{
				JSON: []byte(`{"id":"bar","four":4}`),
			},
		},
	}

	for i, req := range testReqs {
		t.Run(fmt.Sprintf("%d-%d", i, req.Version), func(t *testing.T) {
			resp, err := server.UpgradeResourceState(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}

			if len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					t.Errorf("%#v", d)
				}
				t.Fatal("error")
			}

			val, err := msgpack.Unmarshal(resp.UpgradedState.MsgPack, r.CoreConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			expected := cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListValEmpty(cty.Object(map[string]cty.Type{"attr": cty.String})),
				"id":    cty.StringVal("bar"),
				"four":  cty.NumberIntVal(4),
			})

			if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
			}
		})
	}
}

func TestUpgradeState_flatmapStateMissingMigrateState(t *testing.T) {
	r := &Resource{
		SchemaVersion: 1,
		Schema: map[string]*Schema{
			"one": {
				Type:     TypeInt,
				Required: true,
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	testReqs := []*tfprotov5.UpgradeResourceStateRequest{
		{
			TypeName: "test",
			Version:  0,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":  "bar",
					"one": "1",
				},
			},
		},
		{
			TypeName: "test",
			Version:  1,
			RawState: &tfprotov5.RawState{
				Flatmap: map[string]string{
					"id":  "bar",
					"one": "1",
				},
			},
		},
		{
			TypeName: "test",
			Version:  1,
			RawState: &tfprotov5.RawState{
				JSON: []byte(`{"id":"bar","one":1}`),
			},
		},
	}

	for i, req := range testReqs {
		t.Run(fmt.Sprintf("%d-%d", i, req.Version), func(t *testing.T) {
			resp, err := server.UpgradeResourceState(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}

			if len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					t.Errorf("%#v", d)
				}
				t.Fatal("error")
			}

			val, err := msgpack.Unmarshal(resp.UpgradedState.MsgPack, r.CoreConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			expected := cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("bar"),
				"one": cty.NumberIntVal(1),
			})

			if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
			}
		})
	}
}

func TestUpgradeState_writeOnlyNullification(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"two": {
				Type:      TypeInt,
				Optional:  true,
				WriteOnly: true,
			},
		},
	}

	r.StateUpgraders = []StateUpgrader{
		{
			Version: 0,
			Type: cty.Object(map[string]cty.Type{
				"id":   cty.String,
				"zero": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["zero"].(float64)
				if !ok {
					return nil, fmt.Errorf("zero not found in %#v", m)
				}
				m["one"] = float64(1)
				delete(m, "zero")
				return m, nil
			},
		},
		{
			Version: 1,
			Type: cty.Object(map[string]cty.Type{
				"id":  cty.String,
				"one": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["one"].(float64)
				if !ok {
					return nil, fmt.Errorf("one not found in %#v", m)
				}
				m["two"] = float64(2)
				delete(m, "one")
				return m, nil
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	req := &tfprotov5.UpgradeResourceStateRequest{
		TypeName: "test",
		Version:  0,
		RawState: &tfprotov5.RawState{
			JSON: []byte(`{"id":"bar","zero":0}`),
		},
	}

	resp, err := server.UpgradeResourceState(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	val, err := msgpack.Unmarshal(resp.UpgradedState.MsgPack, r.CoreConfigSchema().ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"two": cty.NullVal(cty.Number),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

func TestReadResource(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		req      *tfprotov5.ReadResourceRequest
		expected *tfprotov5.ReadResourceResponse
	}{
		"read-resource": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_bool": {
								Type:     TypeBool,
								Computed: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
							"test_list": {
								Type: TypeList,
								Elem: &Schema{
									Type: TypeString,
								},
								Computed: true,
							},
						},
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"instance_id": {
										Type:              TypeString,
										RequiredForImport: true,
									},
									"region": {
										Type:              TypeString,
										OptionalForImport: true,
									},
								}
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							err := d.Set("test_bool", true)
							if err != nil {
								return diag.FromErr(err)
							}

							err = d.Set("test_string", "new-state-val")
							if err != nil {
								return diag.FromErr(err)
							}

							identity, err := d.Identity()
							if err != nil {
								return diag.FromErr(err)
							}
							err = identity.Set("region", "new-region")
							if err != nil {
								return diag.FromErr(err)
							}

							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadResourceRequest{
				TypeName: "test",
				CurrentIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"instance_id": cty.String,
								"region":      cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"instance_id": cty.StringVal("test-id"),
								"region":      cty.StringVal("test-region"),
							}),
						),
					},
				},
				CurrentState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"test_bool":   cty.Bool,
							"test_string": cty.String,
							"test_list":   cty.List(cty.String),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.StringVal("test-id"),
							"test_bool":   cty.BoolVal(false),
							"test_string": cty.StringVal("prior-state-val"),
							"test_list": cty.ListVal([]cty.Value{
								cty.StringVal("hello"),
								cty.StringVal("world"),
							}),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadResourceResponse{
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"test_bool":   cty.Bool,
							"test_string": cty.String,
							"test_list":   cty.List(cty.String),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.StringVal("test-id"),
							"test_bool":   cty.BoolVal(true),
							"test_string": cty.StringVal("new-state-val"),
							"test_list": cty.ListVal([]cty.Value{
								cty.StringVal("hello"),
								cty.StringVal("world"),
							}),
						}),
					),
				},
				NewIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"instance_id": cty.String,
								"region":      cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"instance_id": cty.StringVal("test-id"),
								"region":      cty.StringVal("new-region"),
							}),
						),
					},
				},
			},
		},
		"no-identity-schema": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Identity: &ResourceIdentity{
							Version: 1,
						},
					},
				},
			}),
			req: &tfprotov5.ReadResourceRequest{
				TypeName: "test",
				CurrentIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"instance_id": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"instance_id": cty.StringVal("test-id"),
							}),
						),
					},
				},
				CurrentState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadResourceResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test': resource does not have an identity schema",
					},
				},
			},
		},
		"empty-identity": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{}
							},
						},
					},
				},
			}),
			req: &tfprotov5.ReadResourceRequest{
				TypeName: "test",
				CurrentIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"instance_id": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"instance_id": cty.StringVal("test-id"),
							}),
						),
					},
				},
				CurrentState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadResourceResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test': identity schema must have at least one attribute",
					},
				},
			},
		},
		"deferred-response-unknown-val": {
			server: NewGRPCProviderServer(&Provider{
				// Deferred response will skip read function and return current state
				providerDeferred: &Deferred{
					Reason: DeferredReasonProviderConfigUnknown,
				},
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_bool": {
								Type:     TypeBool,
								Computed: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							return diag.Errorf("Test assertion failed: read shouldn't be called when provider deferred response is present")
						},
					},
				},
			}),
			req: &tfprotov5.ReadResourceRequest{
				ClientCapabilities: &tfprotov5.ReadResourceClientCapabilities{
					DeferralAllowed: true,
				},
				TypeName: "test",
				CurrentState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"test_bool":   cty.Bool,
							"test_string": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.StringVal("test-id"),
							"test_bool":   cty.BoolVal(false),
							"test_string": cty.StringVal("prior-state-val"),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadResourceResponse{
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"test_bool":   cty.Bool,
							"test_string": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.StringVal("test-id"),
							"test_bool":   cty.BoolVal(false),
							"test_string": cty.StringVal("prior-state-val"),
						}),
					),
				},
				Deferred: &tfprotov5.Deferred{
					Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
				},
			},
		},
		"write-only values are nullified in ReadResourceResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_bool": {
								Type:     TypeBool,
								Computed: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
							"test_write_only": {
								Type:      TypeString,
								WriteOnly: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							err := d.Set("test_bool", true)
							if err != nil {
								return diag.FromErr(err)
							}

							err = d.Set("test_string", "new-state-val")
							if err != nil {
								return diag.FromErr(err)
							}

							err = d.Set("test_write_only", "write-only-val")
							if err != nil {
								return diag.FromErr(err)
							}

							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadResourceRequest{
				TypeName: "test",
				CurrentState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":              cty.String,
							"test_bool":       cty.Bool,
							"test_string":     cty.String,
							"test_write_only": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":              cty.StringVal("test-id"),
							"test_bool":       cty.BoolVal(false),
							"test_string":     cty.StringVal("prior-state-val"),
							"test_write_only": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadResourceResponse{
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":              cty.String,
							"test_bool":       cty.Bool,
							"test_string":     cty.String,
							"test_write_only": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":              cty.StringVal("test-id"),
							"test_bool":       cty.BoolVal(true),
							"test_string":     cty.StringVal("new-state-val"),
							"test_write_only": cty.NullVal(cty.String),
						}),
					),
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := testCase.server.ReadResource(context.Background(), testCase.req)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, testCase.expected, valueComparer); diff != "" {
				ty := testCase.server.getResourceSchemaBlock("test").ImpliedType()

				if resp != nil && resp.NewState != nil {
					t.Logf("resp.NewState.MsgPack: %s", mustMsgpackUnmarshal(ty, resp.NewState.MsgPack))
				}

				if testCase.expected != nil && testCase.expected.NewState != nil {
					t.Logf("expected: %s", mustMsgpackUnmarshal(ty, testCase.expected.NewState.MsgPack))
				}

				t.Error(diff)
			}
		})
	}
}

func TestPlanResourceChange(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		req      *tfprotov5.PlanResourceChangeRequest
		expected *tfprotov5.PlanResourceChangeResponse
	}{
		"basic-plan": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.Number,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.Number,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				PlannedPrivate:              []byte(`{"_new_extra_shim":{}}`),
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"basic-plan-with-identity": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
						},
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"name": {
										Type:              TypeString,
										RequiredForImport: true,
									},
								}
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.Number,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.Number,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				PriorIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"name": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"name": cty.StringVal("test-name"),
							}),
						),
					},
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				PlannedPrivate:              []byte(`{"_new_extra_shim":{}}`),
				UnsafeToUseLegacyTypeSystem: true,
				PlannedIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"name": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"name": cty.StringVal("test-name"),
							}),
						),
					},
				},
			},
		},
		"new-resource-with-identity": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeString,
								Optional: true,
							},
						},
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"name": {
										Type:              TypeString,
										RequiredForImport: true,
									},
								}
							},
						},
						CustomizeDiff: func(ctx context.Context, d *ResourceDiff, meta interface{}) error {
							identity, err := d.Identity()
							if err != nil {
								return err
							}
							err = identity.Set("name", "Peter")
							if err != nil {
								return err
							}
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("baz"),
						}),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("baz"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.StringVal("baz"),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("baz"),
						}),
					),
				},
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				PlannedPrivate:              []byte(`{"_new_extra_shim":{}}`),
				UnsafeToUseLegacyTypeSystem: true,
				PlannedIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"name": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"name": cty.StringVal("Peter"),
							}),
						),
					},
				},
			},
		},
		"no identity schema": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
						},
						Identity: &ResourceIdentity{
							Version: 1,
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.Number,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.Number,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				PriorIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"name": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"name": cty.StringVal("test-name"),
							}),
						),
					},
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test': resource does not have an identity schema",
					},
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"empty identity schema": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
						},
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{}
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.Number,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.Number,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				PriorIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"name": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"name": cty.StringVal("test-name"),
							}),
						),
					},
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test': identity schema must have at least one attribute",
					},
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"basic-plan-EnableLegacyTypeSystemPlanErrors": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						// Will set UnsafeToUseLegacyTypeSystem to false
						EnableLegacyTypeSystemPlanErrors: true,
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.Number,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.Number,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.Number,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.Number),
						}),
					),
				},
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				PlannedPrivate:              []byte(`{"_new_extra_shim":{}}`),
				UnsafeToUseLegacyTypeSystem: false,
			},
		},
		"deferred-with-provider-plan-modification": {
			server: NewGRPCProviderServer(&Provider{
				providerDeferred: &Deferred{
					Reason: DeferredReasonProviderConfigUnknown,
				},
				ResourcesMap: map[string]*Resource{
					"test": {
						ResourceBehavior: ResourceBehavior{
							ProviderDeferred: ProviderDeferredBehavior{
								// Will ensure that CustomizeDiff is called
								EnablePlanModification: true,
							},
						},
						SchemaVersion: 4,
						CustomizeDiff: func(ctx context.Context, d *ResourceDiff, i interface{}) error {
							return d.SetNew("foo", "new-foo-value")
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				ClientCapabilities: &tfprotov5.PlanResourceChangeClientCapabilities{
					DeferralAllowed: true,
				},
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.String,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.UnknownVal(cty.String),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				Deferred: &tfprotov5.Deferred{
					Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("new-foo-value"),
						}),
					),
				},
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				PlannedPrivate:              []byte(`{"_new_extra_shim":{}}`),
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"deferred-skip-plan-modification": {
			server: NewGRPCProviderServer(&Provider{
				providerDeferred: &Deferred{
					Reason: DeferredReasonProviderConfigUnknown,
				},
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						CustomizeDiff: func(ctx context.Context, d *ResourceDiff, i interface{}) error {
							return errors.New("Test assertion failed: CustomizeDiff shouldn't be called")
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				ClientCapabilities: &tfprotov5.PlanResourceChangeClientCapabilities{
					DeferralAllowed: true,
				},
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.String,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("from-config!"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.StringVal("from-config!"),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				Deferred: &tfprotov5.Deferred{
					Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
				},
				// Returns proposed new state with deferred response
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("from-config!"),
						}),
					),
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"create: write-only value can be retrieved in CustomizeDiff": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						CustomizeDiff: func(ctx context.Context, d *ResourceDiff, i interface{}) error {
							val := d.Get("foo")
							if val != "bar" {
								t.Fatalf("Incorrect write-only value")
							}

							return nil
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.String,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("bar"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.StringVal("bar"),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.String),
						}),
					),
				},
				PlannedPrivate: []byte(`{"_new_extra_shim":{}}`),
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"create: write-only values are nullified in PlanResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
							"bar": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.String,
								"bar": cty.String,
							}),
						),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("baz"),
							"bar": cty.StringVal("boop"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.StringVal("baz"),
							"bar": cty.StringVal("boop"),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.NullVal(cty.String),
							"bar": cty.NullVal(cty.String),
						}),
					),
				},
				PlannedPrivate: []byte(`{"_new_extra_shim":{}}`),
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"update: write-only value can be retrieved in CustomizeDiff": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						CustomizeDiff: func(ctx context.Context, d *ResourceDiff, i interface{}) error {
							val := d.Get("write_only")
							if val != "bar" {
								t.Fatalf("Incorrect write-only value")
							}

							return nil
						},
						Schema: map[string]*Schema{
							"configured": {
								Type:     TypeString,
								Optional: true,
							},
							"write_only": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":         cty.String,
							"configured": cty.String,
							"write_only": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":         cty.NullVal(cty.String),
							"configured": cty.StringVal("prior_val"),
							"write_only": cty.NullVal(cty.String),
						}),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":         cty.String,
							"configured": cty.String,
							"write_only": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":         cty.UnknownVal(cty.String),
							"configured": cty.StringVal("updated_val"),
							"write_only": cty.StringVal("bar"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":         cty.String,
							"configured": cty.String,
							"write_only": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":         cty.NullVal(cty.String),
							"configured": cty.StringVal("updated_val"),
							"write_only": cty.StringVal("bar"),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":         cty.String,
							"configured": cty.String,
							"write_only": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":         cty.UnknownVal(cty.String),
							"configured": cty.StringVal("updated_val"),
							"write_only": cty.NullVal(cty.String),
						}),
					),
				},
				PlannedPrivate: []byte(`{"_new_extra_shim":{}}`),
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"update: write-only values are nullified in PlanResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema: map[string]*Schema{
							"configured": {
								Type:     TypeString,
								Optional: true,
							},
							"write_onlyA": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
							"write_onlyB": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.PlanResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.NullVal(cty.String),
							"configured":  cty.StringVal("prior_val"),
							"write_onlyA": cty.NullVal(cty.String),
							"write_onlyB": cty.NullVal(cty.String),
						}),
					),
				},
				ProposedNewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.UnknownVal(cty.String),
							"configured":  cty.StringVal("updated_val"),
							"write_onlyA": cty.StringVal("foo"),
							"write_onlyB": cty.StringVal("bar"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.NullVal(cty.String),
							"configured":  cty.StringVal("updated_val"),
							"write_onlyA": cty.StringVal("foo"),
							"write_onlyB": cty.StringVal("bar"),
						}),
					),
				},
			},
			expected: &tfprotov5.PlanResourceChangeResponse{
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.UnknownVal(cty.String),
							"configured":  cty.StringVal("updated_val"),
							"write_onlyA": cty.NullVal(cty.String),
							"write_onlyB": cty.NullVal(cty.String),
						}),
					),
				},
				PlannedPrivate: []byte(`{"_new_extra_shim":{}}`),
				RequiresReplace: []*tftypes.AttributePath{
					tftypes.NewAttributePath().WithAttributeName("id"),
				},
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp, err := testCase.server.PlanResourceChange(context.Background(), testCase.req)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, testCase.expected, valueComparer); diff != "" {
				ty := testCase.server.getResourceSchemaBlock("test").ImpliedType()

				if resp != nil && resp.PlannedState != nil {
					t.Logf("resp.PlannedState.MsgPack: %s", mustMsgpackUnmarshal(ty, resp.PlannedState.MsgPack))
				}

				if testCase.expected != nil && testCase.expected.PlannedState != nil {
					t.Logf("expected: %s", mustMsgpackUnmarshal(ty, testCase.expected.PlannedState.MsgPack))
				}

				t.Error(diff)
			}
		})
	}
}

func TestPlanResourceChange_bigint(t *testing.T) {
	r := &Resource{
		UseJSONNumber: true,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Required: true,
			},
		},
	}

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			"test": r,
		},
	})

	schema := r.CoreConfigSchema()
	priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	proposedVal := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.UnknownVal(cty.String),
		"foo": cty.MustParseNumberVal("7227701560655103598"),
	})
	proposedState, err := msgpack.Marshal(proposedVal, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id":  cty.NullVal(cty.String),
		"foo": cty.MustParseNumberVal("7227701560655103598"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	testReq := &tfprotov5.PlanResourceChangeRequest{
		TypeName: "test",
		PriorState: &tfprotov5.DynamicValue{
			MsgPack: priorState,
		},
		ProposedNewState: &tfprotov5.DynamicValue{
			MsgPack: proposedState,
		},
		Config: &tfprotov5.DynamicValue{
			MsgPack: configBytes,
		},
	}

	resp, err := server.PlanResourceChange(context.Background(), testReq)
	if err != nil {
		t.Fatal(err)
	}

	plannedStateVal, err := msgpack.Unmarshal(resp.PlannedState.MsgPack, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(proposedVal, plannedStateVal, valueComparer) {
		t.Fatal(cmp.Diff(proposedVal, plannedStateVal, valueComparer))
	}

	plannedStateFoo, acc := plannedStateVal.GetAttr("foo").AsBigFloat().Int64()
	if acc != big.Exact {
		t.Fatalf("Expected exact accuracy, got %s", acc)
	}
	if plannedStateFoo != 7227701560655103598 {
		t.Fatalf("Expected %d, got %d, this represents a loss of precision in planning large numbers", 7227701560655103598, plannedStateFoo)
	}
}

func TestApplyResourceChange(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		req      *tfprotov5.ApplyResourceChangeRequest
		expected *tfprotov5.ApplyResourceChangeResponse
	}{
		"create: write-only values are nullified in ApplyResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
							rd.SetId("baz")
							return nil
						},
						Schema: map[string]*Schema{
							"foo": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
							"bar": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{
								"foo": cty.String,
								"bar": cty.String,
							}),
						),
					),
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.UnknownVal(cty.String),
							"foo": cty.StringVal("baz"),
							"bar": cty.StringVal("boop"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.NullVal(cty.String),
							"foo": cty.StringVal("baz"),
							"bar": cty.StringVal("boop"),
						}),
					),
				},
			},
			expected: &tfprotov5.ApplyResourceChangeResponse{
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":  cty.String,
							"foo": cty.String,
							"bar": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":  cty.StringVal("baz"),
							"foo": cty.NullVal(cty.String),
							"bar": cty.NullVal(cty.String),
						}),
					),
				},
				Private:                     []uint8(`{"schema_version":"4"}`),
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"update: write-only values are nullified in ApplyResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
							rd.SetId("baz")
							s := rd.Get("configured").(string)
							err := rd.Set("configured", s)
							if err != nil {
								return nil
							}
							return nil
						},
						Schema: map[string]*Schema{
							"configured": {
								Type:     TypeString,
								Optional: true,
							},
							"write_onlyA": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
							"write_onlyB": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
					},
				},
			}),
			req: &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.NullVal(cty.String),
							"configured":  cty.StringVal("prior_val"),
							"write_onlyA": cty.NullVal(cty.String),
							"write_onlyB": cty.NullVal(cty.String),
						}),
					),
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.UnknownVal(cty.String),
							"configured":  cty.StringVal("updated_val"),
							"write_onlyA": cty.StringVal("foo"),
							"write_onlyB": cty.StringVal("bar"),
						}),
					),
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.NullVal(cty.String),
							"configured":  cty.StringVal("updated_val"),
							"write_onlyA": cty.StringVal("foo"),
							"write_onlyB": cty.StringVal("bar"),
						}),
					),
				},
			},
			expected: &tfprotov5.ApplyResourceChangeResponse{
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":          cty.String,
							"configured":  cty.String,
							"write_onlyA": cty.String,
							"write_onlyB": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":          cty.StringVal("baz"),
							"configured":  cty.StringVal("updated_val"),
							"write_onlyA": cty.NullVal(cty.String),
							"write_onlyB": cty.NullVal(cty.String),
						}),
					),
				},
				Private:                     []uint8(`{"schema_version":"4"}`),
				UnsafeToUseLegacyTypeSystem: true,
			},
		},
		"create: identity returned in ApplyResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
							rd.SetId("baz")
							identity, err := rd.Identity()
							if err != nil {
								t.Fatal(err)
							}
							err = identity.Set("ident", "bazz")
							if err != nil {
								t.Fatal(err)
							}
							return nil
						},
						Schema: map[string]*Schema{},
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{
									"ident": {
										Type:              TypeString,
										RequiredForImport: true,
									},
								}
							},
						},
					},
				},
			}),
			req: &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{}),
						),
					),
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
						}),
					),
				},
				PlannedIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"ident": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"ident": cty.UnknownVal(cty.String),
							}),
						),
					},
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ApplyResourceChangeResponse{
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("baz"),
						}),
					),
				},
				Private:                     []uint8(`{"schema_version":"4"}`),
				UnsafeToUseLegacyTypeSystem: true,
				NewIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"ident": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"ident": cty.StringVal("bazz"),
							}),
						),
					},
				},
			},
		},
		"create: no identity schema diag in ApplyResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema:        map[string]*Schema{},
						Identity: &ResourceIdentity{
							Version: 1,
						},
					},
				},
			}),
			req: &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{}),
						),
					),
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
						}),
					),
				},
				PlannedIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"ident": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"ident": cty.UnknownVal(cty.String),
							}),
						),
					},
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ApplyResourceChangeResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test': resource does not have an identity schema",
					},
				},
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(cty.DynamicPseudoType, cty.NullVal(cty.DynamicPseudoType)),
				},
			},
		},
		"create: empty identity schema diag in ApplyResourceChangeResponse": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 4,
						Schema:        map[string]*Schema{},
						Identity: &ResourceIdentity{
							Version: 1,
							SchemaFunc: func() map[string]*Schema {
								return map[string]*Schema{}
							},
						},
					},
				},
			}),
			req: &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{}),
						cty.NullVal(
							cty.Object(map[string]cty.Type{}),
						),
					),
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
						}),
					),
				},
				PlannedIdentity: &tfprotov5.ResourceIdentityData{
					IdentityData: &tfprotov5.DynamicValue{
						MsgPack: mustMsgpackMarshal(
							cty.Object(map[string]cty.Type{
								"ident": cty.String,
							}),
							cty.ObjectVal(map[string]cty.Value{
								"ident": cty.UnknownVal(cty.String),
							}),
						),
					},
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ApplyResourceChangeResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "getting identity schema failed for resource 'test': identity schema must have at least one attribute",
					},
				},
				NewState: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(cty.DynamicPseudoType, cty.NullVal(cty.DynamicPseudoType)),
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp, err := testCase.server.ApplyResourceChange(context.Background(), testCase.req)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, testCase.expected, valueComparer); diff != "" {
				ty := testCase.server.getResourceSchemaBlock("test").ImpliedType()

				if resp != nil && resp.NewState != nil {
					t.Logf("resp.NewState.MsgPack: %s", mustMsgpackUnmarshal(ty, resp.NewState.MsgPack))
				}

				if testCase.expected != nil && testCase.expected.NewState != nil {
					t.Logf("expected: %s", mustMsgpackUnmarshal(ty, testCase.expected.NewState.MsgPack))
				}

				t.Error(diff)
			}
		})
	}
}

func TestApplyResourceChange_ResourceFuncs(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		TestResource                   *Resource
		ExpectedUnsafeLegacyTypeSystem bool
	}{
		"Create": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				Create: func(rd *ResourceData, _ interface{}) error {
					rd.SetId("bar")
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateContext": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("bar")
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateWithoutTimeout": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateWithoutTimeout: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("bar")
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"Create_cty": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateWithoutTimeout: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					if rd.GetRawConfig().IsNull() {
						return diag.FromErr(errors.New("null raw config"))
					}
					if !rd.GetRawState().IsNull() {
						return diag.FromErr(fmt.Errorf("non-null raw state: %s", rd.GetRawState().GoString()))
					}
					if rd.GetRawPlan().IsNull() {
						return diag.FromErr(errors.New("null raw plan"))
					}
					rd.SetId("bar")
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateContext_SchemaFunc": {
			TestResource: &Resource{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"id": {
							Type:     TypeString,
							Computed: true,
						},
					}
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("bar") // expected in response
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"EnableLegacyTypeSystemApplyErrors": {
			TestResource: &Resource{
				EnableLegacyTypeSystemApplyErrors: true,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("bar")
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: false,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": testCase.TestResource,
				},
			})

			schema := testCase.TestResource.CoreConfigSchema()
			priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			// A proposed state with only the ID unknown will produce a nil diff, and
			// should return the proposed state value.
			plannedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: priorState,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: plannedState,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: configBytes,
				},
			}

			resp, err := server.ApplyResourceChange(context.Background(), testReq)
			if err != nil {
				t.Fatal(err)
			}

			newStateVal, err := msgpack.Unmarshal(resp.NewState.MsgPack, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			id := newStateVal.GetAttr("id").AsString()
			if id != "bar" {
				t.Fatalf("incorrect final state: %#v\n", newStateVal)
			}

			//nolint:staticcheck // explicitly for this SDK
			if testCase.ExpectedUnsafeLegacyTypeSystem != resp.UnsafeToUseLegacyTypeSystem {
				//nolint:staticcheck // explicitly for this SDK
				t.Fatalf("expected UnsafeLegacyTypeSystem %t, got: %t", testCase.ExpectedUnsafeLegacyTypeSystem, resp.UnsafeToUseLegacyTypeSystem)
			}
		})
	}
}

func TestApplyResourceChange_bigint(t *testing.T) {
	testCases := []struct {
		Description  string
		TestResource *Resource
	}{
		{
			Description: "Create",
			TestResource: &Resource{
				UseJSONNumber: true,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Required: true,
					},
				},
				Create: func(rd *ResourceData, _ interface{}) error {
					rd.SetId("bar")
					return nil
				},
			},
		},
		{
			Description: "CreateContext",
			TestResource: &Resource{
				UseJSONNumber: true,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Required: true,
					},
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("bar")
					return nil
				},
			},
		},
		{
			Description: "CreateWithoutTimeout",
			TestResource: &Resource{
				UseJSONNumber: true,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Required: true,
					},
				},
				CreateWithoutTimeout: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("bar")
					return nil
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			server := NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": testCase.TestResource,
				},
			})

			schema := testCase.TestResource.CoreConfigSchema()
			priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			plannedVal := cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"foo": cty.MustParseNumberVal("7227701560655103598"),
			})
			plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id":  cty.NullVal(cty.String),
				"foo": cty.MustParseNumberVal("7227701560655103598"),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: priorState,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: plannedState,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: configBytes,
				},
			}

			resp, err := server.ApplyResourceChange(context.Background(), testReq)
			if err != nil {
				t.Fatal(err)
			}

			newStateVal, err := msgpack.Unmarshal(resp.NewState.MsgPack, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			id := newStateVal.GetAttr("id").AsString()
			if id != "bar" {
				t.Fatalf("incorrect final state: %#v\n", newStateVal)
			}

			foo, acc := newStateVal.GetAttr("foo").AsBigFloat().Int64()
			if acc != big.Exact {
				t.Fatalf("Expected exact accuracy, got %s", acc)
			}
			if foo != 7227701560655103598 {
				t.Fatalf("Expected %d, got %d, this represents a loss of precision in applying large numbers", 7227701560655103598, foo)
			}
		})
	}
}

func TestApplyResourceChange_ResourceFuncs_writeOnly(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		TestResource                   *Resource
		ExpectedUnsafeLegacyTypeSystem bool
	}{
		"Create: retrieve write-only value using GetRawConfigAt": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
					"write_only_bar": {
						Type:      TypeString,
						Optional:  true,
						WriteOnly: true,
					},
				},
				Create: func(rd *ResourceData, _ interface{}) error {
					rd.SetId("baz")
					writeOnlyVal, err := rd.GetRawConfigAt(cty.GetAttrPath("write_only_bar"))
					if err != nil {
						t.Errorf("Unable to retrieve write only attribute, err: %v", err)
					}
					if writeOnlyVal.AsString() != "bar" {
						t.Errorf("Incorrect write-only value: expected bar but got %s", writeOnlyVal)
					}
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateContext: retrieve write-only value using GetRawConfigAt": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
					"write_only_bar": {
						Type:      TypeString,
						Optional:  true,
						WriteOnly: true,
					},
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("baz")
					writeOnlyVal, err := rd.GetRawConfigAt(cty.GetAttrPath("write_only_bar"))
					if err != nil {
						t.Errorf("Unable to retrieve write only attribute, err: %v", err)
					}
					if writeOnlyVal.AsString() != "bar" {
						t.Errorf("Incorrect write-only value: expected bar but got %s", writeOnlyVal)
					}
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateWithoutTimeout: retrieve write-only value using GetRawConfigAt": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
					"write_only_bar": {
						Type:      TypeString,
						Optional:  true,
						WriteOnly: true,
					},
				},
				CreateWithoutTimeout: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("baz")
					writeOnlyVal, err := rd.GetRawConfigAt(cty.GetAttrPath("write_only_bar"))
					if err != nil {
						t.Errorf("Unable to retrieve write only attribute, err: %v", err)
					}
					if writeOnlyVal.AsString() != "bar" {
						t.Errorf("Incorrect write-only value: expected bar but got %s", writeOnlyVal)
					}
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateContext with SchemaFunc: retrieve write-only value using GetRawConfigAt": {
			TestResource: &Resource{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"id": {
							Type:     TypeString,
							Computed: true,
						},
						"write_only_bar": {
							Type:      TypeString,
							Optional:  true,
							WriteOnly: true,
						},
					}
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("baz")
					writeOnlyVal, err := rd.GetRawConfigAt(cty.GetAttrPath("write_only_bar"))
					if err != nil {
						t.Errorf("Unable to retrieve write only attribute, err: %v", err)
					}
					if writeOnlyVal.AsString() != "bar" {
						t.Errorf("Incorrect write-only value: expected bar but got %s", writeOnlyVal)
					}
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
		"CreateContext: retrieve write-only value using GetRawConfig": {
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
					"write_only_bar": {
						Type:      TypeString,
						Optional:  true,
						WriteOnly: true,
					},
				},
				CreateContext: func(_ context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					rd.SetId("baz")
					if rd.GetRawConfig().IsNull() {
						return diag.FromErr(errors.New("null raw writeOnly val"))
					}
					if rd.GetRawConfig().GetAttr("write_only_bar").Type() != cty.String {
						return diag.FromErr(errors.New("write_only_bar is not of the expected type string"))
					}
					writeOnlyVal := rd.GetRawConfig().GetAttr("write_only_bar").AsString()
					if writeOnlyVal != "bar" {
						t.Errorf("Incorrect write-only value: expected bar but got %s", writeOnlyVal)
					}
					return nil
				},
			},
			ExpectedUnsafeLegacyTypeSystem: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": testCase.TestResource,
				},
			})

			schema := testCase.TestResource.CoreConfigSchema()
			priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			// A proposed state with only the ID unknown will produce a nil diff, and
			// should return the proposed state value.
			plannedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id":             cty.NullVal(cty.String),
				"write_only_bar": cty.StringVal("bar"),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: priorState,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: plannedState,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: configBytes,
				},
			}

			resp, err := server.ApplyResourceChange(context.Background(), testReq)
			if err != nil {
				t.Fatal(err)
			}

			newStateVal, err := msgpack.Unmarshal(resp.NewState.MsgPack, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			id := newStateVal.GetAttr("id").AsString()
			if id != "baz" {
				t.Fatalf("incorrect final state: %#v\n", newStateVal)
			}

			//nolint:staticcheck // explicitly for this SDK
			if testCase.ExpectedUnsafeLegacyTypeSystem != resp.UnsafeToUseLegacyTypeSystem {
				//nolint:staticcheck // explicitly for this SDK
				t.Fatalf("expected UnsafeLegacyTypeSystem %t, got: %t", testCase.ExpectedUnsafeLegacyTypeSystem, resp.UnsafeToUseLegacyTypeSystem)
			}
		})
	}
}

func TestImportResourceState(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		req      *tfprotov5.ImportResourceStateRequest
		expected *tfprotov5.ImportResourceStateResponse
	}{
		"basic-import": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
						},
						Importer: &ResourceImporter{
							StateContext: func(ctx context.Context, d *ResourceData, meta interface{}) ([]*ResourceData, error) {
								err := d.Set("test_string", "new-imported-val")
								if err != nil {
									return nil, err
								}

								return []*ResourceData{d}, nil
							},
						},
					},
				},
			}),
			req: &tfprotov5.ImportResourceStateRequest{
				TypeName: "test",
				ID:       "imported-id",
			},
			expected: &tfprotov5.ImportResourceStateResponse{
				ImportedResources: []*tfprotov5.ImportedResource{
					{
						TypeName: "test",
						State: &tfprotov5.DynamicValue{
							MsgPack: mustMsgpackMarshal(
								cty.Object(map[string]cty.Type{
									"id":          cty.String,
									"test_string": cty.String,
								}),
								cty.ObjectVal(map[string]cty.Value{
									"id":          cty.StringVal("imported-id"),
									"test_string": cty.StringVal("new-imported-val"),
								}),
							),
						},
						Private: []byte(`{"schema_version":"1"}`),
					},
				},
			},
		},
		"resource-doesnt-exist": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
						},
						Importer: &ResourceImporter{
							StateContext: func(ctx context.Context, d *ResourceData, meta interface{}) ([]*ResourceData, error) {
								return nil, errors.New("Test assertion failed: import shouldn't be called")
							},
						},
					},
				},
			}),
			req: &tfprotov5.ImportResourceStateRequest{
				TypeName: "fake-resource",
				ID:       "imported-id",
			},
			expected: &tfprotov5.ImportResourceStateResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "unknown resource type: fake-resource",
					},
				},
			},
		},
		"deferred-response-resource-doesnt-exist": {
			server: NewGRPCProviderServer(&Provider{
				providerDeferred: &Deferred{
					Reason: DeferredReasonProviderConfigUnknown,
				},
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
						},
						Importer: &ResourceImporter{
							StateContext: func(ctx context.Context, d *ResourceData, meta interface{}) ([]*ResourceData, error) {
								return nil, errors.New("Test assertion failed: import shouldn't be called")
							},
						},
					},
				},
			}),
			req: &tfprotov5.ImportResourceStateRequest{
				TypeName: "fake-resource",
				ID:       "imported-id",
				ClientCapabilities: &tfprotov5.ImportResourceStateClientCapabilities{
					DeferralAllowed: true,
				},
			},
			expected: &tfprotov5.ImportResourceStateResponse{
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "unknown resource type: fake-resource",
					},
				},
			},
		},
		"deferred-response-unknown-val": {
			server: NewGRPCProviderServer(&Provider{
				// Deferred response will skip import function and return an unknown value
				providerDeferred: &Deferred{
					Reason: DeferredReasonProviderConfigUnknown,
				},
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_string": {
								Type:     TypeString,
								Computed: true,
							},
						},
						Importer: &ResourceImporter{
							StateContext: func(ctx context.Context, d *ResourceData, meta interface{}) ([]*ResourceData, error) {
								return nil, errors.New("Test assertion failed: import shouldn't be called when deferred response is present")
							},
						},
					},
				},
			}),
			req: &tfprotov5.ImportResourceStateRequest{
				TypeName: "test",
				ID:       "imported-id",
				ClientCapabilities: &tfprotov5.ImportResourceStateClientCapabilities{
					DeferralAllowed: true,
				},
			},
			expected: &tfprotov5.ImportResourceStateResponse{
				Deferred: &tfprotov5.Deferred{
					Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
				},
				ImportedResources: []*tfprotov5.ImportedResource{
					{
						TypeName: "test",
						State: &tfprotov5.DynamicValue{
							MsgPack: mustMsgpackMarshal(
								cty.Object(map[string]cty.Type{
									"id":          cty.String,
									"test_string": cty.String,
								}),
								cty.UnknownVal(
									cty.Object(map[string]cty.Type{
										"id":          cty.String,
										"test_string": cty.String,
									}),
								),
							),
						},
					},
				},
			},
		},
		"write-only-nullification": {
			server: NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
							"test_string": {
								Type:      TypeString,
								Optional:  true,
								WriteOnly: true,
							},
						},
						Importer: &ResourceImporter{
							StateContext: func(ctx context.Context, d *ResourceData, meta interface{}) ([]*ResourceData, error) {
								err := d.Set("test_string", "new-imported-val")
								if err != nil {
									return nil, err
								}

								return []*ResourceData{d}, nil
							},
						},
					},
				},
			}),
			req: &tfprotov5.ImportResourceStateRequest{
				TypeName: "test",
				ID:       "imported-id",
			},
			expected: &tfprotov5.ImportResourceStateResponse{
				ImportedResources: []*tfprotov5.ImportedResource{
					{
						TypeName: "test",
						State: &tfprotov5.DynamicValue{
							MsgPack: mustMsgpackMarshal(
								cty.Object(map[string]cty.Type{
									"id":          cty.String,
									"test_string": cty.String,
								}),
								cty.ObjectVal(map[string]cty.Value{
									"id":          cty.StringVal("imported-id"),
									"test_string": cty.NullVal(cty.String),
								}),
							),
						},
						Private: []byte(`{"schema_version":"1"}`),
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := testCase.server.ImportResourceState(context.Background(), testCase.req)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, testCase.expected, valueComparer); diff != "" {
				ty := testCase.server.getResourceSchemaBlock("test").ImpliedType()

				if resp != nil && len(resp.ImportedResources) > 0 {
					t.Logf("resp.ImportedResources[0].State.MsgPack: %s", mustMsgpackUnmarshal(ty, resp.ImportedResources[0].State.MsgPack))
				}

				if testCase.expected != nil && len(testCase.expected.ImportedResources) > 0 {
					t.Logf("expected: %s", mustMsgpackUnmarshal(ty, testCase.expected.ImportedResources[0].State.MsgPack))
				}

				t.Error(diff)
			}
		})
	}
}

// Timeouts should never be present in imported resources.
// Reference: https://github.com/hashicorp/terraform-plugin-sdk/issues/1145
func TestImportResourceState_Timeouts_None(t *testing.T) {
	t.Parallel()

	resourceDefinition := &Resource{
		Importer: &ResourceImporter{
			StateContext: ImportStatePassthroughContext,
		},
		Schema: map[string]*Schema{
			"string_attribute": {
				Type:     TypeString,
				Optional: true,
			},
		},
	}
	resourceTypeName := "test"

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			resourceTypeName: resourceDefinition,
		},
	})

	schema := resourceDefinition.CoreConfigSchema()

	// Import shim state should not require all attributes.
	stateVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
	}))

	if err != nil {
		t.Fatalf("unable to coerce state value: %s", err)
	}

	testReq := &tfprotov5.ImportResourceStateRequest{
		ID:       "test",
		TypeName: resourceTypeName,
	}

	resp, err := server.ImportResourceState(context.Background(), testReq)

	if err != nil {
		t.Fatalf("unexpected error during ImportResourceState: %s", err)
	}

	if resp == nil {
		t.Fatal("expected ImportResourceState response")
	}

	if len(resp.Diagnostics) > 0 {
		var diagnostics []string

		for _, diagnostic := range resp.Diagnostics {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: %s: %s", diagnostic.Severity, diagnostic.Summary, diagnostic.Detail))
		}

		t.Fatalf("unexpected ImportResourceState diagnostics: %s", strings.Join(diagnostics, " | "))
	}

	if len(resp.ImportedResources) != 1 {
		t.Fatalf("expected 1 ImportedResource, got: %#v", resp.ImportedResources)
	}

	gotStateVal, err := msgpack.Unmarshal(resp.ImportedResources[0].State.MsgPack, schema.ImpliedType())

	if err != nil {
		t.Fatalf("unexpected error during MessagePack unmarshal: %s", err)
	}

	if diff := cmp.Diff(stateVal, gotStateVal, valueComparer); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

// Timeouts should never be present in imported resources.
// Reference: https://github.com/hashicorp/terraform-plugin-sdk/issues/1145
func TestImportResourceState_Timeouts_Removed(t *testing.T) {
	t.Parallel()

	resourceDefinition := &Resource{
		Importer: &ResourceImporter{
			StateContext: ImportStatePassthroughContext,
		},
		Schema: map[string]*Schema{
			"string_attribute": {
				Type:     TypeString,
				Optional: true,
			},
		},
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(10 * time.Minute),
			Read:   DefaultTimeout(10 * time.Minute),
		},
	}
	resourceTypeName := "test"

	server := NewGRPCProviderServer(&Provider{
		ResourcesMap: map[string]*Resource{
			resourceTypeName: resourceDefinition,
		},
	})

	schema := resourceDefinition.CoreConfigSchema()

	// Import shim state should not require all attributes.
	stateVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
	}))

	if err != nil {
		t.Fatalf("unable to coerce state value: %s", err)
	}

	testReq := &tfprotov5.ImportResourceStateRequest{
		ID:       "test",
		TypeName: resourceTypeName,
	}

	resp, err := server.ImportResourceState(context.Background(), testReq)

	if err != nil {
		t.Fatalf("unexpected error during ImportResourceState: %s", err)
	}

	if resp == nil {
		t.Fatal("expected ImportResourceState response")
	}

	if len(resp.Diagnostics) > 0 {
		var diagnostics []string

		for _, diagnostic := range resp.Diagnostics {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: %s: %s", diagnostic.Severity, diagnostic.Summary, diagnostic.Detail))
		}

		t.Fatalf("unexpected ImportResourceState diagnostics: %s", strings.Join(diagnostics, " | "))
	}

	if len(resp.ImportedResources) != 1 {
		t.Fatalf("expected 1 ImportedResource, got: %#v", resp.ImportedResources)
	}

	gotStateVal, err := msgpack.Unmarshal(resp.ImportedResources[0].State.MsgPack, schema.ImpliedType())

	if err != nil {
		t.Fatalf("unexpected error during MessagePack unmarshal: %s", err)
	}

	if diff := cmp.Diff(stateVal, gotStateVal, valueComparer); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestReadDataSource(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		server   *GRPCProviderServer
		req      *tfprotov5.ReadDataSourceRequest
		expected *tfprotov5.ReadDataSourceResponse
	}{
		"missing-set-id": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.NullVal(cty.Object(map[string]cty.Type{
							"id": cty.String,
						})),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
		},
		"empty": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.EmptyObject,
						cty.NullVal(cty.EmptyObject),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
		},
		"SchemaFunc": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaFunc: func() map[string]*Schema {
							return map[string]*Schema{
								"id": {
									Type:     TypeString,
									Computed: true,
								},
							}
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.EmptyObject,
						cty.NullVal(cty.EmptyObject),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
		},
		"null-object": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.NullVal(cty.Object(map[string]cty.Type{
							"id": cty.String,
						})),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
		},
		"computed-id": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
		},
		"optional-computed-id": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
		},
		"optional-no-id": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"test": {
								Type:     TypeString,
								Optional: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":   cty.String,
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":   cty.NullVal(cty.String),
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":   cty.String,
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":   cty.StringVal("test-id"),
							"test": cty.NullVal(cty.String),
						}),
					),
				},
			},
		},
		"required-id": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"id": {
								Type:     TypeString,
								Required: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("test-id"),
						}),
					),
				},
			},
		},
		"required-no-id": {
			server: NewGRPCProviderServer(&Provider{
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"test": {
								Type:     TypeString,
								Required: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							d.SetId("test-id")
							return nil
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":   cty.String,
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":   cty.NullVal(cty.String),
							"test": cty.StringVal("test-string"),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":   cty.String,
							"test": cty.String,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":   cty.StringVal("test-id"),
							"test": cty.StringVal("test-string"),
						}),
					),
				},
			},
		},
		"deferred-response-unknown-val": {
			server: NewGRPCProviderServer(&Provider{
				// Deferred response will skip read function and return an unknown value
				providerDeferred: &Deferred{
					Reason: DeferredReasonProviderConfigUnknown,
				},
				DataSourcesMap: map[string]*Resource{
					"test": {
						SchemaVersion: 1,
						Schema: map[string]*Schema{
							"test": {
								Type:     TypeString,
								Required: true,
							},
							"test_bool": {
								Type:     TypeBool,
								Computed: true,
							},
						},
						ReadContext: func(ctx context.Context, d *ResourceData, meta interface{}) diag.Diagnostics {
							return diag.Errorf("Test assertion failed: read shouldn't be called when provider deferred response is present")
						},
					},
				},
			}),
			req: &tfprotov5.ReadDataSourceRequest{
				ClientCapabilities: &tfprotov5.ReadDataSourceClientCapabilities{
					DeferralAllowed: true,
				},
				TypeName: "test",
				Config: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":        cty.String,
							"test":      cty.String,
							"test_bool": cty.Bool,
						}),
						cty.ObjectVal(map[string]cty.Value{
							"id":        cty.NullVal(cty.String),
							"test":      cty.StringVal("test-string"),
							"test_bool": cty.NullVal(cty.Bool),
						}),
					),
				},
			},
			expected: &tfprotov5.ReadDataSourceResponse{
				State: &tfprotov5.DynamicValue{
					MsgPack: mustMsgpackMarshal(
						cty.Object(map[string]cty.Type{
							"id":        cty.String,
							"test":      cty.String,
							"test_bool": cty.Bool,
						}),
						cty.UnknownVal(
							cty.Object(map[string]cty.Type{
								"id":        cty.String,
								"test":      cty.String,
								"test_bool": cty.Bool,
							}),
						),
					),
				},
				Deferred: &tfprotov5.Deferred{
					Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := testCase.server.ReadDataSource(context.Background(), testCase.req)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(resp, testCase.expected, valueComparer); diff != "" {
				ty := testCase.server.getDatasourceSchemaBlock("test").ImpliedType()

				if resp != nil && resp.State != nil {
					t.Logf("resp.State.MsgPack: %s", mustMsgpackUnmarshal(ty, resp.State.MsgPack))
				}

				if testCase.expected != nil && testCase.expected.State != nil {
					t.Logf("expected: %s", mustMsgpackUnmarshal(ty, testCase.expected.State.MsgPack))
				}

				t.Error(diff)
			}
		})
	}
}

func TestPrepareProviderConfig(t *testing.T) {
	for _, tc := range []struct {
		Name         string
		Schema       map[string]*Schema
		ConfigVal    cty.Value
		ExpectError  string
		ExpectConfig cty.Value
	}{
		{
			Name: "test prepare",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Optional: true,
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
		},
		{
			Name: "test default",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Optional: true,
					Default:  "default",
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("default"),
			}),
		},
		{
			Name: "test defaultfunc",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						return "defaultfunc", nil
					},
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("defaultfunc"),
			}),
		},
		{
			Name: "test default required",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Required: true,
					DefaultFunc: func() (interface{}, error) {
						return "defaultfunc", nil
					},
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("defaultfunc"),
			}),
		},
		{
			Name: "test incorrect type",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Required: true,
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NumberIntVal(3),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("3"),
			}),
		},
		{
			Name: "test incorrect default type",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Optional: true,
					Default:  true,
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("true"),
			}),
		},
		{
			Name: "test incorrect default bool type",
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeBool,
					Optional: true,
					Default:  "",
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.Bool),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.False,
			}),
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			server := NewGRPCProviderServer(&Provider{
				Schema: tc.Schema,
			})

			block := InternalMap(tc.Schema).CoreConfigSchema()

			rawConfig, err := msgpack.Marshal(tc.ConfigVal, block.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.PrepareProviderConfigRequest{
				Config: &tfprotov5.DynamicValue{
					MsgPack: rawConfig,
				},
			}

			resp, err := server.PrepareProviderConfig(context.Background(), testReq)
			if err != nil {
				t.Fatal(err)
			}

			if tc.ExpectError != "" && len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					if !strings.Contains(d.Summary, tc.ExpectError) {
						t.Fatalf("Unexpected error: %s/%s", d.Summary, d.Detail)
					}
				}
				return
			}

			// we should have no errors past this point
			for _, d := range resp.Diagnostics {
				if d.Severity == tfprotov5.DiagnosticSeverityError {
					t.Fatal(resp.Diagnostics)
				}
			}

			val, err := msgpack.Unmarshal(resp.PreparedConfig.MsgPack, block.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			if tc.ExpectConfig.GoString() != val.GoString() {
				t.Fatalf("\nexpected: %#v\ngot: %#v", tc.ExpectConfig, val)
			}
		})
	}
}

func TestGetSchemaTimeouts(t *testing.T) {
	r := &Resource{
		SchemaVersion: 4,
		Timeouts: &ResourceTimeout{
			Create:  DefaultTimeout(time.Second),
			Read:    DefaultTimeout(2 * time.Second),
			Update:  DefaultTimeout(3 * time.Second),
			Default: DefaultTimeout(10 * time.Second),
		},
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	// verify that the timeouts appear in the schema as defined
	block := r.CoreConfigSchema()
	timeoutsBlock := block.BlockTypes["timeouts"]
	if timeoutsBlock == nil {
		t.Fatal("missing timeouts in schema")
	}

	if timeoutsBlock.Attributes["create"] == nil {
		t.Fatal("missing create timeout in schema")
	}
	if timeoutsBlock.Attributes["read"] == nil {
		t.Fatal("missing read timeout in schema")
	}
	if timeoutsBlock.Attributes["update"] == nil {
		t.Fatal("missing update timeout in schema")
	}
	if d := timeoutsBlock.Attributes["delete"]; d != nil {
		t.Fatalf("unexpected delete timeout in schema: %#v", d)
	}
	if timeoutsBlock.Attributes["default"] == nil {
		t.Fatal("missing default timeout in schema")
	}
}

func TestNormalizeNullValues(t *testing.T) {
	for i, tc := range []struct {
		Src, Dst, Expect cty.Value
		Apply            bool
	}{
		{
			// The known set value is copied over the null set value
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.NullVal(cty.String),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.NullVal(cty.String),
					}),
				}),
			}),
			Apply: true,
		},
		{
			// A zero set value is kept
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.String),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.String),
			}),
		},
		{
			// The known set value is copied over the null set value
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.NullVal(cty.String),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			// If we're only in a plan, we can't compare sets at all
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
		},
		{
			// The empty map is copied over the null map
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.String),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.String)),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.String),
			}),
			Apply: true,
		},
		{
			// A zero value primitive is copied over a null primitive
			Src: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Apply: true,
		},
		{
			// Plan primitives are kept
			Src: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NumberIntVal(0),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.Number),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.Number),
			}),
		},
		{
			// Neither plan nor apply should remove empty strings
			Src: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
		},
		{
			// Neither plan nor apply should remove empty strings
			Src: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Apply: true,
		},
		{
			// The null map is retained, because the src was unknown
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.UnknownVal(cty.Map(cty.String)),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.String)),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.String)),
			}),
			Apply: true,
		},
		{
			// the nul set is retained, because the src set contains an unknown value
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			Apply: true,
		},
		{
			// Retain don't re-add unexpected planned values in a map
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal(""),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
		},
		{
			// Remove extra values after apply
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal("b"),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
			Apply: true,
		},
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a"),
			}),
			Dst: cty.EmptyObjectVal,
			Expect: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
		},

		// a list in an object in a list, going from null to empty
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.NullVal(cty.String),
						"name":          cty.StringVal("nic0"),
					})}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Apply: true,
		},

		// a list in an object in a list, going from empty to null
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.NullVal(cty.String),
						"name":          cty.StringVal("nic0"),
					})}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Apply: true,
		},
		// the empty list should be transferred, but the new unknown should not be overridden
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.NullVal(cty.String),
						"name":          cty.StringVal("nic0"),
					})}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
		},
		{
			// fix unknowns added to a map
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal(""),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.UnknownVal(cty.String),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal(""),
				}),
			}),
		},
		{
			// fix unknowns lost from a list
			Src: cty.ObjectVal(map[string]cty.Value{
				"top": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"values": cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
							}),
						}),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"top": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"values": cty.NullVal(cty.List(cty.String)),
							}),
						}),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"top": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"values": cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
							}),
						}),
					}),
				}),
			}),
		},
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				}))),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				})),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				})),
			}),
		},
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				}))),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				})),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				}))),
			}),
			Apply: true,
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := normalizeNullValues(tc.Dst, tc.Src, tc.Apply)
			if !got.RawEquals(tc.Expect) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.Expect, got)
			}
		})
	}
}

func TestValidateNulls(t *testing.T) {
	for i, tc := range []struct {
		Cfg cty.Value
		Err bool
	}{
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.StringVal("string"),
					cty.NullVal(cty.String),
				}),
			}),
			Err: true,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"string": cty.StringVal("string"),
					"null":   cty.NullVal(cty.String),
				}),
			}),
			Err: false,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
				}),
			}),
			Err: true,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
					"list2": cty.ListVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
				}),
			}),
			Err: true,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{
					"list": cty.SetVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
				}),
			}),
			Err: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			d := validateConfigNulls(context.Background(), tc.Cfg, nil)
			diags := convert.ProtoToDiags(d)
			switch {
			case tc.Err:
				if !diags.HasError() {
					t.Fatal("expected error")
				}
			default:
				for _, d := range diags {
					if d.Severity == diag.Error {
						t.Fatalf("unexpected error: %q", d)
					}
				}
			}
		})
	}
}

func TestStopContext_grpc(t *testing.T) {
	testCases := []struct {
		Description  string
		TestResource *Resource
	}{
		{
			Description: "CreateContext",
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateContext: func(ctx context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					<-ctx.Done()
					rd.SetId("bar")
					return nil
				},
			},
		},
		{
			Description: "CreateWithoutTimeout",
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateWithoutTimeout: func(ctx context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					<-ctx.Done()
					rd.SetId("bar")
					return nil
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			server := NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": testCase.TestResource,
				},
			})

			schema := testCase.TestResource.CoreConfigSchema()
			priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			plannedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: priorState,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: plannedState,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: configBytes,
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			ctx = server.StopContext(ctx)
			doneCh := make(chan struct{})
			errCh := make(chan error)
			go func() {
				if _, err := server.ApplyResourceChange(ctx, testReq); err != nil {
					errCh <- err
				}
				close(doneCh)
			}()
			// GRPC request cancel
			cancel()
			select {
			case <-doneCh:
			case err := <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("context cancel did not propagate")
			}
		})
	}
}

func TestStopContext_stop(t *testing.T) {
	testCases := []struct {
		Description  string
		TestResource *Resource
	}{
		{
			Description: "CreateContext",
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateContext: func(ctx context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					<-ctx.Done()
					rd.SetId("bar")
					return nil
				},
			},
		},
		{
			Description: "CreateWithoutTimeout",
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateWithoutTimeout: func(ctx context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					<-ctx.Done()
					rd.SetId("bar")
					return nil
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			server := NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": testCase.TestResource,
				},
			})

			schema := testCase.TestResource.CoreConfigSchema()
			priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			plannedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: priorState,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: plannedState,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: configBytes,
				},
			}

			ctx := server.StopContext(context.Background())
			doneCh := make(chan struct{})
			errCh := make(chan error)
			go func() {
				if _, err := server.ApplyResourceChange(ctx, testReq); err != nil {
					errCh <- err
				}
				close(doneCh)
			}()

			if _, err := server.StopProvider(context.Background(), &tfprotov5.StopProviderRequest{}); err != nil {
				t.Fatalf("unexpected StopProvider error: %s", err)
			}

			select {
			case <-doneCh:
			case err := <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Stop message did not cancel request context")
			}
		})
	}
}

func TestStopContext_stopReset(t *testing.T) {
	testCases := []struct {
		Description  string
		TestResource *Resource
	}{
		{
			Description: "CreateContext",
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateContext: func(ctx context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					<-ctx.Done()
					rd.SetId("bar")
					return nil
				},
			},
		},
		{
			Description: "CreateWithoutTimeout",
			TestResource: &Resource{
				SchemaVersion: 4,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CreateWithoutTimeout: func(ctx context.Context, rd *ResourceData, _ interface{}) diag.Diagnostics {
					<-ctx.Done()
					rd.SetId("bar")
					return nil
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			server := NewGRPCProviderServer(&Provider{
				ResourcesMap: map[string]*Resource{
					"test": testCase.TestResource,
				},
			})

			schema := testCase.TestResource.CoreConfigSchema()
			priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			plannedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			config, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configBytes, err := msgpack.Marshal(config, schema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &tfprotov5.ApplyResourceChangeRequest{
				TypeName: "test",
				PriorState: &tfprotov5.DynamicValue{
					MsgPack: priorState,
				},
				PlannedState: &tfprotov5.DynamicValue{
					MsgPack: plannedState,
				},
				Config: &tfprotov5.DynamicValue{
					MsgPack: configBytes,
				},
			}

			// test first stop
			ctx := server.StopContext(context.Background())
			if ctx.Err() != nil {
				t.Fatal("StopContext does not produce a non-closed context")
			}
			doneCh := make(chan struct{})
			errCh := make(chan error)
			go func(d chan struct{}) {
				if _, err := server.ApplyResourceChange(ctx, testReq); err != nil {
					errCh <- err
				}
				close(d)
			}(doneCh)

			if _, err := server.StopProvider(context.Background(), &tfprotov5.StopProviderRequest{}); err != nil {
				t.Fatalf("unexpected StopProvider error: %s", err)
			}

			select {
			case <-doneCh:
			case err := <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Stop message did not cancel request context")
			}

			// test internal stop synchronization was reset
			ctx = server.StopContext(context.Background())
			if ctx.Err() != nil {
				t.Fatal("StopContext does not produce a non-closed context")
			}
			doneCh = make(chan struct{})
			errCh = make(chan error)
			go func(d chan struct{}) {
				if _, err := server.ApplyResourceChange(ctx, testReq); err != nil {
					errCh <- err
				}
				close(d)
			}(doneCh)

			if _, err := server.StopProvider(context.Background(), &tfprotov5.StopProviderRequest{}); err != nil {
				t.Fatalf("unexpected StopProvider error: %s", err)
			}

			select {
			case <-doneCh:
			case err := <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Stop message did not cancel request context")
			}
		})
	}
}

func Test_pathToAttributePath_noSteps(t *testing.T) {
	res := pathToAttributePath(cty.Path{})
	if res != nil {
		t.Errorf("Expected nil attribute path, got %+v", res)
	}
}

func mustMsgpackMarshal(ty cty.Type, val cty.Value) []byte {
	result, err := msgpack.Marshal(val, ty)

	if err != nil {
		panic(fmt.Sprintf("cannot marshal msgpack: %s\n\ntype: %v\n\nvalue: %v", err, ty, val))
	}

	return result
}

func mustMsgpackUnmarshal(ty cty.Type, b []byte) cty.Value {
	result, err := msgpack.Unmarshal(b, ty)

	if err != nil {
		panic(fmt.Sprintf("cannot unmarshal msgpack: %s\n\ntype: %v\n\nvalue: %v", err, ty, b))
	}

	return result
}
