package ptr

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Interface .
func Interface(o interface{}) *interface{} { return &o }

// String .
func String(o string) *string { return &o }

// Bool .
func Bool(o bool) *bool { return &o }

// Time .
func Time(o time.Time) *time.Time { return &o }

// UUID .
func UUID(o uuid.UUID) *uuid.UUID { return &o }

// Int .
func Int(o int) *int { return &o }

// Int8 .
func Int8(o int8) *int8 { return &o }

// Int16 .
func Int16(o int16) *int16 { return &o }

// Int32 .
func Int32(o int32) *int32 { return &o }

// Int64 .
func Int64(o int64) *int64 { return &o }

// Uint .
func Uint(o uint) *uint { return &o }

// Uint8 .
func Uint8(o uint8) *uint8 { return &o }

// Uint16 .
func Uint16(o uint16) *uint16 { return &o }

// Uint32 .
func Uint32(o uint32) *uint32 { return &o }

// Uint64 .
func Uint64(o uint64) *uint64 { return &o }

// Float32 .
func Float32(o float32) *float32 { return &o }

// Float64 .
func Float64(o float64) *float64 { return &o }
