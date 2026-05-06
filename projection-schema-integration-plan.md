# Projection Schema Integration Plan

## Overview

This document outlines the design for integrating `projectionSchema` type hints into the serdes deserialization process for untyped table rows. The schema will be used to correctly deserialize values when explicitly decoding into `table.Row` type.

## Problem Statement

When deserializing table rows into untyped structures (e.g., `map[string]any`), the current serdes implementation uses heuristics to determine types:
- Numbers are decoded as `float64` by default
- No distinction between `int`, `bigint`, `varint`, etc.
- Cannot differentiate between text types (`text`, `ascii`)
- Complex types (UDTs, typed collections) lose type information

The Data API returns a `projectionSchema` in the response status that describes the exact column types. We can use this as a "type hint" to deserialize values correctly.

## Design

### 1. Row Type

```go
// In table package (table/row.go)
package table

import (
    "encoding/json"
    "fmt"
    "math/big"
    "net"
    "github.com/datastax/astra-db-go/datatypes"
    "github.com/datastax/astra-db-go/serdes"
)

// Row represents an untyped table row with schema information.
// The Data field contains the row values, and Schema provides type hints
// for correct deserialization.
type Row struct {
    Data   map[string]any
    Schema Columns  // Set before deserialization
}

// UnmarshalAstraRaw implements custom deserialization using schema type hints.
func (r *Row) UnmarshalAstraRaw(target serdes.Target, value []byte) error {
    // Step 1: Parse as map[string]RawMessage to get raw JSON for each field
    rawMap := make(map[string]json.RawMessage)
    if err := serdes.Deserialize(value, &rawMap, target); err != nil {
        return err
    }
    
    // Step 2: Deserialize each field using schema (iterate schema, lookup in rawMap)
    r.Data = make(map[string]any, len(r.Schema))
    for _, nc := range r.Schema {
        rawValue, ok := rawMap[nc.Name]
        if !ok {
            // Field in schema but not in data - set to nil
            r.Data[nc.Name] = nil
            continue
        }
        
        // Check for JSON null
        if string(rawValue) == "null" {
            r.Data[nc.Name] = nil
            continue
        }
        
        // Deserialize with type hint (Column is passed recursively for nested types)
        val, err := deserializeWithTypeHint(rawValue, nc.Column, target)
        if err != nil {
            return fmt.Errorf("field %s: %w", nc.Name, err)
        }
        r.Data[nc.Name] = val
    }
    
    return nil
}
```

### 2. Type-Hint Deserialization (Recursive)

