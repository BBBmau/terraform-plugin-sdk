// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	ctyjson "github.com/hashicorp/go-cty/cty/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform-plugin-sdk/v2/internal/diagutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceApply_create(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	called := false
	r.Create = func(d *ResourceData, m interface{}) error {
		called = true
		d.SetId("foo")
		return nil
	}

	var s *terraform.InstanceState = nil

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "42",
			},
		},
	}

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !called {
		t.Fatal("not called")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_Timeout_state(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(40 * time.Minute),
			Update: DefaultTimeout(80 * time.Minute),
			Delete: DefaultTimeout(40 * time.Minute),
		},
	}

	called := false
	r.Create = func(d *ResourceData, m interface{}) error {
		called = true
		d.SetId("foo")
		return nil
	}

	var s *terraform.InstanceState = nil

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "42",
			},
		},
	}

	diffTimeout := &ResourceTimeout{
		Create: DefaultTimeout(40 * time.Minute),
		Update: DefaultTimeout(80 * time.Minute),
		Delete: DefaultTimeout(40 * time.Minute),
	}

	if err := diffTimeout.DiffEncode(d); err != nil {
		t.Fatalf("Error encoding timeout to diff: %s", err)
	}

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !called {
		t.Fatal("not called")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
			TimeoutKey:       expectedForValues(40, 0, 80, 40, 0),
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Not equal in Timeout State:\n\texpected: %#v\n\tactual: %#v", expected.Meta, actual.Meta)
	}
}

// Regression test to ensure that the meta data is read from state, if a
// resource is destroyed and the timeout meta is no longer available from the
// config
func TestResourceApply_Timeout_destroy(t *testing.T) {
	timeouts := &ResourceTimeout{
		Create: DefaultTimeout(40 * time.Minute),
		Update: DefaultTimeout(80 * time.Minute),
		Delete: DefaultTimeout(40 * time.Minute),
	}

	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: timeouts,
	}

	called := false
	var delTimeout time.Duration
	r.Delete = func(d *ResourceData, m interface{}) error {
		delTimeout = d.Timeout(TimeoutDelete)
		called = true
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
	}

	if err := timeouts.StateEncode(s); err != nil {
		t.Fatalf("Error encoding to state: %s", err)
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !called {
		t.Fatal("delete not called")
	}

	if *timeouts.Delete != delTimeout {
		t.Fatalf("timeouts don't match, expected (%#v), got (%#v)", timeouts.Delete, delTimeout)
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDiff_Timeout_diff(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(40 * time.Minute),
			Update: DefaultTimeout(80 * time.Minute),
			Delete: DefaultTimeout(40 * time.Minute),
		},
	}

	r.Create = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		return nil
	}

	conf := terraform.NewResourceConfigRaw(
		map[string]interface{}{
			"foo": 42,
			TimeoutsConfigKey: map[string]interface{}{
				"create": "2h",
			},
		},
	)
	var s *terraform.InstanceState

	actual, err := r.Diff(context.Background(), s, conf, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "42",
			},
		},
	}

	diffTimeout := &ResourceTimeout{
		Create: DefaultTimeout(120 * time.Minute),
		Update: DefaultTimeout(80 * time.Minute),
		Delete: DefaultTimeout(40 * time.Minute),
	}

	if err := diffTimeout.DiffEncode(expected); err != nil {
		t.Fatalf("Error encoding timeout to diff: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Not equal Meta in Timeout Diff:\n\texpected: %#v\n\tactual: %#v", expected.Meta, actual.Meta)
	}
}

func TestResourceDiff_CustomizeFunc(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	var called bool

	r.CustomizeDiff = func(_ context.Context, d *ResourceDiff, m interface{}) error {
		called = true
		return nil
	}

	conf := terraform.NewResourceConfigRaw(
		map[string]interface{}{
			"foo": 42,
		},
	)

	var s *terraform.InstanceState

	_, err := r.Diff(context.Background(), s, conf, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatalf("diff customization not called")
	}
}

