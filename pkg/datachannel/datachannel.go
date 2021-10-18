/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package datachannel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chzyer/readline"
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
	// GNU like readline used because of macOS terminal os.Stdin 1024 char limit
	rl, err := readline.New("Insert remote SDP: ")
	cobra.CheckErr(err)
	defer rl.Close()

	var in string
	line, err := rl.Readline()
	cobra.CheckErr(err)
	readline.Stdin.Close()
	fmt.Printf("\nSDP read.\n")
	in = line
	in = strings.TrimSpace(in)
	return in
}
