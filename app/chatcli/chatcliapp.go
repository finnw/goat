//  ---------------------------------------------------------------------------
//
//  chatcliapp.go
//
//  Copyright (c) 2014, Jared Chavez. 
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

package main

// External imports.
import (
	"github.com/xaevman/goat/core/goapp"
)

// Stdlib imports.
import (
	"os"
)

// ConsoleCliStart is a goapp.AppStarter implementation which runs a
// ConsoleCli based ChatCli instance.
type ConsoleCliStart struct {}

// PreInit parses command line arguments to set the target server
// address and start-up username.
func (this *ConsoleCliStart) PreInit() {
	if len(os.Args) > 1 {
		srvAddr = os.Args[1]
	}
	if len(os.Args) > 2 {
		chatCli.username = os.Args[2]
	}
	if len(os.Args) > 3 {
		useUdp = true
	}
}

// PostInit connects to the remote server, closing the application if
// a connection cannot be established.
func (this *ConsoleCliStart) PostInit() {
	var err error

	if useUdp {
		sock, err := proto.ListenUdp("127.0.0.1:8902")
		if err != nil {
			goapp.Stop()
			return
		}

		err = proto.DialUdp(srvAddr, sock)
	} else {
		err = proto.DialTcp(srvAddr)
	}

	if err != nil {
		goapp.Stop()
	}
}
