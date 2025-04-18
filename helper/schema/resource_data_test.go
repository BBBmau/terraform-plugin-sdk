// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceDataGet(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Key    string
		Value  interface{}
	}{
		// #0
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "foo",
						New:         "bar",
						NewComputed: true,
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
		},

		// #1
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Key: "availability_zone",

			Value: "foo",
		},

		// #2
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:      "",
						New:      "foo!",
						NewExtra: "foo",
					},
				},
			},

			Key:   "availability_zone",
			Value: "foo",
		},

		// #3
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
				},
			},

			Diff: nil,

			Key: "availability_zone",

			Value: "bar",
		},

		// #4
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "foo",
						New:         "bar",
						NewComputed: true,
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
		},

		// #5
		{
			Schema: map[string]*Schema{
				"port": {
					Type:     TypeInt,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"port": "80",
				},
			},

			Diff: nil,

			Key: "port",

			Value: 80,
		},

		// #6
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "1",
					"ports.1": "2",
					"ports.2": "5",
				},
			},

			Key: "ports.1",

			Value: 2,
		},

		// #7
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "1",
					"ports.1": "2",
					"ports.2": "5",
				},
			},

			Key: "ports.#",

			Value: 3,
		},

		// #8
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Key: "ports.#",

			Value: 0,
		},

		// #9
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "1",
					"ports.1": "2",
					"ports.2": "5",
				},
			},

			Key: "ports",

			Value: []interface{}{1, 2, 5},
		},

		// #10
		{
			Schema: map[string]*Schema{
				"ingress": {
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": {
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ingress.#": {
						Old: "",
						New: "1",
					},
					"ingress.0.from": {
						Old: "",
						New: "8080",
					},
				},
			},

			Key: "ingress.0",

			Value: map[string]interface{}{
				"from": 8080,
			},
		},

		// #11
		{
			Schema: map[string]*Schema{
				"ingress": {
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": {
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ingress.#": {
						Old: "",
						New: "1",
					},
					"ingress.0.from": {
						Old: "",
						New: "8080",
					},
				},
			},

			Key: "ingress",

			Value: []interface{}{
				map[string]interface{}{
					"from": 8080,
				},
			},
		},

		// #12 Computed get
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},

			Key: "availability_zone",

			Value: "foo",
		},

		// #13 Full object
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Key: "",

			Value: map[string]interface{}{
				"availability_zone": "foo",
			},
		},

		// #14 List of maps
		{
			Schema: map[string]*Schema{
				"config_vars": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Schema{
						Type: TypeMap,
					},
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.#": {
						Old: "0",
						New: "2",
					},
					"config_vars.0.foo": {
						Old: "",
						New: "bar",
					},
					"config_vars.1.bar": {
						Old: "",
						New: "baz",
					},
				},
			},

			Key: "config_vars",

			Value: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"bar": "baz",
				},
			},
		},

		// #15 List of maps in state
		{
			Schema: map[string]*Schema{
				"config_vars": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Schema{
						Type: TypeMap,
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "2",
					"config_vars.0.foo": "baz",
					"config_vars.1.bar": "bar",
				},
			},

			Diff: nil,

			Key: "config_vars",

			Value: []interface{}{
				map[string]interface{}{
					"foo": "baz",
				},
				map[string]interface{}{
					"bar": "bar",
				},
			},
		},

		// #16 List of maps with removal in diff
		{
			Schema: map[string]*Schema{
				"config_vars": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Schema{
						Type: TypeMap,
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "1",
					"config_vars.0.FOO": "bar",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.#": {
						Old: "1",
						New: "0",
					},
					"config_vars.0.FOO": {
						Old:        "bar",
						NewRemoved: true,
					},
				},
			},

			Key: "config_vars",

			Value: []interface{}{},
		},

		// #17 Sets
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":  "1",
					"ports.80": "80",
				},
			},

			Diff: nil,

			Key: "ports",

			Value: []interface{}{80},
		},

		// #18
		{
			Schema: map[string]*Schema{
				"data": {
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": {
								Type:     TypeInt,
								Required: true,
							},

							"value": {
								Type:     TypeString,
								Required: true,
							},
						},
					},
					Set: func(a interface{}) int {
						m := a.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"data.#":        "1",
					"data.10.index": "10",
					"data.10.value": "50",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"data.10.value": {
						Old: "50",
						New: "80",
					},
				},
			},

			Key: "data",

			Value: []interface{}{
				map[string]interface{}{
					"index": 10,
					"value": "80",
				},
			},
		},

		// #19 Empty Set
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Diff: nil,

			Key: "ports",

			Value: []interface{}{},
		},

		// #20 Float zero
		{
			Schema: map[string]*Schema{
				"ratio": {
					Type:     TypeFloat,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Diff: nil,

			Key: "ratio",

			Value: 0.0,
		},

		// #21 Float given
		{
			Schema: map[string]*Schema{
				"ratio": {
					Type:     TypeFloat,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ratio": "0.5",
				},
			},

			Diff: nil,

			Key: "ratio",

			Value: 0.5,
		},

		// #22 Float diff
		{
			Schema: map[string]*Schema{
				"ratio": {
					Type:     TypeFloat,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ratio": "-0.5",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ratio": {
						Old: "-0.5",
						New: "33.0",
					},
				},
			},

			Key: "ratio",

			Value: 33.0,
		},

		// #23 Sets with removed elements
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":  "1",
					"ports.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": {
						Old: "2",
						New: "1",
					},
					"ports.80": {
						Old: "80",
						New: "80",
					},
					"ports.8080": {
						Old:        "8080",
						New:        "0",
						NewRemoved: true,
					},
				},
			},

			Key: "ports",

			Value: []interface{}{80},
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		v := d.Get(tc.Key)
		if s, ok := v.(*Set); ok {
			v = s.List()
		}

		if !reflect.DeepEqual(v, tc.Value) {
			t.Fatalf("Bad: %d\n\n%#v\n\nExpected: %#v", i, v, tc.Value)
		}
	}
}

