/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */

// Package datachannel provides WebRTC data channel utilities for signal exchange and file transfer.
package datachannel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Encode encodes a Signal to a compact binary format (74% smaller than JSON).
// For other types, falls back to JSON encoding.
func Encode(obj interface{}) string {
	// Use compact encoding for Signal type
	if signal, ok := obj.(Signal); ok {
		encoded, err := EncodeCompact(signal)
		if err != nil {
			logrus.Debugf("Compact encoding failed, falling back to JSON: %v", err)
			// Fall back to JSON on error
			b, err := json.Marshal(obj)
			cobra.CheckErr(err)
			return base64.StdEncoding.EncodeToString(b)
		}
		logrus.Debugf("Encoded compact signal: %d chars", len(encoded))
		return encoded
	}

	// For non-Signal types, use JSON encoding
	b, err := json.Marshal(obj)
	logrus.Debugf("%#v\n", string(b))
	cobra.CheckErr(err)
	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes a base64-encoded signal (compact or JSON format).
// Automatically detects the format based on the prefix.
func Decode(in string, obj interface{}) {
	// Clean input (remove whitespace)
	in = strings.TrimSpace(in)

	// Auto-detect format for Signal type
	if signal, ok := obj.(*Signal); ok {
		// Compact format starts with 'H' (base64: "SA")
		// JSON format starts with '{' (base64: "ey")
		if IsCompactFormat(in) {
			logrus.Debug("Detected compact signal format")
			err := DecodeCompact(in, signal)
			cobra.CheckErr(err)
			return
		}
		logrus.Debug("Detected legacy JSON signal format")
	}

	// Fall back to JSON decoding
	b, err := base64.StdEncoding.DecodeString(in)
	cobra.CheckErr(err)

	logrus.Debugf("%#v\n", string(b))
	err = json.Unmarshal(b, obj)
	cobra.CheckErr(err)
}

// MustReadStdin waiting for base64 encoded SDP for connection
func MustReadStdin() string {
	var sdpOffer string
	prompt := &survey.Multiline{
		Message: "Paste your SDP offer (end with Ctrl+D):",
	}
	err := survey.AskOne(prompt, &sdpOffer)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	fmt.Println("Received SDP Offer:")
	fmt.Println(sdpOffer)
	return sdpOffer
}
