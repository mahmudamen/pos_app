package identity

import "context"

// Device integrity statuses stored on installations.device_integrity_status.
const (
	IntegrityVerified    = "verified"
	IntegrityUnavailable = "unavailable"
	IntegrityFailed      = "failed"
)

// DeviceIntegrityVerifier is the optional platform-integrity backend. It keeps
// app/device attestation (Play Integrity, Apple App Attest) behind an
// interface that configuration can turn on or off.
//
// Contract:
//   - the verifier never returns a client-supplied "device_is_valid" boolean;
//   - a token that cannot be verified yields IntegrityFailed/Unavailable, it
//     never fabricates a pass;
//   - the result is stored as data and only *contributes* to risk — attestation
//     is never the sole trial-protection mechanism and never blocks the POS by
//     default (require_device_integrity=false).
type DeviceIntegrityVerifier interface {
	// Verify checks an attestation assertion for a given installation and
	// returns the server-side integrity status.
	Verify(ctx context.Context, installationPublicID, assertion string) (string, error)
}

// NoopVerifier is the default: integrity attestation is unavailable. With
// require_device_integrity=false (the default) this keeps POS fully usable.
type NoopVerifier struct{}

func (NoopVerifier) Verify(context.Context, string, string) (string, error) {
	return IntegrityUnavailable, nil
}
