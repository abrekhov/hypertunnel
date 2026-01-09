/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"crypto/aes"
	"crypto/cipher"
	"io"
	"log"
	"os"

	"github.com/abrekhov/hypertunnel/pkg/hashutils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// decryptCmd represents the decrypt command
var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt some file with keyphrase",
	Run:   decryptFile,
}

func init() {
	rootCmd.AddCommand(decryptCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// decryptCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	decryptCmd.Flags().StringVarP(&keyphrase, "key", "k", "", "Keyphrase to decrypt file")
	decryptCmd.Flags().Int32VarP(&bufferSize, "buffer", "b", 1024, "Buffer size")
}

func decryptFile(cmd *cobra.Command, args []string) {
	// KEY Proccessing
	if keyphrase == "" {
		logrus.Fatalln("Keyphrase is empty!")
	}
	keyHash := hashutils.FromKeyToAESKey(keyphrase)
	logrus.Debugln("keyHash:", keyHash)

	// Input file
	filename := args[0]
	infile, err := os.Open(filename)
	if err != nil {
		logrus.Fatalln(err)
	}
	defer infile.Close()

	fi, err := infile.Stat()
	if err != nil {
		log.Fatal(err)
	}

	// Output file
	outfile, err := os.OpenFile(filename+".dec", os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		logrus.Fatal(err)
	}
	defer outfile.Close()

	// Block Cipher
	block, err := aes.NewCipher(keyHash)
	if err != nil {
		logrus.Fatalln(err)
	}
	iv := make([]byte, block.BlockSize())
	logrus.Debugf("BlockSize: %#v\n", block.BlockSize())
	msgLen := fi.Size() - int64(len(iv))
	_, err = infile.ReadAt(iv, msgLen)
	if err != nil {
		logrus.Fatalln(err)
	}

	// buffer stream
	buf := make([]byte, bufferSize)
	stream := cipher.NewCTR(block, iv)
	for {
		n, err := infile.Read(buf)
		if n > 0 {
			if n > int(msgLen) {
				n = int(msgLen)
			}
			msgLen -= int64(n)
			stream.XORKeyStream(buf, buf[:n])
			outfile.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Fatalf("Read %d bytes, err: %v", n, err)
			break
		}
	}
}
