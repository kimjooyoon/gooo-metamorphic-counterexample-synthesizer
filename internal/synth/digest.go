package synth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestValue(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(encoded), nil
}

func unsignedIRDigest(ir SemanticIR) (string, error) {
	ir.IRDigest = ""
	return DigestValue(ir)
}

func unsignedCandidateDigest(event CandidateEvent) (string, error) {
	event.Digest = ""
	event.Counterexample = nil
	return DigestValue(event)
}

func unsignedEvidenceDigest(evidence CandidateEvidence) (string, error) {
	evidence.Digest = ""
	return DigestValue(evidence)
}
