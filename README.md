# seriald
Serial Port as TCP Server

## Installation

```bash
go get -u github.com/moisespsena-go/seriald
```
## Usage

### Start Server

```bash
cd $GOPATH/bin
./seriald -h
```

output:

    Usage ./seriald HOST:PORT

    Examples:

    ./seriald :5000
    ./seriald localhost:5000
    ./seriald 0.0.0.0:5000
    
Run:

```bash
./seriald localhost:5000
```
 
### Client

```go
package main

import (
    "net"
    "os"
    "fmt"
)

func main() {
    strEcho := "Halo"
    servAddr := "localhost:5000"
    tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
    if err != nil {
        println("ResolveTCPAddr failed:", err.Error())
        os.Exit(1)
    }

    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    if err != nil {
        println("Dial failed:", err.Error())
        os.Exit(1)
    }
    
    var (
        serialPath     = "/dev/ttyUSB0"
        serialBaudRate = 9600
        cfg            = fmt.Sprintf("%s:%d", serialPath, serialBaudRate)
    )
    
    _, err = conn.Write([]byte(cfg + "\r\n"))
    
    if err != nil {
        println("Write to server failed:", err.Error())
        os.Exit(1)
    }

    println("write to serial = ", strEcho)

    reply := make([]byte, 1024)

    _, err = conn.Read(reply)
    if err != nil {
        println("Write to serial failed:", err.Error())
        os.Exit(1)
    }

    println("reply from serial=", string(reply))

    conn.Close()
}
```
