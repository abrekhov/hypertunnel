/*
 *   Copyright (c) 2021 Anton Brekhov anton.brekhov@rsc-tech.ru
 *   All rights reserved.
 */

// Package hashutils provides cryptographic hashing utilities.
package hashutils

import (
	"crypto/sha256"

	"github.com/sirupsen/logrus"
)

// FromKeyToAESKey any pwd to 16byte string as hash
func FromKeyToAESKey(userkey string) []byte {
	h := sha256.New()
	wrtn, err := h.Write([]byte(userkey))
	logrus.Debugf("written: %#v\n", wrtn)
	if err != nil {
		logrus.Fatalln(err)
	}
	return h.Sum(nil)
}