func TestResourceDataGetChange(t *testing.T) {
	cases := []struct {
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Diff     *terraform.InstanceDiff
		Key      string
		OldValue interface{}
		NewValue interface{}
	}{
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Key: "availability_zone",

			OldValue: "",
			NewValue: "foo",
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Key: "availability_zone",

			OldValue: "foo",
			NewValue: "foo",
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		o, n := d.GetChange(tc.Key)
		if !reflect.DeepEqual(o, tc.OldValue) {
			t.Fatalf("Old Bad: %d\n\n%#v", i, o)
		}
		if !reflect.DeepEqual(n, tc.NewValue) {
			t.Fatalf("New Bad: %d\n\n%#v", i, n)
		}
	}
}

func TestResourceDataGetOk(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Key    string
		Value  interface{}
		Ok     bool
	}{
		/*
		 * Primitives
		 */
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "",
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		/*
		 * Lists
		 */

		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		/*
		 * Map
		 */

		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeMap,
					Optional: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: map[string]interface{}{},
			Ok:    false,
		},

		/*
		 * Set
		 */

		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports.0",
			Value: 0,
			Ok:    false,
		},

		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": {
						Old: "0",
						New: "0",
					},
				},
			},

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		// Further illustrates and clarifies the GetOk semantics from #933, and
		// highlights the limitation that zero-value config is currently
		// indistinguishable from unset config.
		{
			Schema: map[string]*Schema{
				"from_port": {
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"from_port": {
						Old: "",
						New: "0",
					},
				},
			},

			Key:   "from_port",
			Value: 0,
			Ok:    false,
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		v, ok := d.GetOk(tc.Key)
		if s, ok := v.(*Set); ok {
			v = s.List()
		}

		if !reflect.DeepEqual(v, tc.Value) {
			t.Fatalf("Bad: %d\n\n%#v", i, v)
		}
		if ok != tc.Ok {
			t.Fatalf("%d: expected ok: %t, got: %t", i, tc.Ok, ok)
		}
	}
}

func TestResourceDataGetOkExists(t *testing.T) {
	cases := []struct {
		Name   string
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Key    string
		Value  interface{}
		Ok     bool
	}{
		/*
		 * Primitives
		 */
		{
			Name: "string-literal-empty",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "",
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    true,
		},

		{
			Name: "string-computed-empty",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		{
			Name: "string-optional-computed-nil-diff",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		/*
		 * Lists
		 */

		{
			Name: "list-optional",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		/*
		 * Map
		 */

		{
			Name: "map-optional",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeMap,
					Optional: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: map[string]interface{}{},
			Ok:    false,
		},

		/*
		 * Set
		 */

		{
			Name: "set-optional",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		{
			Name: "set-optional-key",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports.0",
			Value: 0,
			Ok:    false,
		},

		{
			Name: "bool-literal-empty",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "",
					},
				},
			},

			Key:   "availability_zone",
			Value: false,
			Ok:    true,
		},

		{
			Name: "bool-literal-set",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						New: "true",
					},
				},
			},

			Key:   "availability_zone",
			Value: true,
			Ok:    true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
			if err != nil {
				t.Fatalf("%s err: %s", tc.Name, err)
			}

			v, ok := d.GetOkExists(tc.Key)
			if s, ok := v.(*Set); ok {
				v = s.List()
			}

			if !reflect.DeepEqual(v, tc.Value) {
				t.Fatalf("Bad %s: \n%#v", tc.Name, v)
			}
			if ok != tc.Ok {
				t.Fatalf("%s: expected ok: %t, got: %t", tc.Name, tc.Ok, ok)
			}
		})
	}
}

func TestResourceDataTimeout(t *testing.T) {
	cases := []struct {
		Name     string
		Rd       *ResourceData
		Expected *ResourceTimeout
	}{
		{
			Name:     "Basic example default",
			Rd:       &ResourceData{timeouts: timeoutForValues(10, 3, 0, 15, 0)},
			Expected: expectedTimeoutForValues(10, 3, 0, 15, 0),
		},
		{
			Name:     "Resource and config match update, create",
			Rd:       &ResourceData{timeouts: timeoutForValues(10, 0, 3, 0, 0)},
			Expected: expectedTimeoutForValues(10, 0, 3, 0, 0),
		},
		{
			Name:     "Resource provides default",
			Rd:       &ResourceData{timeouts: timeoutForValues(10, 0, 0, 0, 7)},
			Expected: expectedTimeoutForValues(10, 7, 7, 7, 7),
		},
		{
			Name:     "Resource provides default and delete",
			Rd:       &ResourceData{timeouts: timeoutForValues(10, 0, 0, 15, 7)},
			Expected: expectedTimeoutForValues(10, 7, 7, 15, 7),
		},
		{
			Name:     "Resource provides default, config overwrites other values",
			Rd:       &ResourceData{timeouts: timeoutForValues(10, 3, 0, 0, 13)},
			Expected: expectedTimeoutForValues(10, 3, 13, 13, 13),
		},
		{
			Name:     "Resource has no config",
			Rd:       &ResourceData{},
			Expected: expectedTimeoutForValues(0, 0, 0, 0, 0),
		},
	}

	keys := timeoutKeys()
	for i, c := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, c.Name), func(t *testing.T) {

			for _, k := range keys {
				got := c.Rd.Timeout(k)
				var ex *time.Duration
				switch k {
				case TimeoutCreate:
					ex = c.Expected.Create
				case TimeoutRead:
					ex = c.Expected.Read
				case TimeoutUpdate:
					ex = c.Expected.Update
				case TimeoutDelete:
					ex = c.Expected.Delete
				case TimeoutDefault:
					ex = c.Expected.Default
				}

				if got > 0 && ex == nil {
					t.Fatalf("Unexpected value in (%s), case %d check 1:\n\texpected: %#v\n\tgot: %#v", k, i, ex, got)
				}
				if got == 0 && ex != nil {
					t.Fatalf("Unexpected value in (%s), case %d check 2:\n\texpected: %#v\n\tgot: %#v", k, i, *ex, got)
				}

				// confirm values
				if ex != nil {
					if got != *ex {
						t.Fatalf("Timeout %s case (%d) expected (%s), got (%s)", k, i, *ex, got)
					}
				}
			}

		})
	}
}

