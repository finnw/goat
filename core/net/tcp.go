//  ---------------------------------------------------------------------------
//
//  tcp.go
//
//  Copyright (c) 2013, Jared Chavez. 
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

package net

// External imports.
import (
	"github.com/xaevman/goat/core/log"
	"github.com/xaevman/goat/lib/lifecycle"
)

// Stdlib imports.
import (
	"net"
	"sync"
	"sync/atomic"
)

// TCP Buffer size for reading data off the line.
const TCP_BUFFER_SIZE_B = 1 * 1024 // KB * B


// TCPCli represents a TCP client object. The client object handles connecting
// to server processes and manages basic IO and synchronization tasks, accepting
// data from, and passing data up to, the net service's registered Protocol 
// objects.
type TCPCli struct {
	discoChan  chan *tcpCon
	id         uint32
	srvMap     map[uint32]*tcpCon
	mutex      sync.RWMutex
	syncObj    *lifecycle.Lifecycle
}

// NewTCPCli is a helper constructor function which returns a pointer to a
// newly intialized TCPCli object for use.
func NewTCPCli() *TCPCli {
	cli := TCPCli {
		discoChan: make(chan *tcpCon, 1),
		id:        atomic.AddUint32(&netId, 1),
		srvMap:    make(map[uint32]*tcpCon, 0),
		syncObj:   lifecycle.New(),
	}

	return &cli
}

// Dial connects the TCPCli object to a new server endpoint. A TCPCli can
// establish multiple connections for redundancy or load balancing.
func (this *TCPCli) Dial(addr string) {
	go func() {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Error("%v", err)
			return
		}

		log.Info("Connected to %v", addr)
		
		tCon := tcpCon{
			discoChan: this.discoChan,
			id:        atomic.AddUint32(&netId, 1),
			readChan:  make(chan []byte, 1),
			socket:    conn.(*net.TCPConn),
			syncObj:   lifecycle.New(),
			writeChan: make(chan []byte, 1),
		}

		this.mutex.Lock()
		this.srvMap[tCon.id] = &tCon
		this.mutex.Unlock()

		onConnect(&tCon)

		tCon.startHandlers()

		for this.syncObj.QueryRun() {
			select {
			case srv := <-this.discoChan:
				onDisconnect(srv)

				this.mutex.Lock()
				delete(this.srvMap, srv.id)
				this.mutex.Unlock()
			case <-this.syncObj.QueryShutdown():
				log.Info("Shutting down %v", addr)
			}
		}

		this.syncObj.ShutdownComplete()
	}()
}

// Shutdown closes all of the TCPCli's existing sockets and shuts down
// the client's go routines.
func (this *TCPCli) Shutdown() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, srv := range this.srvMap {
		srv.Close()
	}

	this.syncObj.Shutdown()
}


// TCPSrv represents a TCP server object. The server object handles basic
// communications, client synchronization, and error handling. Client code
// only establishes a server listener via the server object. Sending and receiving
// messages is done via Protocol objects registered with the net service.
type TCPSrv struct {
	discoChan  chan *tcpCon
	cliMap     map[uint32]*tcpCon
	id         uint32
	listener   net.Listener
	mutex      sync.RWMutex
	syncObj    *lifecycle.Lifecycle
}

// NewTCPSrv is a helper function which initializes a new TCPSrv instance
// and returns a pointer to it for use.
func NewTCPSrv() *TCPSrv {
	srv := TCPSrv {
		discoChan:  make(chan *tcpCon, 1),
		cliMap:     make(map[uint32]*tcpCon, 0),
		id:         atomic.AddUint32(&netId, 1),
		syncObj:    lifecycle.New(),
	}

	return &srv
}

// Start initializes and starts the TCP server in a new goroutine, 
// on the given network address.
func (this *TCPSrv) Start(addr string) {
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("%v", err)
			return
		}

		log.Info("Startup complete %v", addr)

		this.listener = ln
		go this.acceptConnections()

		for this.syncObj.QueryRun() {
			select {
			case cli := <-this.discoChan:
				onDisconnect(cli)

				this.mutex.Lock()
				delete(this.cliMap, cli.id)
				this.mutex.Unlock()
			case <-this.syncObj.QueryShutdown():
				log.Info("Shutting down %v", addr)
			}
		}

		this.syncObj.ShutdownComplete()
	}()
}

// Stop shuts the TCP server down.
func (this *TCPSrv) Stop() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, cli := range this.cliMap {
		cli.Close()
	}

	this.syncObj.Shutdown()
}

