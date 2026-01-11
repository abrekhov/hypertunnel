/*
 *   Copyright (c) 2026 Anton Brekhov
 *   All rights reserved.
 */

// Package datachannel provides WebRTC data channel utilities for HyperTunnel.
package datachannel

import (
	"strings"

	"github.com/pion/webrtc/v3"
)

// DetermineICERole determines the ICE role based on ICE parameters.
// This implements symmetric role negotiation by comparing UsernameFragments.
// The peer with the lexicographically greater ufrag becomes Controlling.
// This allows peers to start in any order without pre-determined roles.
func DetermineICERole(localParams, remoteParams webrtc.ICEParameters) webrtc.ICERole {
	// Compare username fragments lexicographically
	comparison := strings.Compare(localParams.UsernameFragment, remoteParams.UsernameFragment)

	if comparison > 0 {
		// Local ufrag is greater, become controlling
		return webrtc.ICERoleControlling
	}
	// Local ufrag is less than or equal, become controlled
	return webrtc.ICERoleControlled
}

// DetermineICERoleWithDTLS determines the ICE role with DTLS fallback.
// If ICE parameters are identical (rare), it uses DTLS fingerprint comparison.
// This ensures deterministic role assignment even in edge cases.
func DetermineICERoleWithDTLS(
	localParams, remoteParams webrtc.ICEParameters,
	localDTLS, remoteDTLS webrtc.DTLSParameters,
) webrtc.ICERole {
	// First try ICE ufrag comparison
	comparison := strings.Compare(localParams.UsernameFragment, remoteParams.UsernameFragment)

	if comparison != 0 {
		// Ufrags are different, use standard comparison
		if comparison > 0 {
			return webrtc.ICERoleControlling
		}
		return webrtc.ICERoleControlled
	}

	// Fallback: Compare DTLS fingerprints (very rare case)
	if len(localDTLS.Fingerprints) > 0 && len(remoteDTLS.Fingerprints) > 0 {
		localFingerprint := localDTLS.Fingerprints[0].Value
		remoteFingerprint := remoteDTLS.Fingerprints[0].Value

		fingerprintComparison := strings.Compare(localFingerprint, remoteFingerprint)
		if fingerprintComparison > 0 {
			return webrtc.ICERoleControlling
		}
		return webrtc.ICERoleControlled
	}

	// Ultimate fallback (should never happen in practice)
	// Default to controlled to avoid both being controlling
	return webrtc.ICERoleControlled
}
