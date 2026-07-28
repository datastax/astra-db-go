// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serdes

import (
	"encoding/json"
	"fmt"
)

// RawMap is a partially-deserialized map that retains the DecodeCtx from the
// original decode call. This allows lazy per-field deserialization to correctly
// apply field hints (e.g. $vector → datatypes.Vector) that would otherwise be
// lost when raw bytes are stored and decoded independently later.
type RawMap struct {
	Data map[string]json.RawMessage
	ctx  DecodeCtx // retained w/o payload - payload is ephemeral per-call
}

// UnmarshalAstraRaw captures the decode context and decodes the value as a
// map[string]json.RawMessage, deferring per-field deserialization to GetAny/DecodeAny.
func (m *RawMap) UnmarshalAstraRaw(ctx DecodeCtx, value []byte) error {
	ctx.payload = nil // don't hold a pointer into old data
	m.ctx = ctx
	m.Data = make(map[string]json.RawMessage)
	return Deserialize(value, &m.Data, nil, ctx.Target, ctx.Flags)
}

// Flags returns the DesFlags that were active when this map was decoded.
func (m *RawMap) Flags() DesFlags {
	return m.ctx.Flags
}

// Ctx returns the DecodeCtx that was active when this map was decoded.
// Prefer this over manually reconstructing a DecodeCtx from Target/TargetCtx/Flags.
func (m *RawMap) Ctx() DecodeCtx {
	return m.ctx
}

// GetAny deserializes the value for key as an any, applying the correct field
// hint derived from the key name (e.g. "$vector" → datatypes.Vector).
func (m *RawMap) GetAny(key string) (any, bool) {
	raw, ok := m.Data[key]
	if !ok {
		return nil, false
	}
	return m.GetField(key, raw)
}

// DecodeAny deserializes the value for key into dest, applying the correct
// field hint derived from the key name.
func (m *RawMap) DecodeAny(key string, dest any) error {
	raw, ok := m.Data[key]
	if !ok {
		return fmt.Errorf("astra: key %q not found", key)
	}
	return m.DecodeField(key, raw, dest)
}

// GetField deserializes raw as an any, applying the field hint for fieldName.
// Use this when you've already navigated to the raw bytes and know the field name.
func (m *RawMap) GetField(fieldName string, raw json.RawMessage) (any, bool) {
	if string(raw) == "null" {
		return nil, true
	}
	rawBytes := []byte(raw)
	ctx := m.ctx
	ctx.fieldHint = extractFieldHint(fieldName)
	var val any
	if err := deserializeWithContext(rawBytes, &val, ctx); err != nil {
		return nil, false
	}
	return val, true
}

// DecodeField deserializes raw into dest, applying the field hint for fieldName.
// Use this when you've already navigated to the raw bytes and know the field name.
func (m *RawMap) DecodeField(fieldName string, raw json.RawMessage, dest any) error {
	rawBytes := []byte(raw)
	ctx := m.ctx
	ctx.fieldHint = extractFieldHint(fieldName)
	return deserializeWithContext(rawBytes, dest, ctx)
}