func TestResourceApply_destroy(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	called := false
	r.Delete = func(d *ResourceData, m interface{}) error {
		called = true
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !called {
		t.Fatal("delete not called")
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_destroyCreate(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},

			"tags": {
				Type:     TypeMap,
				Optional: true,
				Computed: true,
			},
		},
	}

	change := false
	r.Create = func(d *ResourceData, m interface{}) error {
		change = d.HasChange("tags")
		d.SetId("foo")
		return nil
	}
	r.Delete = func(d *ResourceData, m interface{}) error {
		return nil
	}

	var s *terraform.InstanceState = &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo":       "bar",
			"tags.Name": "foo",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New:         "42",
				RequiresNew: true,
			},
			"tags.Name": {
				Old:         "foo",
				New:         "foo",
				RequiresNew: true,
			},
		},
	}

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !change {
		t.Fatal("should have change")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":        "foo",
			"foo":       "42",
			"tags.%":    "1",
			"tags.Name": "foo",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_destroyPartial(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
		SchemaVersion: 3,
	}

	r.Delete = func(d *ResourceData, m interface{}) error {
		if err := d.Set("foo", 42); err != nil {
			return fmt.Errorf("unexpected Set error: %s", err)
		}
		return fmt.Errorf("some error")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, err := r.Apply(context.Background(), s, d, nil)
	if err == nil {
		t.Fatal("should error")
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "3",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected:\n%#v\n\ngot:\n%#v", expected, actual)
	}
}

func TestResourceApply_update(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Update = func(d *ResourceData, m interface{}) error {
		if err := d.Set("foo", 42); err != nil {
			return fmt.Errorf("unexpected Set error: %s", err)
		}
		return nil
	}

	s := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "13",
			},
		},
	}

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_updateNoCallback(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Update = nil

	s := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "13",
			},
		},
	}

	actual, err := r.Apply(context.Background(), s, d, nil)
	if err == nil {
		t.Fatal("should error")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_isNewResource(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeString,
				Optional: true,
			},
		},
	}

	updateFunc := func(d *ResourceData, _ interface{}) error {
		if err := d.Set("foo", "updated"); err != nil {
			return fmt.Errorf("unexpected Set error: %s", err)
		}
		if d.IsNewResource() {
			if err := d.Set("foo", "new-resource"); err != nil {
				return fmt.Errorf("unexpected Set error: %s", err)
			}
		}
		return nil
	}
	r.Create = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		if err := d.Set("foo", "created"); err != nil {
			return fmt.Errorf("unexpected Set error: %s", err)
		}
		return updateFunc(d, m)
	}
	r.Update = updateFunc

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "bla-blah",
			},
		},
	}

	// positive test
	var s *terraform.InstanceState = nil

	actual, diags := r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "new-resource",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %#v\nexpected: %#v",
			actual, expected)
	}

	// negative test
	s = &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "new-resource",
		},
	}

	actual, diags = r.Apply(context.Background(), s, d, nil)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	expected = &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "updated",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %#v\nexpected: %#v",
			actual, expected)
	}
}

