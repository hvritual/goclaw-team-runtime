package workstation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	deviceKeyBytes             = 32
	evidenceSignatureAlgorithm = "hmac-sha256"
	MaxEvidenceDiffPatchBytes  = 10 * 1024 * 1024
)

func GenerateDeviceKey() ([]byte, error) {
	return randomBytes(deviceKeyBytes)
}

func DeviceKeyID(deviceKey []byte) (string, error) {
	if len(deviceKey) < deviceKeyBytes {
		return "", fmt.Errorf("device key must be at least %d bytes", deviceKeyBytes)
	}
	sum := sha256.Sum256(deviceKey)
	return hex.EncodeToString(sum[:]), nil
}

func HashExecutionPack(pack ExecutionPack) (string, error) {
	data, err := json.Marshal(pack)
	if err != nil {
		return "", err
	}
	return sha256Bytes(data), nil
}

// SignEvidenceBundle returns a signed copy and never mutates the caller's
// value. The device key is not embedded in the bundle.
func SignEvidenceBundle(bundle EvidenceBundle, deviceKey []byte) (EvidenceBundle, error) {
	keyID, err := DeviceKeyID(deviceKey)
	if err != nil {
		return EvidenceBundle{}, err
	}
	bundle.SchemaVersion = SchemaVersion
	bundle.KeyID = keyID
	bundle.SignatureAlgorithm = evidenceSignatureAlgorithm
	bundle.BundleSHA256 = ""
	bundle.Signature = ""
	canonical, err := json.Marshal(bundle)
	if err != nil {
		return EvidenceBundle{}, err
	}
	bundle.BundleSHA256 = sha256Bytes(canonical)
	mac := hmac.New(sha256.New, deviceKey)
	_, _ = mac.Write(canonical)
	bundle.Signature = hex.EncodeToString(mac.Sum(nil))
	return bundle, nil
}

func VerifyEvidenceBundle(bundle EvidenceBundle, deviceKey []byte) error {
	keyID, err := DeviceKeyID(deviceKey)
	if err != nil {
		return err
	}
	if bundle.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %d", ErrInvalidEvidence, bundle.SchemaVersion)
	}
	if bundle.SignatureAlgorithm != evidenceSignatureAlgorithm {
		return fmt.Errorf("%w: unsupported signature algorithm %q", ErrInvalidSignature, bundle.SignatureAlgorithm)
	}
	if !hmac.Equal([]byte(strings.ToLower(bundle.KeyID)), []byte(keyID)) {
		return fmt.Errorf("%w: key id does not match runner credential", ErrInvalidSignature)
	}
	suppliedDigest := strings.ToLower(strings.TrimSpace(bundle.BundleSHA256))
	suppliedSignature := strings.ToLower(strings.TrimSpace(bundle.Signature))
	if suppliedDigest == "" || suppliedSignature == "" {
		return fmt.Errorf("%w: digest and signature are required", ErrInvalidSignature)
	}
	unsigned := bundle
	unsigned.BundleSHA256 = ""
	unsigned.Signature = ""
	canonical, err := json.Marshal(unsigned)
	if err != nil {
		return err
	}
	expectedDigest := sha256Bytes(canonical)
	if !hmac.Equal([]byte(suppliedDigest), []byte(expectedDigest)) {
		return fmt.Errorf("%w: bundle digest mismatch", ErrInvalidSignature)
	}
	suppliedMAC, err := hex.DecodeString(suppliedSignature)
	if err != nil {
		return fmt.Errorf("%w: malformed signature", ErrInvalidSignature)
	}
	mac := hmac.New(sha256.New, deviceKey)
	_, _ = mac.Write(canonical)
	if !hmac.Equal(suppliedMAC, mac.Sum(nil)) {
		return fmt.Errorf("%w: HMAC mismatch", ErrInvalidSignature)
	}
	return nil
}

