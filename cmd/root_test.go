/*
Copyright © 2021 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"testing"

	webrtc "github.com/pion/webrtc/v3"
)

func TestDecideICERoleLocalGreater(t *testing.T) {
	localICE := webrtc.ICEParameters{UsernameFragment: "z-local", Password: "pw-local"}
	remoteICE := webrtc.ICEParameters{UsernameFragment: "a-remote", Password: "pw-remote"}
	localDTLS := webrtc.DTLSParameters{Fingerprints: []webrtc.DTLSFingerprint{{Algorithm: "sha-256", Value: "AA:BB:CC"}}}
	remoteDTLS := webrtc.DTLSParameters{Fingerprints: []webrtc.DTLSFingerprint{{Algorithm: "sha-256", Value: "11:22:33"}}}

	role, _, _ := decideICERole(localICE, localDTLS, remoteICE, remoteDTLS)
	if role != webrtc.ICERoleControlling {
		t.Fatalf("expected controlling, got %s", role)
	}

	roleAgain, _, _ := decideICERole(localICE, localDTLS, remoteICE, remoteDTLS)
	if roleAgain != role {
		t.Fatalf("expected deterministic role %s, got %s", role, roleAgain)
	}

	remoteRole, _, _ := decideICERole(remoteICE, remoteDTLS, localICE, localDTLS)
	if remoteRole != webrtc.ICERoleControlled {
		t.Fatalf("expected symmetric controlled, got %s", remoteRole)
	}
}

func TestDecideICERoleLocalLess(t *testing.T) {
	localICE := webrtc.ICEParameters{UsernameFragment: "a-local", Password: "pw-local"}
	remoteICE := webrtc.ICEParameters{UsernameFragment: "z-remote", Password: "pw-remote"}
	localDTLS := webrtc.DTLSParameters{Fingerprints: []webrtc.DTLSFingerprint{{Algorithm: "sha-256", Value: "11:22:33"}}}
	remoteDTLS := webrtc.DTLSParameters{Fingerprints: []webrtc.DTLSFingerprint{{Algorithm: "sha-256", Value: "AA:BB:CC"}}}

	role, _, _ := decideICERole(localICE, localDTLS, remoteICE, remoteDTLS)
	if role != webrtc.ICERoleControlled {
		t.Fatalf("expected controlled, got %s", role)
	}

	remoteRole, _, _ := decideICERole(remoteICE, remoteDTLS, localICE, localDTLS)
	if remoteRole != webrtc.ICERoleControlling {
		t.Fatalf("expected symmetric controlling, got %s", remoteRole)
	}
}

func TestDecideICERoleEqualKeys(t *testing.T) {
	localICE := webrtc.ICEParameters{UsernameFragment: "same", Password: "pw"}
	remoteICE := webrtc.ICEParameters{UsernameFragment: "same", Password: "pw"}
	localDTLS := webrtc.DTLSParameters{Fingerprints: []webrtc.DTLSFingerprint{{Algorithm: "sha-256", Value: "AA:BB:CC"}}}
	remoteDTLS := webrtc.DTLSParameters{Fingerprints: []webrtc.DTLSFingerprint{{Algorithm: "sha-256", Value: "AA:BB:CC"}}}

	role, _, _ := decideICERole(localICE, localDTLS, remoteICE, remoteDTLS)
	if role != webrtc.ICERoleControlling {
		t.Fatalf("expected controlling on equal keys, got %s", role)
	}
}