func TestResourceInternalValidate(t *testing.T) {
	cases := map[string]struct {
		In       *Resource
		Writable bool
		Err      bool
	}{
		"nil": {
			nil,
			true,
			true,
		},

		"No optional and no required": {
			&Resource{
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeInt,
						Optional: true,
						Required: true,
					},
				},
			},
			true,
			true,
		},

		"Update undefined for non-ForceNew field": {
			&Resource{
				Create: Noop,
				Schema: map[string]*Schema{
					"boo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		"Update defined for ForceNew field": {
			&Resource{
				Create: Noop,
				Update: Noop,
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
						ForceNew: true,
					},
				},
			},
			true,
			true,
		},

		"non-writable doesn't need Update, Create or Delete": {
			&Resource{
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			false,
			false,
		},

		"non-writable *must not* have Create": {
			&Resource{
				Create: Noop,
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			false,
			true,
		},

		"writable must have Read": {
			&Resource{
				Create: Noop,
				Update: Noop,
				Delete: Noop,
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		"writable must have Delete": {
			&Resource{
				Create: Noop,
				Read:   Noop,
				Update: Noop,
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		"Reserved name at root should be disallowed": {
			&Resource{
				Create: Noop,
				Read:   Noop,
				Update: Noop,
				Delete: Noop,
				Schema: map[string]*Schema{
					"count": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		"Reserved name at nested levels should be allowed": {
			&Resource{
				Create: Noop,
				Read:   Noop,
				Update: Noop,
				Delete: Noop,
				Schema: map[string]*Schema{
					"parent_list": {
						Type:     TypeString,
						Optional: true,
						Elem: &Resource{
							Schema: map[string]*Schema{
								"provisioner": {
									Type:     TypeString,
									Optional: true,
								},
							},
						},
					},
				},
			},
			true,
			false,
		},

		"Provider reserved name should be allowed in resource": {
			&Resource{
				Create: Noop,
				Read:   Noop,
				Update: Noop,
				Delete: Noop,
				Schema: map[string]*Schema{
					"alias": {
						Type:     TypeString,
						Optional: true,
					},
				},
			},
			true,
			false,
		},

		"ID should be allowed in data source": {
			&Resource{
				Read: Noop,
				Schema: map[string]*Schema{
					"id": {
						Type:     TypeString,
						Optional: true,
					},
				},
			},
			false,
			false,
		},

		"Deprecated ID should be allowed in resource": {
			&Resource{
				Create: Noop,
				Read:   Noop,
				Update: Noop,
				Delete: Noop,
				Schema: map[string]*Schema{
					"id": {
						Type:       TypeString,
						Computed:   true,
						Optional:   true,
						Deprecated: "Use x_id instead",
					},
				},
			},
			true,
			false,
		},

		"non-writable must not define CustomizeDiff": {
			&Resource{
				Read: Noop,
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				CustomizeDiff: func(context.Context, *ResourceDiff, interface{}) error { return nil },
			},
			false,
			true,
		},
		"Deprecated resource": {
			&Resource{
				Read: Noop,
				Schema: map[string]*Schema{
					"goo": {
						Type:     TypeInt,
						Optional: true,
					},
				},
				DeprecationMessage: "This resource has been deprecated.",
			},
			true,
			true,
		},
		"Create and CreateContext should not both be set": {
			&Resource{
				Create:        Noop,
				CreateContext: NoopContext,
				Read:          Noop,
				Update:        Noop,
				Delete:        Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Read and ReadContext should not both be set": {
			&Resource{
				Create:      Noop,
				Read:        Noop,
				ReadContext: NoopContext,
				Update:      Noop,
				Delete:      Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Update and UpdateContext should not both be set": {
			&Resource{
				Create:        Noop,
				Read:          Noop,
				Update:        Noop,
				UpdateContext: NoopContext,
				Delete:        Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Delete and DeleteContext should not both be set": {
			&Resource{
				Create:        Noop,
				Read:          Noop,
				Update:        Noop,
				Delete:        Noop,
				DeleteContext: NoopContext,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Create and CreateWithoutTimeout should not both be set": {
			&Resource{
				Create:               Noop,
				CreateWithoutTimeout: NoopContext,
				Read:                 Noop,
				Update:               Noop,
				Delete:               Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Read and ReadWithoutTimeout should not both be set": {
			&Resource{
				Create:             Noop,
				Read:               Noop,
				ReadWithoutTimeout: NoopContext,
				Update:             Noop,
				Delete:             Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Update and UpdateWithoutTimeout should not both be set": {
			&Resource{
				Create:               Noop,
				Read:                 Noop,
				Update:               Noop,
				UpdateWithoutTimeout: NoopContext,
				Delete:               Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Delete and DeleteWithoutTimeout should not both be set": {
			&Resource{
				Create:               Noop,
				Read:                 Noop,
				Update:               Noop,
				Delete:               Noop,
				DeleteWithoutTimeout: NoopContext,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"CreateContext and CreateWithoutTimeout should not both be set": {
			&Resource{
				CreateContext:        NoopContext,
				CreateWithoutTimeout: NoopContext,
				Read:                 Noop,
				Update:               Noop,
				Delete:               Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"ReadContext and ReadWithoutTimeout should not both be set": {
			&Resource{
				Create:             Noop,
				ReadContext:        NoopContext,
				ReadWithoutTimeout: NoopContext,
				Update:             Noop,
				Delete:             Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"UpdateContext and UpdateWithoutTimeout should not both be set": {
			&Resource{
				Create:               Noop,
				Read:                 Noop,
				UpdateContext:        NoopContext,
				UpdateWithoutTimeout: NoopContext,
				Delete:               Noop,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"DeleteContext and DeleteWithoutTimeout should not both be set": {
			&Resource{
				Create:               Noop,
				Read:                 Noop,
				Update:               Noop,
				DeleteContext:        NoopContext,
				DeleteWithoutTimeout: NoopContext,
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			true,
			true,
		},
		"Non-Writable SchemaFunc and Schema should not both be set": {
			In: &Resource{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Required: true,
					},
				},
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"test": {
							Type:     TypeString,
							Required: true,
						},
					}
				},
				Read: Noop,
			},
			Writable: false,
			Err:      true,
		},
		"Writable SchemaFunc and Schema should not both be set": {
			In: &Resource{
				Schema: map[string]*Schema{
					"test": {
						Type:     TypeString,
						Required: true,
					},
				},
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"test": {
							Type:     TypeString,
							Required: true,
						},
					}
				},
				Create: Noop,
				Read:   Noop,
				Update: Noop,
				Delete: Noop,
			},
			Writable: true,
			Err:      true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sm := schemaMap{}
			if tc.In != nil {
				sm = schemaMap(tc.In.Schema)
			}
			err := tc.In.InternalValidate(sm, tc.Writable)
			if err != nil && !tc.Err {
				t.Fatalf("%s: expected validation to pass: %s", name, err)
			}
			if err == nil && tc.Err {
				t.Fatalf("%s: expected validation to fail", name)
			}
		})
	}
}

func TestResourceRefresh(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		if m != 42 {
			return fmt.Errorf("meta not passed")
		}

		return d.Set("foo", d.Get("foo").(int)+1)
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "13",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	actual, diags := r.RefreshWithoutUpgrade(context.Background(), s, 42)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_DiffSuppressOnRefresh(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeString,
				Optional: true,
				DiffSuppressFunc: func(key, oldV, newV string, d *ResourceData) bool {
					return true
				},
				DiffSuppressOnRefresh: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("foo", "howdy")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "hello",
		},
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "hello", // new value was suppressed
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	actual, diags := r.RefreshWithoutUpgrade(context.Background(), s, 42)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_blankId(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		return nil
	}

	s := &terraform.InstanceState{
		ID:         "",
		Attributes: map[string]string{},
	}

	actual, diags := r.RefreshWithoutUpgrade(context.Background(), s, 42)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}
	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_delete(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		d.SetId("")
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, diags := r.RefreshWithoutUpgrade(context.Background(), s, 42)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_existsError(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Exists = func(*ResourceData, interface{}) (bool, error) {
		return false, fmt.Errorf("error")
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		panic("shouldn't be called")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.RefreshWithoutUpgrade(context.Background(), s, 42)
	if err == nil {
		t.Fatalf("should error")
	}
	if !reflect.DeepEqual(actual, s) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_noExists(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Exists = func(*ResourceData, interface{}) (bool, error) {
		return false, nil
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		panic("shouldn't be called")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, diags := r.RefreshWithoutUpgrade(context.Background(), s, 42)
	if diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}
	if actual != nil {
		t.Fatalf("should have no state")
	}
}

func TestResourceData(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	state := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
	}

	data := r.Data(state)
	if data.Id() != "foo" {
		t.Fatalf("err: %s", data.Id())
	}
	if v := data.Get("foo"); v != 42 {
		t.Fatalf("bad: %#v", v)
	}

	// Set expectations
	state.Meta = map[string]interface{}{
		"schema_version": "2",
	}

	result := data.State()
	if !reflect.DeepEqual(result, state) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestResourceData_blank(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	data := r.Data(nil)
	if data.Id() != "" {
		t.Fatalf("err: %s", data.Id())
	}
	if v := data.Get("foo"); v != 0 {
		t.Fatalf("bad: %#v", v)
	}
}

func TestResourceData_timeouts(t *testing.T) {
	one := 1 * time.Second
	two := 2 * time.Second
	three := 3 * time.Second
	four := 4 * time.Second
	five := 5 * time.Second

	timeouts := &ResourceTimeout{
		Create:  &one,
		Read:    &two,
		Update:  &three,
		Delete:  &four,
		Default: &five,
	}

	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: timeouts,
	}

	data := r.Data(nil)
	if data.Id() != "" {
		t.Fatalf("err: %s", data.Id())
	}

	if !reflect.DeepEqual(timeouts, data.timeouts) {
		t.Fatalf("incorrect ResourceData timeouts: %#v\n", *data.timeouts)
	}
}

func TestResource_UpgradeState(t *testing.T) {
	// While this really only calls itself and therefore doesn't test any of
	// the Resource code directly, it still serves as an example of registering
	// a StateUpgrader.
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.StateUpgraders = []StateUpgrader{
		{
			Version: 1,
			Type: cty.Object(map[string]cty.Type{
				"id":     cty.String,
				"oldfoo": cty.Number,
			}),
			Upgrade: func(ctx context.Context, m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {

				oldfoo, ok := m["oldfoo"].(float64)
				if !ok {
					t.Fatalf("expected 1.2, got %#v", m["oldfoo"])
				}
				m["newfoo"] = int(oldfoo * 10)
				delete(m, "oldfoo")

				return m, nil
			},
		},
	}

	oldStateAttrs := map[string]string{
		"id":     "bar",
		"oldfoo": "1.2",
	}

	// convert the legacy flatmap state to the json equivalent
	ty := r.StateUpgraders[0].Type
	val, err := hcl2shim.HCL2ValueFromFlatmap(oldStateAttrs, ty)
	if err != nil {
		t.Fatal(err)
	}
	js, err := ctyjson.Marshal(val, ty)
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal the state using the json default types
	var m map[string]interface{}
	if err := json.Unmarshal(js, &m); err != nil {
		t.Fatal(err)
	}

	actual, err := r.StateUpgraders[0].Upgrade(context.Background(), m, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{
		"id":     "bar",
		"newfoo": 12,
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected: %#v\ngot: %#v\n", expected, actual)
	}
}

func TestResource_ValidateUpgradeState(t *testing.T) {
	r := &Resource{
		SchemaVersion: 3,
		Schema: map[string]*Schema{
			"newfoo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 2,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(ctx context.Context, m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	// check for missing type
	r.StateUpgraders[0].Type = cty.Type{}
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgrader must have type")
	}
	r.StateUpgraders[0].Type = cty.Object(map[string]cty.Type{
		"id": cty.String,
	})

	// check for missing Upgrade func
	r.StateUpgraders[0].Upgrade = nil
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgrader must have an Upgrade func")
	}
	r.StateUpgraders[0].Upgrade = func(ctx context.Context, m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
		return m, nil
	}

	// check for skipped version
	r.StateUpgraders[0].Version = 0
	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 2,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(ctx context.Context, m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgraders cannot skip versions")
	}

	// add the missing version, but fail because it's still out of order
	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 1,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(ctx context.Context, m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("upgraders must be defined in order")
	}

	r.StateUpgraders[1], r.StateUpgraders[2] = r.StateUpgraders[2], r.StateUpgraders[1]
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	// can't add an upgrader for a schema >= the current version
	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 3,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(ctx context.Context, m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgraders cannot have a version >= current SchemaVersion")
	}
}

func TestResource_ContextTimeout(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": {
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(40 * time.Minute),
		},
	}

	var deadlineSet bool
	r.CreateContext = func(ctx context.Context, d *ResourceData, m interface{}) diag.Diagnostics {
		d.SetId("foo")
		_, deadlineSet = ctx.Deadline()
		return nil
	}

	var s *terraform.InstanceState = nil

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": {
				New: "42",
			},
		},
	}

	if _, diags := r.Apply(context.Background(), s, d, nil); diags.HasError() {
		t.Fatalf("err: %s", diagutils.ErrorDiags(diags))
	}

	if !deadlineSet {
		t.Fatal("context does not have timeout")
	}
}

func TestResourceInternalIdentityValidate(t *testing.T) {
	cases := map[string]struct {
		In  *ResourceIdentity
		Err bool
	}{
		"nil": {
			nil,
			true,
		},

		"schema is nil": {
			&ResourceIdentity{},
			true,
		},

		"OptionalForImport and RequiredForImport both false": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							OptionalForImport: false,
							RequiredForImport: false,
						},
					}
				},
			},
			true,
		},

		"OptionalForImport and RequiredForImport both true": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							OptionalForImport: true,
							RequiredForImport: true,
						},
					}
				},
			},
			true,
		},

		"TypeMap is not valid": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {Type: TypeMap, OptionalForImport: true},
					}
				},
			},
			true,
		},

		"TypeSet is not valid": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {Type: TypeSet, OptionalForImport: true},
					}
				},
			},
			true,
		},

		"TypeObject is not valid": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {Type: typeObject, OptionalForImport: true},
					}
				},
			},
			true,
		},

		"TypeInvalid is not valid": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {Type: TypeInvalid, OptionalForImport: true},
					}
				},
			},
			true,
		},

		"TypeList contains TypeMap": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type: TypeList, Elem: TypeMap, OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		" TypeList contains TypeSet": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type: TypeList,
							Elem: &Resource{
								Schema: map[string]*Schema{
									"bar": {
										Type:              TypeSet,
										RequiredForImport: true,
									},
								},
							},
						},
					}
				},
			},
			true,
		},

		"TypeList contains TypeInvalid": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type: TypeList, Elem: TypeInvalid, OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"ForceNew is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							ForceNew:          true,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"Optional is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							Optional:          true,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"Required is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							Required:          true,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"WriteOnly is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							WriteOnly:         true,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"Computed is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							Computed:          true,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"Deprecated is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							Deprecated:        "deprecated",
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"Default is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							Default:           42,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"MaxItems is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							MaxItems:          5,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"MinItems is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							MinItems:          1,
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"DiffSuppressOnRefresh is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:                  TypeInt,
							DiffSuppressOnRefresh: true,
							OptionalForImport:     true,
						},
					}
				},
			},
			true,
		},

		"RequiredWith is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							RequiredWith:      []string{"bar"},
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"ComputedWhen is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							ComputedWhen:      []string{"bar"},
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"DefaultFunc is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							DefaultFunc:       func() (interface{}, error) { return 42, nil },
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"StateFunc is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							StateFunc:         func(val interface{}) string { return "" },
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"ValidateFunc is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							ValidateFunc:      func(val interface{}, key string) (ws []string, es []error) { return nil, nil },
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"AtLeastOneOf is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							AtLeastOneOf:      []string{"bar"},
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"ConflictsWith is set": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type:              TypeInt,
							ConflictsWith:     []string{"bar"},
							OptionalForImport: true,
						},
					}
				},
			},
			true,
		},

		"Valid resource identity OptionalForImport": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type: TypeInt, OptionalForImport: true},
					}
				},
			},
			false,
		},

		"Valid resource identity RequiredorImport": {
			&ResourceIdentity{
				SchemaFunc: func() map[string]*Schema {
					return map[string]*Schema{
						"foo": {
							Type: TypeInt, RequiredForImport: true},
					}
				},
			},
			false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.In.InternalIdentityValidate()
			if err != nil && !tc.Err {
				t.Fatalf("%s: expected validation to pass: %s", name, err)
			}
			if err == nil && tc.Err {
				t.Fatalf("%s: expected validation to fail", name)
			}
		})
	}
}