func TestResourceDataHasChanges(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Keys   []string
		Change bool
	}{
		// empty call d.HasChanges()
		{
			Schema: map[string]*Schema{},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},

			Keys: []string{},

			Change: false,
		},
		// neither has change
		{
			Schema: map[string]*Schema{
				"a": {
					Type: TypeString,
				},
				"b": {
					Type: TypeString,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"a": "foo",
					"b": "foo",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"a": {
						Old: "",
						New: "foo",
					},
					"b": {
						Old: "",
						New: "foo",
					},
				},
			},

			Keys: []string{"a", "b"},

			Change: false,
		},
		// one key has change
		{
			Schema: map[string]*Schema{
				"a": {
					Type: TypeString,
				},
				"b": {
					Type: TypeString,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"a": "foo",
					"b": "foo",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"a": {
						Old: "",
						New: "bar",
					},
					"b": {
						Old: "",
						New: "foo",
					},
				},
			},

			Keys: []string{"a", "b"},

			Change: true,
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := d.HasChanges(tc.Keys...)
		if actual != tc.Change {
			t.Fatalf("Bad: %d %#v", i, actual)
		}
	}
}

func TestResourceDataHasChangesExcept(t *testing.T) {
	testCases := map[string]struct {
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Diff     *terraform.InstanceDiff
		Keys     []string
		Expected bool
	}{
		"single self string diff": {
			Schema: map[string]*Schema{
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self": "test1",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self": {
						Old: "test1",
						New: "test2",
					},
				},
			},

			Keys: []string{"self"},

			Expected: false,
		},

		"single self map diff": {
			Schema: map[string]*Schema{
				"self": {
					Type:     TypeMap,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self.Name": {
						Old: "foo",
						New: "foo",
					},
				},
			},

			Keys: []string{"self"},

			Expected: false,
		},

		"single self set diff": {
			Schema: map[string]*Schema{
				"self": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self.#":  "1",
					"self.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self.#": {
						Old: "1",
						New: "0",
					},
				},
			},

			Keys: []string{"self"},

			Expected: false,
		},

		"single other string diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeString,
					Optional: true,
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self": "test",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other": {
						Old: "",
						New: "foo",
					},
				},
			},

			Keys: []string{"self"},

			Expected: true,
		},

		// https://github.com/hashicorp/terraform/issues/927
		"single other map diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeMap,
					Optional: true,
					Elem:     &Schema{Type: TypeString},
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self": "test",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other.foo": {
						Old: "",
						New: "bar",
					},
				},
			},

			Keys: []string{"self"},

			Expected: true,
		},

		"single other set diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"other.#":  "1",
					"other.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other.#": {
						Old: "1",
						New: "0",
					},
				},
			},

			Keys: []string{"self"},

			Expected: true,
		},

		"single other and self diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeString,
					Optional: true,
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other": {
						Old: "",
						New: "test",
					},
					"self": {
						Old: "",
						New: "test",
					},
				},
			},

			Keys: []string{"self"},

			Expected: true,
		},

		"multiple only other diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeString,
					Optional: true,
				},
				"self1": {
					Type:     TypeString,
					Optional: true,
				},
				"self2": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other": {
						Old: "",
						New: "test",
					},
				},
			},

			Keys: []string{"self1", "self2"},

			Expected: true,
		},

		"multiple only self diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeString,
					Optional: true,
				},
				"self1": {
					Type:     TypeString,
					Optional: true,
				},
				"self2": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self1": {
						Old: "",
						New: "test",
					},
					"self2": {
						Old: "",
						New: "test",
					},
				},
			},

			Keys: []string{"self1", "self2"},

			Expected: false,
		},

		"multiple partial self diff one of two": {
			Schema: map[string]*Schema{
				"self1": {
					Type:     TypeString,
					Optional: true,
				},
				"self2": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self1": {
						Old: "",
						New: "test",
					},
				},
			},

			Keys: []string{"self1", "self2"},

			Expected: false,
		},

		"multiple partial self diff one of three": {
			Schema: map[string]*Schema{
				"self1": {
					Type:     TypeString,
					Optional: true,
				},
				"self2": {
					Type:     TypeString,
					Optional: true,
				},
				"self3": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self1": {
						Old: "",
						New: "test",
					},
				},
			},

			Keys: []string{"self1", "self2", "self3"},

			Expected: false,
		},

		"multiple partial self diff two of three": {
			Schema: map[string]*Schema{
				"self1": {
					Type:     TypeString,
					Optional: true,
				},
				"self2": {
					Type:     TypeString,
					Optional: true,
				},
				"self3": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self1": {
						Old: "",
						New: "test",
					},
					"self2": {
						Old: "",
						New: "test",
					},
				},
			},

			Keys: []string{"self1", "self2", "self3"},

			Expected: false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			d, err := schemaMap(testCase.Schema).Data(testCase.State, testCase.Diff)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			actual := d.HasChangesExcept(testCase.Keys...)
			if actual != testCase.Expected {
				t.Errorf("expected: %t, got: %t", testCase.Expected, actual)
			}
		})
	}
}