```go
// In table/row.go

// deserializeWithTypeHint deserializes a value using the column type hint.
// For nested types (UDTs, collections), the Column definition is passed recursively.
func deserializeWithTypeHint(raw json.RawMessage, col Column, target serdes.Target) (any, error) {
    switch col.Type {
    // Numeric types
    case TypeInt:
        var v int
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeBigInt:
        var v int64
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeSmallInt:
        var v int16
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeTinyInt:
        var v int8
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeFloat:
        var v float32
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeDouble:
        var v float64
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeVarint:
        var v big.Int
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeDecimal:
        var v big.Float
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // String types
    case TypeText, TypeAscii:
        var v string
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // Boolean
    case TypeBoolean:
        var v bool
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // Date/Time types
    case TypeDate, TypeTime, TypeTimestamp:
        var v datatypes.DataAPITimestamp
        err := serdes.Deserialize(raw, &v, target)
        return v, err
        
    case TypeDuration:
        // Use string representation for now
        var v string
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // UUID types
    case TypeUUID, TypeTimeUUID:
        var v datatypes.UUID
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // Binary
    case TypeBlob:
        var v []byte
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // Network
    case TypeInet:
        var v net.IP
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // Vector
    case TypeVector:
        var v datatypes.DataAPIVector
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    
    // Collection types - pass Column recursively for element types
    case TypeSet:
        return deserializeSet(raw, col, target)
        
    case TypeList:
        return deserializeList(raw, col, target)
        
    case TypeMap:
        return deserializeMap(raw, col, target)
    
    // User-defined type - pass Column recursively for field types
    case TypeUDT:
        return deserializeUDT(raw, col, target)
    
    default:
        // Unknown type - fallback to generic deserialization
        var v any
        err := serdes.Deserialize(raw, &v, target)
        return v, err
    }
}

// deserializeSet deserializes a set with typed elements.
// The element type is obtained from col.ValueType and passed recursively.
func deserializeSet(raw json.RawMessage, col Column, target serdes.Target) (any, error) {
    if col.ValueType == nil {
        return nil, fmt.Errorf("set column missing valueType")
    }
    
    // Parse as array first
    var rawArray []json.RawMessage
    if err := serdes.Deserialize(raw, &rawArray, target); err != nil {
        return nil, err
    }
    
    // Deserialize each element with type hint (recursive)
    result := make([]any, len(rawArray))
    for i, rawElem := range rawArray {
        val, err := deserializeWithTypeHint(rawElem, *col.ValueType, target)
        if err != nil {
            return nil, fmt.Errorf("set element %d: %w", i, err)
        }
        result[i] = val
    }
    
    // Return as []any for now
    return result, nil
}

// deserializeList deserializes a list with typed elements.
// The element type is obtained from col.ValueType and passed recursively.
func deserializeList(raw json.RawMessage, col Column, target serdes.Target) (any, error) {
    if col.ValueType == nil {
        return nil, fmt.Errorf("list column missing valueType")
    }
    
    // Parse as array first
    var rawArray []json.RawMessage
    if err := serdes.Deserialize(raw, &rawArray, target); err != nil {
        return nil, err
    }
    
    // Deserialize each element with type hint (recursive)
    result := make([]any, len(rawArray))
    for i, rawElem := range rawArray {
        val, err := deserializeWithTypeHint(rawElem, *col.ValueType, target)
        if err != nil {
            return nil, fmt.Errorf("list element %d: %w", i, err)
        }
        result[i] = val
    }
    
    return result, nil
}

// deserializeMap deserializes a map with typed keys and values.
// Key and value types are obtained from col.KeyType and col.ValueType and passed recursively.
func deserializeMap(raw json.RawMessage, col Column, target serdes.Target) (any, error) {
    if col.KeyType == nil || col.ValueType == nil {
        return nil, fmt.Errorf("map column missing keyType or valueType")
    }
    
    // Parse as map[string]RawMessage first (JSON keys are always strings)
    var rawMap map[string]json.RawMessage
    if err := serdes.Deserialize(raw, &rawMap, target); err != nil {
        return nil, err
    }
    
    // Deserialize each key-value pair with type hints (recursive)
    result := make(map[any]any, len(rawMap))
    keyCol := Column{Type: *col.KeyType}
    
    for rawKeyStr, rawValue := range rawMap {
        // Deserialize key: convert string key to proper type
        // JSON keys are always strings, so we need to deserialize the string representation
        keyBytes := json.RawMessage(`"` + rawKeyStr + `"`)
        key, err := deserializeWithTypeHint(keyBytes, keyCol, target)
        if err != nil {
            return nil, fmt.Errorf("map key %s: %w", rawKeyStr, err)
        }
        
        // Deserialize value with type hint (recursive)
        val, err := deserializeWithTypeHint(rawValue, *col.ValueType, target)
        if err != nil {
            return nil, fmt.Errorf("map value for key %s: %w", rawKeyStr, err)
        }
        
        result[key] = val
    }
    
    return result, nil
}

// deserializeUDT deserializes a user-defined type.
// UDT field definitions are obtained from col.Definition() and passed recursively.
func deserializeUDT(raw json.RawMessage, col Column, target serdes.Target) (any, error) {
    // Parse as map[string]RawMessage
    var rawMap map[string]json.RawMessage
    if err := serdes.Deserialize(raw, &rawMap, target); err != nil {
        return nil, err
    }
    
    result := make(map[string]any, len(rawMap))
    
    // Get UDT field definitions from Column (via getter)
    udtDef := col.Definition()
    if udtDef == nil {
        // No definition available - fallback to generic deserialization
        for key, rawValue := range rawMap {
            if string(rawValue) == "null" {
                result[key] = nil
                continue
            }
            
            var val any
            if err := serdes.Deserialize(rawValue, &val, target); err != nil {
                return nil, fmt.Errorf("UDT field %s: %w", key, err)
            }
            result[key] = val
        }
        return result, nil
    }
    
    // Deserialize each field with type hint from UDT definition
    for key, rawValue := range rawMap {
        if string(rawValue) == "null" {
            result[key] = nil
            continue
        }
        
        fieldCol, ok := udtDef.Fields[key]
        if !ok {
            // Field not in definition - fallback to generic deserialization
            var val any
            if err := serdes.Deserialize(rawValue, &val, target); err != nil {
                return nil, fmt.Errorf("UDT field %s: %w", key, err)
            }
            result[key] = val
            continue
        }
        
        // Deserialize with type hint (recursive for nested UDTs/collections)
        val, err := deserializeWithTypeHint(rawValue, fieldCol, target)
        if err != nil {
            return nil, fmt.Errorf("UDT field %s: %w", key, err)
        }
        result[key] = val
    }
    
    return result, nil
}
```

