// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/moisespsena-go/seriald"
	"github.com/moisespsena/go-default-logger"

	"github.com/spf13/cobra"
)

const (
	p_addr = "addr"
	p_exit = "exit"
)

// remoteCmd represents the connect command
var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var (
			exit    bool
			addr    string
			tcpAddr *net.TCPAddr
			conn    net.Conn
		)
		if exit, err = cmd.Flags().GetBool(p_exit); err != nil {
			return fmt.Errorf("Get `exit` flag failed: %v", err)
		}

		if strings.HasPrefix(addr, "unix:") {
			if conn, err = net.Dial("unix", addr[5:]); err != nil {
				return fmt.Errorf("Dial failed: %v", err.Error())
			}
		} else {
			if addr, err = cmd.Flags().GetString(p_addr); err != nil {
				return fmt.Errorf("Get `addr` flag failed: %v", err)
			}

			if tcpAddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
				return fmt.Errorf("ResolveTCPAddr failed: %v", err.Error())
			}

			if conn, err = net.DialTCP("tcp", nil, tcpAddr); err != nil {
				return fmt.Errorf("Dial failed: %v", err.Error())
			}
		}

		log := defaultlogger.NewLogger(addr)
		connRW := &seriald.NamedConn{conn, addr, log}
		copiers := seriald.NewStreamCopiers(
			seriald.NewStreamCopier(connRW, &seriald.NamedWriteClose{os.Stdout, "STDOUT"}),
			seriald.NewStreamCopier(&seriald.NamedReadClose{os.Stdin, "STDIN"}, connRW),
		)

		for _, c := range copiers.Copiers {
			func(c *seriald.CopyStream) {
				c.Closer(func() error {
					copiers.Close()
					return nil
				})
				l := *log
				l.Module = c.String()
				c.Log = &l
			}(c)
		}
		copiers.Log = log
		var errs seriald.Errors
		copiers.StartError(func(err error) {
			errs = errs.Append(err)
		})

		done := make(chan bool)
		copiers.Done = done

		copiers.Start()

		if len(args) != 0 {
			fmt.Fprintf(conn, "%s\n", strings.Join(args, " "))
			if exit {
				fmt.Fprintf(conn, "exit\n")
			}
		}

		<-done
		return errs.GetError()
	},
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	remoteCmd.Flags().StringP(p_addr, "a", "localhost:5000", "The server addr. Example: `localhost:5000` or `unix:/path/conn.sock`.")
	remoteCmd.Flags().BoolP(p_exit, "e", false, "Close connection")
}
