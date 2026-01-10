/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package datachannel

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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
	return mustReadStdinFrom(os.Stdin, os.Stdout)
}

func mustReadStdinFrom(reader io.Reader, writer io.Writer) string {
	fmt.Fprintln(writer, "Paste your SDP offer (end with double newline):")
	sdpOffer, err := readUntilDoubleNewline(reader)
	if err != nil {
		fmt.Fprintln(writer, "Error:", err)
		return ""
	}
	fmt.Fprintln(writer, "Received SDP Offer:")
	fmt.Fprintln(writer, sdpOffer)
	return sdpOffer
}

func readUntilDoubleNewline(reader io.Reader) (string, error) {
	buf := bufio.NewReader(reader)
	var builder strings.Builder
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				builder.WriteString(line)
				return strings.TrimSuffix(builder.String(), "\n"), nil
			}
			return "", err
		}
		if line == "\n" || line == "\r\n" {
			return strings.TrimSuffix(builder.String(), "\n"), nil
		}
		builder.WriteString(line)
	}
}
