package brain

import (
	"encoding/json"
	"testing"
)

const testMinConfidence = 0.7

func TestValidateEnvelope_ValidContact(t *testing.T) {
	env := &Envelope{
		RecordType:    "crm.contact",
		SchemaVersion: "1.0.0",
		ContentText:   "Added contact Ada Lovelace",
		Confidence:    0.92,
		Payload:       json.RawMessage(`{"name":"Ada Lovelace","company":"Analytical Engines"}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if !result.Valid {
		t.Errorf("expected valid, got failure_mode=%q error=%q", result.FailureMode, result.ErrorMessage)
	}
}

func TestValidateEnvelope_ValidThought(t *testing.T) {
	env := &Envelope{
		RecordType:    "note.thought",
		SchemaVersion: "1.0.0",
		ContentText:   "Some thought",
		Confidence:    0.85,
		Payload:       json.RawMessage(`{"content":"What is the meaning of life?"}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if !result.Valid {
		t.Errorf("expected valid, got failure_mode=%q error=%q", result.FailureMode, result.ErrorMessage)
	}
}

func TestValidateEnvelope_LowConfidence(t *testing.T) {
	env := &Envelope{
		RecordType:    "crm.contact",
		SchemaVersion: "1.0.0",
		ContentText:   "Some text",
		Confidence:    0.5,
		Payload:       json.RawMessage(`{"name":"Bob"}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if result.Valid {
		t.Error("expected invalid due to low confidence")
	}
	if result.FailureMode != "low_confidence" {
		t.Errorf("expected failure_mode=low_confidence, got %q", result.FailureMode)
	}
}

func TestValidateEnvelope_MissingRequiredField(t *testing.T) {
	// crm.contact requires "name"
	env := &Envelope{
		RecordType:    "crm.contact",
		SchemaVersion: "1.0.0",
		ContentText:   "Some contact",
		Confidence:    0.9,
		Payload:       json.RawMessage(`{"company":"Acme"}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if result.Valid {
		t.Error("expected invalid due to missing required field")
	}
	if result.FailureMode != "validation_failure" {
		t.Errorf("expected failure_mode=validation_failure, got %q", result.FailureMode)
	}
}

func TestValidateEnvelope_AdditionalProperties(t *testing.T) {
	// crm.contact has additionalProperties:false
	env := &Envelope{
		RecordType:    "crm.contact",
		SchemaVersion: "1.0.0",
		ContentText:   "Some contact",
		Confidence:    0.9,
		Payload:       json.RawMessage(`{"name":"Bob","unknown_field":"oops"}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if result.Valid {
		t.Error("expected invalid due to additionalProperties violation")
	}
	if result.FailureMode != "validation_failure" {
		t.Errorf("expected failure_mode=validation_failure, got %q", result.FailureMode)
	}
}

func TestValidateEnvelope_UnknownRecordType(t *testing.T) {
	env := &Envelope{
		RecordType:    "unknown.type",
		SchemaVersion: "1.0.0",
		ContentText:   "Some text",
		Confidence:    0.9,
		Payload:       json.RawMessage(`{}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if result.Valid {
		t.Error("expected invalid for unknown record type")
	}
	if result.FailureMode != "validation_failure" {
		t.Errorf("expected failure_mode=validation_failure, got %q", result.FailureMode)
	}
}

func TestValidateEnvelope_MissingContentText(t *testing.T) {
	env := &Envelope{
		RecordType:    "crm.contact",
		SchemaVersion: "1.0.0",
		ContentText:   "",
		Confidence:    0.9,
		Payload:       json.RawMessage(`{"name":"Bob"}`),
	}
	result := ValidateEnvelope(env, testMinConfidence)
	if result.Valid {
		t.Error("expected invalid due to missing content_text")
	}
}

func TestFallbackPayload_WithEnvelope(t *testing.T) {
	env := &Envelope{RecordType: "crm.contact"}
	result := ValidationResult{FailureMode: "low_confidence", ErrorMessage: "confidence 0.50 below threshold 0.70"}

	raw, err := FallbackPayload("original text", env, result)
	if err != nil {
		t.Fatalf("FallbackPayload: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal fallback: %v", err)
	}
	if m["content"] != "original text" {
		t.Errorf("expected content=original text, got %v", m["content"])
	}
	if m["attempted_record_type"] != "crm.contact" {
		t.Errorf("expected attempted_record_type=crm.contact, got %v", m["attempted_record_type"])
	}
	if m["failure_mode"] != "low_confidence" {
		t.Errorf("expected failure_mode=low_confidence, got %v", m["failure_mode"])
	}
}

func TestFallbackPayload_NilEnvelope(t *testing.T) {
	result := ValidationResult{FailureMode: "validation_failure", ErrorMessage: "bad JSON"}
	raw, err := FallbackPayload("raw text", nil, result)
	if err != nil {
		t.Fatalf("FallbackPayload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["content"] != "raw text" {
		t.Errorf("expected content=raw text, got %v", m["content"])
	}
}