func TestResourceDataHasChange(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Key    string
		Change bool
	}{
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Key: "availability_zone",

			Change: true,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Key: "availability_zone",

			Change: false,
		},

		{
			Schema: map[string]*Schema{
				"tags": {
					Type:     TypeMap,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"tags.Name": {
						Old: "foo",
						New: "foo",
					},
				},
			},

			Key: "tags",

			Change: true,
		},

		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":  "1",
					"ports.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": {
						Old: "1",
						New: "0",
					},
				},
			},

			Key: "ports",

			Change: true,
		},

		// https://github.com/hashicorp/terraform/issues/927
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":  "1",
					"ports.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"tags.foo": {
						Old: "",
						New: "bar",
					},
				},
			},

			Key: "ports",

			Change: false,
		},

		{
			Schema: map[string]*Schema{
				"network_configuration": {
					Type:     TypeList,
					MaxItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"security_groups": {
								Type:     TypeSet,
								Optional: true,
								Elem:     &Schema{Type: TypeString},
								Set:      HashString,
							},
						},
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"network_configuration.#":                            "1",
					"network_configuration.0.security_groups.#":          "2",
					"network_configuration.0.security_groups.1268622331": "sg2",
					"network_configuration.0.security_groups.3532976705": "sg1",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},

			Key: "network_configuration",

			Change: false,
		},

		{
			Schema: map[string]*Schema{
				"network_configuration": {
					Type:     TypeList,
					MaxItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"security_groups": {
								Type:     TypeSet,
								Optional: true,
								Elem:     &Schema{Type: TypeString},
								Set:      HashString,
							},
						},
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"network_configuration.#":                            "1",
					"network_configuration.0.security_groups.#":          "2",
					"network_configuration.0.security_groups.1268622331": "sg2",
					"network_configuration.0.security_groups.3532976705": "sg1",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"network_configuration.0.security_groups.1268622331": {
						Old: "sg2",
						New: "",
					},
					"network_configuration.0.security_groups.1016763245": {
						Old: "",
						New: "sg3",
					},
				},
			},

			Key: "network_configuration",

			Change: true,
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := d.HasChange(tc.Key)
		if actual != tc.Change {
			t.Fatalf("Bad: %d %#v", i, actual)
		}
	}
}

func TestResourceDataHasChangeExcept(t *testing.T) {
	testCases := map[string]struct {
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Diff     *terraform.InstanceDiff
		Key      string
		Expected bool
	}{
		"self string diff": {
			Schema: map[string]*Schema{
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self": "test1",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self": {
						Old: "test1",
						New: "test2",
					},
				},
			},

			Key: "self",

			Expected: false,
		},

		"self map diff": {
			Schema: map[string]*Schema{
				"self": {
					Type:     TypeMap,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self.Name": {
						Old: "foo",
						New: "foo",
					},
				},
			},

			Key: "self",

			Expected: false,
		},

		"self set diff": {
			Schema: map[string]*Schema{
				"self": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self.#":  "1",
					"self.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"self.#": {
						Old: "1",
						New: "0",
					},
				},
			},

			Key: "self",

			Expected: false,
		},

		"other string diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeString,
					Optional: true,
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self": "test",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other": {
						Old: "",
						New: "foo",
					},
				},
			},

			Key: "self",

			Expected: true,
		},

		// https://github.com/hashicorp/terraform/issues/927
		"other map diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeMap,
					Optional: true,
					Elem:     &Schema{Type: TypeString},
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"self": "test",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other.foo": {
						Old: "",
						New: "bar",
					},
				},
			},

			Key: "self",

			Expected: true,
		},

		"other set diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"other.#":  "1",
					"other.80": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other.#": {
						Old: "1",
						New: "0",
					},
				},
			},

			Key: "self",

			Expected: true,
		},

		"other and self diff": {
			Schema: map[string]*Schema{
				"other": {
					Type:     TypeString,
					Optional: true,
				},
				"self": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"other": {
						Old: "",
						New: "test",
					},
					"self": {
						Old: "",
						New: "test",
					},
				},
			},

			Key: "self",

			Expected: true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			d, err := schemaMap(testCase.Schema).Data(testCase.State, testCase.Diff)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			actual := d.HasChangeExcept(testCase.Key)
			if actual != testCase.Expected {
				t.Errorf("expected: %t, got: %t", testCase.Expected, actual)
			}
		})
	}
}