func validateEvidenceIdentity(task Task, runnerID, leaseID string, bundle EvidenceBundle) error {
	switch {
	case bundle.TaskID != task.ID:
		return fmt.Errorf("%w: evidence task id does not match", ErrInvalidEvidence)
	case bundle.ProjectID != task.ProjectID:
		return fmt.Errorf("%w: evidence project id does not match", ErrInvalidEvidence)
	case bundle.ExecutionPackSHA256 != task.ExecutionPackSHA256:
		return fmt.Errorf("%w: execution pack digest does not match", ErrInvalidEvidence)
	case bundle.RunnerID != runnerID:
		return fmt.Errorf("%w: evidence runner id does not match", ErrInvalidEvidence)
	case bundle.LeaseID != leaseID:
		return fmt.Errorf("%w: evidence lease id does not match", ErrInvalidEvidence)
	case bundle.Attempt != task.Attempt:
		return fmt.Errorf("%w: evidence attempt does not match", ErrInvalidEvidence)
	case bundle.BaseCommit != task.ExecutionPack.BaseCommit:
		return fmt.Errorf("%w: evidence base commit does not match", ErrInvalidEvidence)
	case bundle.StartedAt.IsZero() || bundle.FinishedAt.IsZero():
		return fmt.Errorf("%w: evidence timestamps are required", ErrInvalidEvidence)
	case bundle.FinishedAt.Before(bundle.StartedAt):
		return fmt.Errorf("%w: evidence finished_at precedes started_at", ErrInvalidEvidence)
	}
	if bundle.Outcome != "completed" && bundle.Outcome != "failed" {
		return fmt.Errorf("%w: outcome must be completed or failed", ErrInvalidEvidence)
	}
	for _, artifact := range bundle.Artifacts {
		if strings.TrimSpace(artifact.Name) == "" || len(strings.TrimSpace(artifact.SHA256)) != sha256.Size*2 {
			return fmt.Errorf("%w: artifact name and SHA-256 are required", ErrInvalidEvidence)
		}
		if _, err := hex.DecodeString(artifact.SHA256); err != nil {
			return fmt.Errorf("%w: artifact %q has malformed SHA-256", ErrInvalidEvidence, artifact.Name)
		}
	}
	if bundle.DiffSHA256 != "" {
		if len(bundle.DiffSHA256) != sha256.Size*2 {
			return fmt.Errorf("%w: malformed diff SHA-256", ErrInvalidEvidence)
		}
		if _, err := hex.DecodeString(bundle.DiffSHA256); err != nil {
			return fmt.Errorf("%w: malformed diff SHA-256", ErrInvalidEvidence)
		}
	}
	if len(bundle.DiffPatch) > MaxEvidenceDiffPatchBytes {
		return fmt.Errorf(
			"%w: diff patch exceeds %d bytes",
			ErrInvalidEvidence,
			MaxEvidenceDiffPatchBytes,
		)
	}
	if bundle.DiffPatch != "" {
		if bundle.DiffSHA256 == "" {
			return fmt.Errorf("%w: diff SHA-256 is required with diff patch", ErrInvalidEvidence)
		}
		if sha256Bytes([]byte(bundle.DiffPatch)) != strings.ToLower(bundle.DiffSHA256) {
			return fmt.Errorf("%w: diff patch SHA-256 mismatch", ErrInvalidEvidence)
		}
	} else if bundle.Outcome == "completed" && len(bundle.ChangedFiles) > 0 {
		return fmt.Errorf("%w: completed changes require a recoverable diff patch", ErrInvalidEvidence)
	}
	return nil
}

func validateCompletionEvidence(task Task, bundle EvidenceBundle) error {
	if bundle.HeadCommit != task.ExecutionPack.BaseCommit {
		return fmt.Errorf(
			"%w: completed evidence must retain the frozen base commit as HEAD",
			ErrInvalidEvidence,
		)
	}
	if strings.TrimSpace(bundle.CommitSHA) != "" {
		return fmt.Errorf(
			"%w: completed evidence must not contain an automatic commit",
			ErrInvalidEvidence,
		)
	}

	expected := map[string]int{
		"runner-setup":        1,
		"codex-exec":          1,
		"scope-policy":        1,
		"no-automatic-commit": 1,
	}
	for _, command := range task.ExecutionPack.Verification {
		name := valueOr(command.Name, strings.Join(command.Argv, " "))
		expected[name]++
	}
	seen := make(map[string]int, len(bundle.Checks))
	for _, check := range bundle.Checks {
		name := strings.TrimSpace(check.Name)
		if name == "" {
			return fmt.Errorf("%w: evidence check name is required", ErrInvalidEvidence)
		}
		if !check.Passed || check.ExitCode != 0 {
			return fmt.Errorf(
				"%w: evidence check %q did not pass",
				ErrInvalidEvidence,
				name,
			)
		}
		seen[name]++
	}
	for name, count := range expected {
		if seen[name] != count {
			return fmt.Errorf(
				"%w: evidence requires %d passing %q check(s), got %d",
				ErrInvalidEvidence,
				count,
				name,
				seen[name],
			)
		}
	}

	changed := make(map[string]struct{}, len(bundle.ChangedFiles))
	for _, path := range bundle.ChangedFiles {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if clean == "" || clean == "." || clean == ".." ||
			strings.HasPrefix(clean, "../") || filepath.IsAbs(path) {
			return fmt.Errorf(
				"%w: invalid changed path %q",
				ErrInvalidEvidence,
				path,
			)
		}
		if _, duplicate := changed[clean]; duplicate {
			return fmt.Errorf(
				"%w: duplicate changed path %q",
				ErrInvalidEvidence,
				clean,
			)
		}
		changed[clean] = struct{}{}
		if localMatchesAnyPath(clean, task.ExecutionPack.DeniedPaths) {
			return fmt.Errorf(
				"%w: denied path changed: %s",
				ErrInvalidEvidence,
				clean,
			)
		}
		if len(task.ExecutionPack.AllowedPaths) > 0 &&
			!localMatchesAnyPath(clean, task.ExecutionPack.AllowedPaths) {
			return fmt.Errorf(
				"%w: path outside approved scope: %s",
				ErrInvalidEvidence,
				clean,
			)
		}
	}
	return nil
}

func isInvalidSignature(err error) bool {
	return errors.Is(err, ErrInvalidSignature)
}
