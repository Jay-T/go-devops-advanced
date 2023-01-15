package helpers

import (
	"context"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc/metadata"
)

// GetReqID helper returns Request-ID from metadata.
func GetReqID(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	return md.Get("Request-ID")[0]
}

// GetLocalInterfaceAddress returns IP address of interface <ifname>.
func GetLocalInterfaceAddress(remoteAddress string) (string, error) {
	var localAddress string
	// iface, err := net.InterfaceByName(ifname)
	// if err != nil {
	// 	log.Fatal("Error while getting local interfaces.", err.Error())
	// }

	// if iface != nil {
	// 	addresses, err := iface.Addrs()
	// 	if err != nil {
	// 		log.Fatal("Error while getting an address from local interface.", err.Error())
	// 	}
	// 	address := addresses[0]
	// 	if ipnet, ok := address.(*net.IPNet); ok {
	// 		localAddress = ipnet.IP.String()
	// 	}
	// }

	conn, err := net.Dial("tcp4", remoteAddress)

	if err != nil {
		log.Println(err)
		return "", err
	}
	localAddress = strings.Split(conn.LocalAddr().String(), ":")[0]

	return localAddress, nil
}