// acceptConnections handles accepting connections from new clients.
func (this *TCPSrv) acceptConnections() {
	for {
		cliCon, err := this.listener.Accept()
		if err != nil {
			log.Error("%v", err)
			continue
		}

		cli := tcpCon{
			discoChan: this.discoChan,
			id:        atomic.AddUint32(&netId, 1),
			readChan:  make(chan []byte, 1),
			socket:    cliCon.(*net.TCPConn),
			syncObj:   lifecycle.New(),
			writeChan: make(chan []byte, 1),
		}

		this.mutex.Lock()
		this.cliMap[cli.id] = &cli
		this.mutex.Unlock()

		onConnect(&cli)

		cli.startHandlers()
	}
}


// tcpCon represents a TCP connection.
type tcpCon struct {
	discoChan chan *tcpCon
	id        uint32
	key       string
	nextMsg   *NetMsg
	readChan  chan []byte
	socket    *net.TCPConn
	syncObj   *lifecycle.Lifecycle
	writeChan chan []byte
}

// close shuts down the client TCP connection.
func (this *tcpCon) Close() {
	this.socket.Close()
}

// Id returns the net service id of this tcp connection.
func (this *tcpCon) Id() uint32 {
	return this.id
}

// Key returns the key information assigned to this tcp connection.
func (this *tcpCon) Key() string {
	return this.key
}

// LocalAddr returns the local endpoint address for this tcp connection.
func (this *tcpCon) LocalAddr() net.Addr {
	return this.socket.LocalAddr()
}

// RemoteAddr returns the remote endpoint's address for this tcp connection.
func (this *tcpCon) RemoteAddr() net.Addr {
	return this.socket.RemoteAddr()
}

// buildMsg is called when raw data is received off of the line. This function
// handles the segmentation of messages across multiple receive buffers or
// the packing of multiple messages into a single buffer in the stream.
func (this *tcpCon) buildMsg(msgData []byte) []byte {
	if len(msgData) < 1 {
		return nil
	}

	if this.nextMsg == nil {
		// search for a good header
		if !ValidateMsgHeader(msgData) {
			if len(msgData) > 1 {
				return msgData[1:]
			}

			return nil
		}

		// start msg object for this message
		size        := GetMsgSize(msgData)
		this.nextMsg = &NetMsg {
			con:    this,
			data:   make([]byte, size),
			header: GetMsgHeader(msgData),
		}
	}

	// read available data, up to msg length
	leftovers, complete := this.nextMsg.addData(msgData[4:])
	if complete {
		// dispatch message
		routeMsg(this.nextMsg)
		this.nextMsg = nil
	}

	if leftovers != nil {
		return leftovers
	}

	return nil
}

// handleReads runs in its own goroutine, looping endlessly, reading data
// of of the line.
func (this *tcpCon) handleReads() {
	var count int
	var err   error
	
	buffer := make([]byte, TCP_BUFFER_SIZE_B)

	for {
		count, err = this.socket.Read(buffer)
		// disco
		if count < 1 {
			this.notifyDisco()
			return
		}

		// received data
		this.readChan<- buffer[:count]

		// a real error
		if err != nil {
			log.Error("%v", err)
			continue
		}
	}
}

// handleWrites runs in its own goroutine, looping endlessly, putting
// write events onto the line.
func (this *tcpCon) handleWrites() {
	var count int
	var err   error

	for {
		select {
		case data := <-this.writeChan:
			count, err = this.socket.Write(data)
			// disco
			if count < 1 {
				this.notifyDisco()
				return
			}

			if err != nil {
				log.Error("%v", err)
				continue
			}
		}
	}
}

// runCli runs in its own goroutine, handling read events that bubble up
// from the handleReads goroutine, building message objects, and sending them
// up the pipeline to registered protocols. runCli is also responsible for 
// handling and coordinating shutdown of a tcpCon when signaled.
func (this *tcpCon) runCli() {
	for this.syncObj.QueryRun() {
		select {
		case buffer := <-this.readChan:
			extra := this.buildMsg(buffer)
			for extra != nil {
				extra = this.buildMsg(extra)
			}
		case <-this.syncObj.QueryShutdown():
		}
	}

	this.Close()

	this.syncObj.ShutdownComplete()
}

// notifyDisco bubbles a disco event up to the tcpCon's parent TCPSrv so 
// that it can properly handle the client's disconnection.
func (this *tcpCon) notifyDisco() {
	this.syncObj.Shutdown()

	if this.discoChan != nil {
		this.discoChan<- this
	}
}

// startHandlers starts the 3 goroutines responsible for handling IO and 
// synchronization for this client.
func (this *tcpCon) startHandlers() {
	go this.runCli()
	go this.handleReads()
	go this.handleWrites()
}

// writeData accepts write events and forwards them to the handleWrites goroutine.
func (this *tcpCon) writeData(data []byte) {
	this.writeChan<- data
}