func TestResourceDataSet(t *testing.T) {
	var testNilPtr *string

	cases := []struct {
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Diff     *terraform.InstanceDiff
		Key      string
		Value    interface{}
		Err      bool
		GetKey   string
		GetValue interface{}

		// GetPreProcess can be set to munge the return value before being
		// compared to GetValue
		GetPreProcess func(interface{}) interface{}
	}{
		// #0: Basic good
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: "foo",

			GetKey:   "availability_zone",
			GetValue: "foo",
		},

		// #1: Basic int
		{
			Schema: map[string]*Schema{
				"port": {
					Type:     TypeInt,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "port",
			Value: 80,

			GetKey:   "port",
			GetValue: 80,
		},

		// #2: Basic bool
		{
			Schema: map[string]*Schema{
				"vpc": {
					Type:     TypeBool,
					Optional: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "vpc",
			Value: true,

			GetKey:   "vpc",
			GetValue: true,
		},

		// #3
		{
			Schema: map[string]*Schema{
				"vpc": {
					Type:     TypeBool,
					Optional: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "vpc",
			Value: false,

			GetKey:   "vpc",
			GetValue: false,
		},

		// #4: Invalid type
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: 80,
			Err:   true,

			GetKey:   "availability_zone",
			GetValue: "",
		},

		// #5: List of primitives, set list
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: []int{1, 2, 5},

			GetKey:   "ports",
			GetValue: []interface{}{1, 2, 5},
		},

		// #6: List of primitives, set list with error
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{1, "NOPE", 5},
			Err:   true,

			GetKey:   "ports",
			GetValue: []interface{}{},
		},

		// #7: Set a list of maps
		{
			Schema: map[string]*Schema{
				"config_vars": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Schema{
						Type: TypeMap,
					},
				},
			},

			State: nil,

			Diff: nil,

			Key: "config_vars",
			Value: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"bar": "baz",
				},
			},
			Err: false,

			GetKey: "config_vars",
			GetValue: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"bar": "baz",
				},
			},
		},

		// #8: Set, with list
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "100",
					"ports.1": "80",
					"ports.2": "80",
				},
			},

			Key:   "ports",
			Value: []interface{}{100, 125, 125},

			GetKey:   "ports",
			GetValue: []interface{}{100, 125},
		},

		// #9: Set, with Set
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":   "3",
					"ports.100": "100",
					"ports.80":  "80",
					"ports.81":  "81",
				},
			},

			Key: "ports",
			Value: &Set{
				m: map[string]interface{}{
					"1": 1,
					"2": 2,
				},
			},

			GetKey:   "ports",
			GetValue: []interface{}{1, 2},
		},

		// #10: Set single item
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":   "2",
					"ports.100": "100",
					"ports.80":  "80",
				},
			},

			Key:   "ports.100",
			Value: 256,
			Err:   true,

			GetKey:   "ports",
			GetValue: []interface{}{100, 80},
		},

		// #11: Set with nested set
		{
			Schema: map[string]*Schema{
				"ports": {
					Type: TypeSet,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"port": {
								Type: TypeInt,
							},

							"set": {
								Type: TypeSet,
								Elem: &Schema{Type: TypeInt},
								Set: func(a interface{}) int {
									return a.(int)
								},
							},
						},
					},
					Set: func(a interface{}) int {
						return a.(map[string]interface{})["port"].(int)
					},
				},
			},

			State: nil,

			Key: "ports",
			Value: []interface{}{
				map[string]interface{}{
					"port": 80,
				},
			},

			GetKey: "ports",
			GetValue: []interface{}{
				map[string]interface{}{
					"port": 80,
					"set":  []interface{}{},
				},
			},

			GetPreProcess: func(v interface{}) interface{} {
				if v == nil {
					return v
				}
				s, ok := v.([]interface{})
				if !ok {
					return v
				}
				for _, v := range s {
					m, ok := v.(map[string]interface{})
					if !ok {
						continue
					}
					if m["set"] == nil {
						continue
					}
					if s, ok := m["set"].(*Set); ok {
						m["set"] = s.List()
					}
				}

				return v
			},
		},

		// #12: List of floats, set list
		{
			Schema: map[string]*Schema{
				"ratios": {
					Type:     TypeList,
					Computed: true,
					Elem:     &Schema{Type: TypeFloat},
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ratios",
			Value: []float64{1.0, 2.2, 5.5},

			GetKey:   "ratios",
			GetValue: []interface{}{1.0, 2.2, 5.5},
		},

		// #12: Set of floats, set list
		{
			Schema: map[string]*Schema{
				"ratios": {
					Type:     TypeSet,
					Computed: true,
					Elem:     &Schema{Type: TypeFloat},
					Set: func(a interface{}) int {
						return int(math.Float64bits(a.(float64)))
					},
				},
			},

			State: nil,

			Diff: nil,

			Key:   "ratios",
			Value: []float64{1.0, 2.2, 5.5},

			GetKey:   "ratios",
			GetValue: []interface{}{1.0, 2.2, 5.5},
		},

		// #13: Basic pointer
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: testPtrTo("foo"),

			GetKey:   "availability_zone",
			GetValue: "foo",
		},

		// #14: Basic nil value
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: testPtrTo(nil),

			GetKey:   "availability_zone",
			GetValue: "",
		},

		// #15: Basic nil pointer
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: testNilPtr,

			GetKey:   "availability_zone",
			GetValue: "",
		},

		// #16: Set in a list
		{
			Schema: map[string]*Schema{
				"ports": {
					Type: TypeList,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"set": {
								Type: TypeSet,
								Elem: &Schema{Type: TypeInt},
								Set: func(a interface{}) int {
									return a.(int)
								},
							},
						},
					},
				},
			},

			State: nil,

			Key: "ports",
			Value: []interface{}{
				map[string]interface{}{
					"set": []interface{}{
						1,
					},
				},
			},

			GetKey: "ports",
			GetValue: []interface{}{
				map[string]interface{}{
					"set": []interface{}{
						1,
					},
				},
			},
			GetPreProcess: func(v interface{}) interface{} {
				if v == nil {
					return v
				}
				s, ok := v.([]interface{})
				if !ok {
					return v
				}
				for _, v := range s {
					m, ok := v.(map[string]interface{})
					if !ok {
						continue
					}
					if m["set"] == nil {
						continue
					}
					if s, ok := m["set"].(*Set); ok {
						m["set"] = s.List()
					}
				}

				return v
			},
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		err = d.Set(tc.Key, tc.Value)
		if err != nil != tc.Err {
			t.Fatalf("%d err: %s", i, err)
		}

		v := d.Get(tc.GetKey)
		if s, ok := v.(*Set); ok {
			v = s.List()
		}

		if tc.GetPreProcess != nil {
			v = tc.GetPreProcess(v)
		}

		if !reflect.DeepEqual(v, tc.GetValue) {
			t.Fatalf("Get Bad: %d\n\n%#v", i, v)
		}
	}
}

