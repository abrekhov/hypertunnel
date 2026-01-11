/*
 *   Copyright (c) 2026 Anton Brekhov
 *   All rights reserved.
 */

package datachannel

import (
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

// TestDetermineICERole tests the ICE role determination logic
func TestDetermineICERole(t *testing.T) {
	testCases := []struct {
		name           string
		localUfrag     string
		remoteUfrag    string
		expectedRole   webrtc.ICERole
		expectedRemote webrtc.ICERole
	}{
		{
			name:           "local ufrag greater than remote",
			localUfrag:     "zebra",
			remoteUfrag:    "apple",
			expectedRole:   webrtc.ICERoleControlling,
			expectedRemote: webrtc.ICERoleControlled,
		},
		{
			name:           "local ufrag less than remote",
			localUfrag:     "apple",
			remoteUfrag:    "zebra",
			expectedRole:   webrtc.ICERoleControlled,
			expectedRemote: webrtc.ICERoleControlling,
		},
		{
			name:           "equal ufrag defaults to controlled",
			localUfrag:     "same",
			remoteUfrag:    "same",
			expectedRole:   webrtc.ICERoleControlled, // Both equal, default to controlled
			expectedRemote: webrtc.ICERoleControlled,
		},
		{
			name:           "numeric ufrag comparison",
			localUfrag:     "user123",
			remoteUfrag:    "user456",
			expectedRole:   webrtc.ICERoleControlled,
			expectedRemote: webrtc.ICERoleControlling,
		},
		{
			name:           "case sensitivity check",
			localUfrag:     "User",
			remoteUfrag:    "user",
			expectedRole:   webrtc.ICERoleControlled, // 'U' < 'u' in ASCII
			expectedRemote: webrtc.ICERoleControlling,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localParams := webrtc.ICEParameters{
				UsernameFragment: tc.localUfrag,
				Password:         "localpass",
			}
			remoteParams := webrtc.ICEParameters{
				UsernameFragment: tc.remoteUfrag,
				Password:         "remotepass",
			}

			// Test local perspective
			role := DetermineICERole(localParams, remoteParams)
			assert.Equal(t, tc.expectedRole, role,
				"Local role should be %v when local=%s, remote=%s",
				tc.expectedRole, tc.localUfrag, tc.remoteUfrag)

			// Test symmetry: remote peer should get opposite role
			remoteRole := DetermineICERole(remoteParams, localParams)
			assert.Equal(t, tc.expectedRemote, remoteRole,
				"Remote role should be %v when swapping parameters",
				tc.expectedRemote)

			// Verify one is controlling and one is controlled (except when equal)
			if tc.localUfrag != tc.remoteUfrag {
				assert.NotEqual(t, role, remoteRole,
					"Roles must be different when ufrags are different")
			} else {
				// When ufrags are equal, both should be controlled
				assert.Equal(t, role, remoteRole,
					"Both should have same role when ufrags are equal")
			}
		})
	}
}

// TestDetermineICERoleWithDTLSFingerprint tests fallback to DTLS fingerprint comparison
func TestDetermineICERoleWithDTLSFingerprint(t *testing.T) {
	testCases := []struct {
		name                string
		localFingerprint    string
		remoteFingerprint   string
		expectedRole        webrtc.ICERole
		expectedRemoteRole  webrtc.ICERole
	}{
		{
			name:               "local fingerprint greater",
			localFingerprint:   "FF:FF:FF:FF:FF:FF",
			remoteFingerprint:  "00:00:00:00:00:00",
			expectedRole:       webrtc.ICERoleControlling,
			expectedRemoteRole: webrtc.ICERoleControlled,
		},
		{
			name:               "remote fingerprint greater",
			localFingerprint:   "00:00:00:00:00:00",
			remoteFingerprint:  "FF:FF:FF:FF:FF:FF",
			expectedRole:       webrtc.ICERoleControlled,
			expectedRemoteRole: webrtc.ICERoleControlling,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create parameters with same ufrag but different fingerprints
			localParams := webrtc.ICEParameters{
				UsernameFragment: "same-ufrag",
				Password:         "localpass",
			}
			remoteParams := webrtc.ICEParameters{
				UsernameFragment: "same-ufrag",
				Password:         "remotepass",
			}

			localDTLS := webrtc.DTLSParameters{
				Fingerprints: []webrtc.DTLSFingerprint{
					{
						Algorithm: "sha-256",
						Value:     tc.localFingerprint,
					},
				},
			}
			remoteDTLS := webrtc.DTLSParameters{
				Fingerprints: []webrtc.DTLSFingerprint{
					{
						Algorithm: "sha-256",
						Value:     tc.remoteFingerprint,
					},
				},
			}

			// Test with DTLS parameters
			role := DetermineICERoleWithDTLS(localParams, remoteParams, localDTLS, remoteDTLS)
			assert.Equal(t, tc.expectedRole, role)

			// Test symmetry
			remoteRole := DetermineICERoleWithDTLS(remoteParams, localParams, remoteDTLS, localDTLS)
			assert.Equal(t, tc.expectedRemoteRole, remoteRole)

			// Verify roles are opposite
			assert.NotEqual(t, role, remoteRole, "Roles must be different")
		})
	}
}

// TestDetermineICERoleDeterminism ensures both peers agree on roles
func TestDetermineICERoleDeterminism(t *testing.T) {
	// Create two sets of parameters simulating two peers
	peerAParams := webrtc.ICEParameters{
		UsernameFragment: "alice123",
		Password:         "alicepass",
	}
	peerBParams := webrtc.ICEParameters{
		UsernameFragment: "bob456",
		Password:         "bobpass",
	}

	// Peer A determines its role based on B's parameters
	roleA := DetermineICERole(peerAParams, peerBParams)

	// Peer B determines its role based on A's parameters
	roleB := DetermineICERole(peerBParams, peerAParams)

	// They must have opposite roles
	assert.NotEqual(t, roleA, roleB, "Peers must have different roles")

	// One must be controlling, one must be controlled
	if roleA == webrtc.ICERoleControlling {
		assert.Equal(t, webrtc.ICERoleControlled, roleB)
	} else {
		assert.Equal(t, webrtc.ICERoleControlling, roleB)
	}
}
