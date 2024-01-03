// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package util

// This package contains utility functions that are shared
// between the manifest provider and the main provider

import (
	"encoding/base64"
	"fmt"
	"strings"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ParseResourceID processes the resource ID string and extracts
// the values for GVK, name and (optionally) namespace of the target resource
//
// The expected format for the resource ID is:
// "apiVersion=<value>,kind=<value>,name=<value>[,namespace=<value>"]
//
// where 'namespace' is only required for resources that expect a namespace.
// Example: "apiVersion=v1,kind=Secret,namespace=default,name=default-token-qgm6s"
func ParseResourceID(id string) (schema.GroupVersionKind, string, string, error) {
	parts := strings.Split(id, ",")
	if len(parts) < 3 || len(parts) > 4 {
		return schema.GroupVersionKind{}, "", "",
			fmt.Errorf("could not parse ID: %q. ID must contain apiVersion, kind, and name", id)
	}

	namespace := "default"
	var apiVersion, kind, name string
	for _, p := range parts {
		pp := strings.Split(p, "=")
		if len(pp) != 2 {
			return schema.GroupVersionKind{}, "", "",
				fmt.Errorf("could not parse ID: %q. ID must be in key=value format", id)
		}
		key := pp[0]
		val := pp[1]
		switch key {
		case "apiVersion":
			apiVersion = val
		case "kind":
			kind = val
		case "name":
			name = val
		case "namespace":
			namespace = val
		default:
			return schema.GroupVersionKind{}, "", "",
				fmt.Errorf("could not parse ID: %q. ID contained unknown key %q", id, key)
		}
	}

	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	return gvk, name, namespace, nil
}

func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}

func Int32Ptr(i int32) *int32 {
	return &i
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func sliceOfString(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, s := range slice {
		result[i] = s.(string)
	}
	return result
}

func base64EncodeStringMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		value := v.(string)
		result[k] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	return result
}

func base64EncodeByteMap(m map[string][]byte) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString(v)
	}
	return result
}

func base64DecodeStringMap(m map[string]interface{}) (map[string][]byte, error) {
	mm := map[string][]byte{}
	for k, v := range m {
		d, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return nil, err
		}
		mm[k] = []byte(d)
	}
	return mm, nil
}

func flattenResourceList(l api.ResourceList) map[string]string {
	m := make(map[string]string)
	for k, v := range l {
		m[string(k)] = v.String()
	}
	return m
}

func idParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		err := fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
		return "", "", err
	}

	return parts[0], parts[1], nil
}

func FlattenByteMapToBase64Map(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	return result
}

func FlattenByteMapToStringMap(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = string(v)
	}
	return result
}

func ExpandBase64MapToByteMap(m map[string]interface{}) map[string][]byte {
	result := make(map[string][]byte)
	for k, v := range m {
		b, err := base64.StdEncoding.DecodeString(v.(string))
		if err == nil {
			result[k] = b
		}
	}
	return result
}

func ExpandStringSlice(s []interface{}) []string {
	result := make([]string, len(s))
	for k, v := range s {
		// Handle the Terraform parser bug which turns empty strings in lists to nil.
		if v == nil {
			result[k] = ""
		} else {
			result[k] = v.(string)
		}
	}
	return result
}