func TestResourceDataState_schema(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Set    map[string]interface{}
		Result *terraform.InstanceState
	}{
		// #0 Basic primitive in diff
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
		},

		// #1 Basic primitive set override
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Set: map[string]interface{}{
				"availability_zone": "bar",
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
				},
			},
		},

		// #2
		{
			Schema: map[string]*Schema{
				"vpc": {
					Type:     TypeBool,
					Optional: true,
				},
			},

			State: nil,

			Diff: nil,

			Set: map[string]interface{}{
				"vpc": true,
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"vpc": "true",
				},
			},
		},

		// #3 Basic primitive with StateFunc set
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:      TypeString,
					Optional:  true,
					Computed:  true,
					StateFunc: func(interface{}) string { return "" },
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:      "",
						New:      "foo",
						NewExtra: "foo!",
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
		},

		// #4 List
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "1",
					"ports.0": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": {
						Old: "1",
						New: "2",
					},
					"ports.1": {
						Old: "",
						New: "100",
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.0": "80",
					"ports.1": "100",
				},
			},
		},

		// #5 List of resources
		{
			Schema: map[string]*Schema{
				"ingress": {
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": {
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ingress.#":      "1",
					"ingress.0.from": "80",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ingress.#": {
						Old: "1",
						New: "2",
					},
					"ingress.0.from": {
						Old: "80",
						New: "150",
					},
					"ingress.1.from": {
						Old: "",
						New: "100",
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ingress.#":      "2",
					"ingress.0.from": "150",
					"ingress.1.from": "100",
				},
			},
		},

		// #6 List of maps
		{
			Schema: map[string]*Schema{
				"config_vars": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Schema{
						Type: TypeMap,
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "2",
					"config_vars.0.%":   "2",
					"config_vars.0.foo": "bar",
					"config_vars.0.bar": "bar",
					"config_vars.1.%":   "1",
					"config_vars.1.bar": "baz",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.0.bar": {
						NewRemoved: true,
					},
				},
			},

			Set: map[string]interface{}{
				"config_vars": []map[string]interface{}{
					{
						"foo": "bar",
					},
					{
						"baz": "bang",
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "2",
					"config_vars.0.%":   "1",
					"config_vars.0.foo": "bar",
					"config_vars.1.%":   "1",
					"config_vars.1.baz": "bang",
				},
			},
		},

		// #7 List of maps with removal in diff
		{
			Schema: map[string]*Schema{
				"config_vars": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Schema{
						Type: TypeMap,
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "1",
					"config_vars.0.FOO": "bar",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.#": {
						Old: "1",
						New: "0",
					},
					"config_vars.0.FOO": {
						Old:        "bar",
						NewRemoved: true,
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#": "0",
				},
			},
		},

		// #8 Basic state with other keys
		{
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				ID: "bar",
				Attributes: map[string]string{
					"id": "bar",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Result: &terraform.InstanceState{
				ID: "bar",
				Attributes: map[string]string{
					"id":                "bar",
					"availability_zone": "foo",
				},
			},
		},

		// #9 Sets
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":   "3",
					"ports.100": "100",
					"ports.80":  "80",
					"ports.81":  "81",
				},
			},

			Diff: nil,

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":   "3",
					"ports.80":  "80",
					"ports.81":  "81",
					"ports.100": "100",
				},
			},
		},

		// #10
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Diff: nil,

			Set: map[string]interface{}{
				"ports": []interface{}{100, 80},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":   "2",
					"ports.80":  "80",
					"ports.100": "100",
				},
			},
		},

		// #11
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"order": {
								Type: TypeInt,
							},

							"a": {
								Type: TypeList,
								Elem: &Schema{Type: TypeInt},
							},

							"b": {
								Type: TypeList,
								Elem: &Schema{Type: TypeInt},
							},
						},
					},
					Set: func(a interface{}) int {
						m := a.(map[string]interface{})
						return m["order"].(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":        "2",
					"ports.10.order": "10",
					"ports.10.a.#":   "1",
					"ports.10.a.0":   "80",
					"ports.20.order": "20",
					"ports.20.b.#":   "1",
					"ports.20.b.0":   "100",
				},
			},

			Set: map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"order": 20,
						"b":     []interface{}{100},
					},
					map[string]interface{}{
						"order": 10,
						"a":     []interface{}{80},
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":        "2",
					"ports.10.order": "10",
					"ports.10.a.#":   "1",
					"ports.10.a.0":   "80",
					"ports.10.b.#":   "0",
					"ports.20.order": "20",
					"ports.20.a.#":   "0",
					"ports.20.b.#":   "1",
					"ports.20.b.0":   "100",
				},
			},
		},

		// #12 Maps
		{
			Schema: map[string]*Schema{
				"tags": {
					Type:     TypeMap,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"tags.Name": {
						Old: "",
						New: "foo",
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"tags.%":    "1",
					"tags.Name": "foo",
				},
			},
		},

		// #13 empty computed map
		{
			Schema: map[string]*Schema{
				"tags": {
					Type:     TypeMap,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"tags.Name": {
						Old: "",
						New: "foo",
					},
				},
			},

			Set: map[string]interface{}{
				"tags": map[string]string{},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"tags.%": "0",
				},
			},
		},

		// #14
		{
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": {
						NewComputed: true,
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{},
			},
		},

		// #15
		{
			Schema: map[string]*Schema{
				"foo": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": {
						NewComputed: true,
					},
				},
			},

			Set: map[string]interface{}{
				"foo": "bar",
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "bar",
				},
			},
		},

		// #16 Set of maps
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": {Type: TypeInt},
							"uuids": {Type: TypeMap},
						},
					},
					Set: func(a interface{}) int {
						m := a.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.10.uuids.#": {
						NewComputed: true,
					},
				},
			},

			Set: map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"index": 10,
						"uuids": map[string]interface{}{
							"80": "value",
						},
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":           "1",
					"ports.10.index":    "10",
					"ports.10.uuids.%":  "1",
					"ports.10.uuids.80": "value",
				},
			},
		},

		// #17
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":   "3",
					"ports.100": "100",
					"ports.80":  "80",
					"ports.81":  "81",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": {
						Old: "3",
						New: "0",
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "0",
				},
			},
		},

		// #18
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Diff: nil,

			Set: map[string]interface{}{
				"ports": []interface{}{},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "0",
				},
			},
		},

		// #19
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Diff: nil,

			Set: map[string]interface{}{
				"ports": []interface{}{},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "0",
				},
			},
		},

		// #20 Set lists
		{
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": {Type: TypeInt},
							"uuids": {Type: TypeMap},
						},
					},
				},
			},

			State: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": {
						NewComputed: true,
					},
				},
			},

			Set: map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"index": 10,
						"uuids": map[string]interface{}{
							"80": "value",
						},
					},
				},
			},

			Result: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#":          "1",
					"ports.0.index":    "10",
					"ports.0.uuids.%":  "1",
					"ports.0.uuids.80": "value",
				},
			},
		},
	}

	for i, tc := range cases {
		d, err := schemaMap(tc.Schema).Data(tc.State, tc.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		for k, v := range tc.Set {
			if err := d.Set(k, v); err != nil {
				t.Fatalf("%d err: %s", i, err)
			}
		}

		// Set an ID so that the state returned is not nil
		idSet := false
		if d.Id() == "" {
			idSet = true
			d.SetId("foo")
		}

		actual := d.State()

		// If we set an ID, then undo what we did so the comparison works
		if actual != nil && idSet {
			actual.ID = ""
			delete(actual.Attributes, "id")
		}

		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("Bad: %d\n\n%#v\n\nExpected:\n\n%#v", i, actual, tc.Result)
		}
	}
}