### 3. Usage Example

```go
// Direct usage with serdes package
rawJSON := []byte(`{"title": "Some Book", "rating": 5, "pages": 300}`)

// Create Row with schema
row := &table.Row{
    Schema: table.Columns{
        {Name: "title", Column: table.Column{Type: "text"}},
        {Name: "rating", Column: table.Column{Type: "int"}},
        {Name: "pages", Column: table.Column{Type: "int"}},
    },
}

// Deserialize using schema type hints
target := serdes.NewTarget()
if err := serdes.Deserialize(rawJSON, row, target); err != nil {
    return err
}

// Access typed values
title := row.Data["title"].(string)
rating := row.Data["rating"].(int)  // Correctly typed as int, not float64
pages := row.Data["pages"].(int)    // Correctly typed as int, not float64
```

## Performance Considerations

### Zero-Cost for Typed Operations

The Row type and schema-aware deserialization **only** affects untyped row deserialization. Fully typed struct deserialization is unaffected:

```go
type Book struct {
    Title  string  `json:"title"`
    Rating int     `json:"rating"`
}

var book Book
result.Decode(&book)  // No Row involved, no schema lookup, no overhead
```

### Row Deserialization Overhead

When using Row:
1. **First parse**: `map[string]json.RawMessage` - one allocation per field
2. **Schema lookup**: O(1) map lookup per top-level field
3. **Second parse**: Type-specific deserialization per field
4. **Recursive overhead**: For nested types (UDTs, collections), Column is passed recursively

### No Flattening Overhead

Unlike a flattening approach, this design:
- Only builds a simple map for top-level columns
- Passes Column definitions recursively for nested types
- No complex path tracking or pre-computation
- Natural handling of collections with UDTs

## Implementation Notes

### Null Value Handling
- Missing fields in data: explicitly set to `nil` in `row.Data`
- JSON `null` values: explicitly set to `nil` in `row.Data`
- This provides consistent behavior and allows distinguishing between missing and null

### Type Constants
- References to `TypeInt`, `TypeBigInt`, etc. assume these constants are defined elsewhere in the codebase
- These should be available in the `table` package or imported from appropriate location

### Recursive Definition Population
- The `Column.UnmarshalAstraRaw` method handles recursive deserialization correctly
- When deserializing nested Columns (in `ValueType` or `UDT.Definition.Fields`), each nested Column's `UnmarshalAstraRaw` is called
- This ensures all nested `definition` fields are properly populated throughout the structure

## Implementation Phases

### Phase 1: Basic Row Type (MVP)
- [ ] Add `definition` field to `Column` type in table/definition.go
- [ ] Implement `Column.Definition()` getter method
- [ ] Implement `Column.UnmarshalAstraRaw` to populate unexported definition field
- [ ] Implement `table.Row` type in table/row.go
- [ ] Implement `Row.UnmarshalAstraRaw` with top-level field support and null handling
- [ ] Implement `deserializeWithTypeHint` for scalar types
- [ ] Write tests for basic scalar types and null values

### Phase 2: Collection Types
- [ ] Implement `deserializeSet` with recursive type hints
- [ ] Implement `deserializeList` with recursive type hints
- [ ] Implement `deserializeMap` with recursive type hints and proper key handling
- [ ] Write tests for collection types with nested UDTs

### Phase 3: UDT Support
- [ ] Implement `UDTDefinition` type
- [ ] Implement `deserializeUDT` with recursive field type hints
- [ ] Write tests for UDT types
- [ ] Write tests for deeply nested structures (UDT in list in map, etc.)

### Phase 4: Integration & Testing
- [ ] Integration with existing result/cursor types
- [ ] Performance benchmarks
- [ ] Documentation and examples

## Testing Strategy

1. **Unit tests** for `deserializeWithTypeHint` with all column types
2. **Null handling tests** for missing fields and JSON null values
3. **Comparison tests** showing correct typing (int vs float64, etc.)
4. **Nested structure tests** for lists, maps, and UDTs
5. **Deep nesting tests** (e.g., `list<map<text, udt>>`)
6. **Recursive definition tests** verifying nested Column definitions are populated
