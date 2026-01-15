/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package datachannel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Encode base64 SDP
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	logrus.Debugf("%#v\n", string(b))
	cobra.CheckErr(err)
	return base64.StdEncoding.EncodeToString(b)
}

// Decode base64 SDP
func Decode(in string, obj interface{}) {
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