func TestResourceData_nonStringValuesInMap(t *testing.T) {
	cases := []struct {
		Schema       map[string]*Schema
		Diff         *terraform.InstanceDiff
		MapFieldName string
		ItemName     string
		ExpectedType string
	}{
		{
			Schema: map[string]*Schema{
				"boolMap": {
					Type:     TypeMap,
					Elem:     TypeBool,
					Optional: true,
				},
			},
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"boolMap.%": {
						Old: "",
						New: "1",
					},
					"boolMap.boolField": {
						Old: "",
						New: "true",
					},
				},
			},
			MapFieldName: "boolMap",
			ItemName:     "boolField",
			ExpectedType: "bool",
		},
		{
			Schema: map[string]*Schema{
				"intMap": {
					Type:     TypeMap,
					Elem:     TypeInt,
					Optional: true,
				},
			},
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"intMap.%": {
						Old: "",
						New: "1",
					},
					"intMap.intField": {
						Old: "",
						New: "8",
					},
				},
			},
			MapFieldName: "intMap",
			ItemName:     "intField",
			ExpectedType: "int",
		},
		{
			Schema: map[string]*Schema{
				"floatMap": {
					Type:     TypeMap,
					Elem:     TypeFloat,
					Optional: true,
				},
			},
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"floatMap.%": {
						Old: "",
						New: "1",
					},
					"floatMap.floatField": {
						Old: "",
						New: "8.22",
					},
				},
			},
			MapFieldName: "floatMap",
			ItemName:     "floatField",
			ExpectedType: "float64",
		},
	}

	for _, c := range cases {
		d, err := schemaMap(c.Schema).Data(nil, c.Diff)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		m, ok := d.Get(c.MapFieldName).(map[string]interface{})
		if !ok {
			t.Fatalf("expected %q to be castable to a map", c.MapFieldName)
		}
		field, ok := m[c.ItemName]
		if !ok {
			t.Fatalf("expected %q in the map", c.ItemName)
		}

		typeName := reflect.TypeOf(field).Name()
		if typeName != c.ExpectedType {
			t.Fatalf("expected %q to be %q, it is %q.",
				c.ItemName, c.ExpectedType, typeName)
		}
	}
}

func TestResourceDataGetRawConfigAt(t *testing.T) {
	cases := map[string]struct {
		RawConfig     cty.Value
		Path          cty.Path
		Value         cty.Value
		ExpectedDiags diag.Diagnostics
	}{
		"null RawConfig returns error": {
			RawConfig: cty.NullVal(cty.EmptyObject),
			Path:      cty.GetAttrPath("invalid_root_path"),
			Value:     cty.DynamicVal,
			ExpectedDiags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Empty Raw Config",
					Detail: "The Terraform Provider unexpectedly received an empty configuration. " +
						"This is almost always an issue with the Terraform Plugin SDK used to create providers. " +
						"Please report this to the provider developers. \n\n" +
						"The RawConfig is empty.",
					AttributePath: cty.Path{
						cty.GetAttrStep{Name: "invalid_root_path"},
					},
				},
			},
		},
		"null value in config": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.NullVal(cty.Number),
			}),
			Path:  cty.GetAttrPath("ConfigAttribute"),
			Value: cty.NullVal(cty.Number),
		},
		"invalid path returns error": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.NumberIntVal(42),
			}),
			Path:  cty.GetAttrPath("invalid_root_path"),
			Value: cty.DynamicVal,
			ExpectedDiags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Invalid config path",
					Detail: "The Terraform Provider unexpectedly provided a path that does not match the current schema. " +
						"This can happen if the path does not correctly follow the schema in structure or types. " +
						"Please report this to the provider developers. \n\n" +
						"Cannot find config value for given path.",
					AttributePath: cty.Path{
						cty.GetAttrStep{Name: "invalid_root_path"},
					},
				},
			},
		},
		"root level attribute": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.NumberIntVal(42),
			}),
			Path:  cty.GetAttrPath("ConfigAttribute"),
			Value: cty.NumberIntVal(42),
		},
		"root level set attribute": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.SetVal([]cty.Value{
					cty.StringVal("valueA"),
					cty.StringVal("valueB"),
				}),
			}),
			Path:  cty.GetAttrPath("ConfigAttribute").Index(cty.StringVal("valueA")),
			Value: cty.StringVal("valueA"),
		},
		"root level list attribute": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.ListVal([]cty.Value{
					cty.StringVal("valueA"),
					cty.StringVal("valueB"),
				}),
			}),
			Path:  cty.GetAttrPath("ConfigAttribute").IndexInt(0),
			Value: cty.StringVal("valueA"),
		},
		"root level map attribute": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.MapVal(map[string]cty.Value{
					"mapA": cty.StringVal("valueA"),
					"mapB": cty.StringVal("valueB"),
				}),
			}),
			Path:  cty.GetAttrPath("ConfigAttribute").IndexString("mapB"),
			Value: cty.StringVal("valueB"),
		},
		"list nested block attribute - get attribute value": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"list_nested_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"ConfigAttribute": cty.StringVal("valueA"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"ConfigAttribute": cty.StringVal("valueB"),
					}),
				}),
			}),
			Path:  cty.GetAttrPath("list_nested_block").IndexInt(1).GetAttr("ConfigAttribute"),
			Value: cty.StringVal("valueB"),
		},
		"set nested block attribute - get attribute value": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"set_nested_block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"ConfigAttribute": cty.StringVal("valueA"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"ConfigAttribute": cty.StringVal("valueB"),
					}),
				}),
			}),
			Path: cty.GetAttrPath("set_nested_block").Index(cty.ObjectVal(map[string]cty.Value{
				"ConfigAttribute": cty.StringVal("valueB"),
			})).GetAttr("ConfigAttribute"),
			Value: cty.StringVal("valueB"),
		},
		"map nested block attribute - get attribute value": {
			RawConfig: cty.ObjectVal(map[string]cty.Value{
				"map_nested_block": cty.MapVal(map[string]cty.Value{
					"mapA": cty.ObjectVal(map[string]cty.Value{
						"ConfigAttribute": cty.StringVal("valueA"),
					}),
					"mapB": cty.ObjectVal(map[string]cty.Value{
						"ConfigAttribute": cty.StringVal("valueB"),
					}),
				}),
			}),
			Path:  cty.GetAttrPath("map_nested_block").IndexString("mapB").GetAttr("ConfigAttribute"),
			Value: cty.StringVal("valueB"),
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			diff := &terraform.InstanceDiff{
				RawConfig: tc.RawConfig,
			}
			d := &ResourceData{
				diff: diff,
			}

			v, diags := d.GetRawConfigAt(tc.Path)
			if len(diags) != 0 && tc.ExpectedDiags == nil {
				t.Fatalf("expected no diagnostics but got %v", diags)
			}

			if diff := cmp.Diff(tc.ExpectedDiags, diags,
				cmp.AllowUnexported(cty.GetAttrStep{}, cty.IndexStep{}),
				cmp.Comparer(indexStepComparer),
			); diff != "" {
				t.Errorf("Unexpected diagnostics (-wanted +got): %s", diff)
			}

			if !reflect.DeepEqual(v, tc.Value) {
				t.Errorf("Bad: %s\n\n%#v\n\nExpected: %#v", tn, v, tc.Value)
			}
		})
	}
}

func TestResourceDataSetConnInfo(t *testing.T) {
	d := &ResourceData{}
	d.SetId("foo")
	d.SetConnInfo(map[string]string{
		"foo": "bar",
	})

	expected := map[string]string{
		"foo": "bar",
	}

	actual := d.State()
	if !reflect.DeepEqual(actual.Ephemeral.ConnInfo, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDataSetMeta_Timeouts(t *testing.T) {
	d := &ResourceData{}
	d.SetId("foo")

	rt := ResourceTimeout{
		Create: DefaultTimeout(7 * time.Minute),
	}

	d.timeouts = &rt

	expected := expectedForValues(7, 0, 0, 0, 0)

	actual := d.State()
	if !reflect.DeepEqual(actual.Meta[TimeoutKey], expected) {
		t.Fatalf("Bad Meta_timeout match:\n\texpected: %#v\n\tgot: %#v", expected, actual.Meta[TimeoutKey])
	}
}

func TestResourceDataSetId(t *testing.T) {
	d := &ResourceData{
		state: &terraform.InstanceState{
			ID: "test",
			Attributes: map[string]string{
				"id": "test",
			},
		},
	}
	d.SetId("foo")

	actual := d.State()

	// SetId should set both the ID field as well as the attribute, to aid in
	// transitioning to the new type system.
	if actual.ID != "foo" || actual.Attributes["id"] != "foo" {
		t.Fatalf("bad: %#v", actual)
	}

	d.SetId("")
	actual = d.State()
	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDataSetId_clear(t *testing.T) {
	d := &ResourceData{
		state: &terraform.InstanceState{ID: "bar"},
	}
	d.SetId("")

	actual := d.State()
	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDataSetId_override(t *testing.T) {
	d := &ResourceData{
		state: &terraform.InstanceState{ID: "bar"},
	}
	d.SetId("foo")

	actual := d.State()
	if actual.ID != "foo" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDataSetType(t *testing.T) {
	d := &ResourceData{}
	d.SetId("foo")
	d.SetType("bar")

	actual := d.State()
	if v := actual.Ephemeral.Type; v != "bar" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDataIdentity(t *testing.T) {
	d := &ResourceData{
		identitySchema: map[string]*Schema{
			"foo": {
				Type:              TypeString,
				RequiredForImport: true,
			},
		},
	}
	d.SetId("baz") // just required to be able to call .State()
	identity, err := d.Identity()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// test setting
	err = identity.Set("foo", "bar")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// test memoization
	identity2, err := d.Identity()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if identity2.Get("foo").(string) != "bar" {
		t.Fatalf("expected identity to contain value for foo: %#v", identity2)
	}

	// test identity added to state
	state := d.State()
	if state.Identity == nil {
		t.Fatalf("expected identity to be added to state: %#v", state)
	}
	if state.Identity["foo"] != "bar" {
		t.Fatalf("expected identity to contain value for foo: %#v", state)
	}
}

func TestResourceDataIdentity_initial_data_from_state(t *testing.T) {
	d := &ResourceData{
		identitySchema: map[string]*Schema{
			"foo": {
				Type:              TypeString,
				RequiredForImport: true,
			},
		},
		state: &terraform.InstanceState{
			Identity: map[string]string{
				"foo": "bar",
			},
		},
	}
	identity, err := d.Identity()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if identity.Get("foo").(string) != "bar" {
		t.Fatalf("expected identity to contain value for foo: %#v", identity)
	}
}

func TestResourceDataIdentity_initial_data_from_diff(t *testing.T) {
	d := &ResourceData{
		identitySchema: map[string]*Schema{
			"foo": {
				Type:              TypeString,
				RequiredForImport: true,
			},
		},
		// we also keep this to ensure diff takes precedence over state
		state: &terraform.InstanceState{
			Identity: map[string]string{
				"foo": "bar",
			},
		},
		diff: &terraform.InstanceDiff{
			Identity: map[string]string{
				"foo": "baz",
			},
		},
	}
	identity, err := d.Identity()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if identity.Get("foo").(string) != "baz" {
		t.Fatalf("expected identity to contain baz value for foo: %#v", identity)
	}
}

func TestResourceDataIdentity_changing_initial_data(t *testing.T) {
	d := &ResourceData{
		identitySchema: map[string]*Schema{
			"foo": {
				Type:              TypeString,
				RequiredForImport: true,
			},
		},
		diff: &terraform.InstanceDiff{
			Identity: map[string]string{
				"foo": "baz",
			},
		},
	}
	d.SetId("bar") // just required to be able to call .State()
	identity, err := d.Identity()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = identity.Set("foo", "qux")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state := d.State()
	if state.Identity == nil {
		t.Fatalf("expected identity to be added to state: %#v", state)
	}
	if state.Identity["foo"] != "qux" {
		t.Fatalf("expected identity to contain qux value for foo: %#v", state)
	}
}

func TestResourceDataIdentity_no_schema(t *testing.T) {
	d := &ResourceData{}
	_, err := d.Identity()
	if err == nil {
		t.Fatalf("expected error since there's no identity schema, got: nil")
	}
	if diff := cmp.Diff("Resource does not have Identity schema. Please set one in order to use Identity(). This is always a problem in the provider code.", err.Error()); diff != "" {
		t.Fatalf("unexpected error message (-want +got):\n%s", diff)
	}
}

func testPtrTo(raw interface{}) interface{} {
	return &raw
}
